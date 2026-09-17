package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/generation"
	"github.com/lrx0014/ResumeGPT/internal/hunter"
	"github.com/lrx0014/ResumeGPT/internal/identity"
	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/platform/requestcontext"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	"github.com/lrx0014/ResumeGPT/internal/settings"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
	"github.com/lrx0014/ResumeGPT/internal/taskmonitor"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/trace"
)

type Dependencies struct {
	Profiles           *profile.Service
	Jobs               *job.Service
	JobImports         *job.ImportService
	Logger             *slog.Logger
	WebOrigin          string
	Authenticator      identity.Authenticator
	Access             identity.AccessRepository
	DefaultWorkspaceID string
	Blobs              blobstore.Signer
	Documents          *document.Service
	Settings           *settings.Service
	Templates          *resumetemplate.Service
	Generations        *generation.Service
	Tasks              *taskmonitor.Service
	Hunters            *hunter.Service
}

type API struct {
	profiles           *profile.Service
	jobs               *job.Service
	jobImports         *job.ImportService
	logger             *slog.Logger
	webOrigin          string
	authenticator      identity.Authenticator
	access             identity.AccessRepository
	defaultWorkspaceID string
	blobs              blobstore.Signer
	documents          *document.Service
	settings           *settings.Service
	templates          *resumetemplate.Service
	generations        *generation.Service
	tasks              *taskmonitor.Service
	hunters            *hunter.Service
}

func New(deps Dependencies) http.Handler {
	api := &API{
		profiles:           deps.Profiles,
		jobs:               deps.Jobs,
		jobImports:         deps.JobImports,
		logger:             deps.Logger,
		webOrigin:          deps.WebOrigin,
		authenticator:      deps.Authenticator,
		access:             deps.Access,
		defaultWorkspaceID: deps.DefaultWorkspaceID,
		blobs:              deps.Blobs,
		documents:          deps.Documents,
		settings:           deps.Settings,
		templates:          deps.Templates,
		generations:        deps.Generations,
		tasks:              deps.Tasks,
		hunters:            deps.Hunters,
	}

	protected := http.NewServeMux()
	api.registerDocuments(protected)
	api.registerSettings(protected)
	api.registerTemplates(protected)
	api.registerGenerations(protected)
	api.registerTaskMonitor(protected)
	api.registerHunters(protected)
	protected.Handle("GET /v1/system/capabilities", api.requireRole(identity.RoleViewer, api.capabilities))
	protected.Handle("GET /v1/profiles", api.requireRole(identity.RoleViewer, api.listProfiles))
	protected.Handle("POST /v1/profiles", api.requireRole(identity.RoleEditor, api.createProfile))
	protected.Handle("GET /v1/profiles/{profileID}", api.requireRole(identity.RoleViewer, api.getProfile))
	protected.Handle("PUT /v1/profiles/{profileID}", api.requireRole(identity.RoleEditor, api.updateProfile))
	protected.Handle("DELETE /v1/profiles/{profileID}", api.requireRole(identity.RoleEditor, api.deleteProfile))
	protected.Handle("POST /v1/profiles/{profileID}/avatar-upload", api.requireRole(identity.RoleEditor, api.createAvatarUpload))
	protected.Handle("GET /v1/profiles/{profileID}/avatar", api.requireRole(identity.RoleViewer, api.getProfileAvatar))
	protected.Handle("GET /v1/jobs", api.requireRole(identity.RoleViewer, api.listJobs))
	protected.Handle("POST /v1/jobs", api.requireRole(identity.RoleEditor, api.createJob))
	if api.jobImports != nil {
		protected.Handle("POST /v1/jobs/imports", api.requireRole(identity.RoleEditor, api.importJobs))
	}
	protected.Handle("GET /v1/jobs/{jobID}", api.requireRole(identity.RoleViewer, api.getJob))
	protected.Handle("PUT /v1/jobs/{jobID}", api.requireRole(identity.RoleEditor, api.updateJob))
	protected.Handle("DELETE /v1/jobs/{jobID}", api.requireRole(identity.RoleEditor, api.deleteJob))
	protected.Handle("POST /v1/storage/uploads", api.requireRole(identity.RoleEditor, api.createUpload))
	protected.Handle("GET /v1/storage/objects/{objectID}/download", api.requireRole(identity.RoleViewer, api.createDownload))

	root := http.NewServeMux()
	root.HandleFunc("GET /healthz", api.health)
	root.HandleFunc("GET /readyz", api.health)
	root.Handle("/v1/", api.authenticate(protected))
	return otelhttp.NewHandler(api.requestID(api.recoverPanic(api.accessLog(api.cors(root)))), "resumegpt.http")
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) capabilities(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"apiVersion": "v1",
		"features": map[string]bool{
			"documents":        a.documents != nil,
			"profiles":         true,
			"jobs":             true,
			"jobImports":       a.jobImports != nil,
			"generation":       a.generations != nil,
			"rendering":        a.generations != nil,
			"settings":         a.settings != nil,
			"templates":        a.templates != nil,
			"templateUploads":  a.templates != nil && a.templates.UploadsEnabled(),
			"taskMonitor":      a.tasks != nil,
			"jobHunters":       a.hunters != nil,
			"templatePreviews": a.templates != nil && a.templates.PreviewsEnabled(),
		},
	})
}

func (a *API) listProfiles(w http.ResponseWriter, r *http.Request) {
	items, err := a.profiles.List(r.Context(), workspaceID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profiles_list_failed", "Could not load profiles.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) getProfile(w http.ResponseWriter, r *http.Request) {
	item, err := a.profiles.Get(r.Context(), workspaceID(r), r.PathValue("profileID"))
	if errors.Is(err, profile.ErrNotFound) {
		writeError(w, http.StatusNotFound, "profile_not_found", "The requested profile does not exist.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profile_read_failed", "Could not load the profile.")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) createProfile(w http.ResponseWriter, r *http.Request) {
	var input profile.CreateInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return
	}
	item, err := a.profiles.Create(r.Context(), workspaceID(r), input)
	if errors.Is(err, profile.ErrInvalid) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_profile", "Provide a profile name and valid profile fields.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profile_create_failed", "Could not create the profile.")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) updateProfile(w http.ResponseWriter, r *http.Request) {
	var input profile.UpdateInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return
	}
	item, err := a.profiles.Update(r.Context(), workspaceID(r), r.PathValue("profileID"), input)
	if errors.Is(err, profile.ErrInvalid) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_profile", "Provide a profile name and valid profile fields.")
		return
	}
	if errors.Is(err, profile.ErrNotFound) {
		writeError(w, http.StatusNotFound, "profile_not_found", "The requested profile does not exist.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profile_update_failed", "Could not update the profile.")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) deleteProfile(w http.ResponseWriter, r *http.Request) {
	err := a.profiles.Delete(r.Context(), workspaceID(r), r.PathValue("profileID"))
	if errors.Is(err, profile.ErrNotFound) {
		writeError(w, http.StatusNotFound, "profile_not_found", "The requested profile does not exist.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profile_delete_failed", "Could not delete the profile.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listJobs(w http.ResponseWriter, r *http.Request) {
	items, err := a.jobs.List(r.Context(), workspaceID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "jobs_list_failed", "Could not load jobs.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) getJob(w http.ResponseWriter, r *http.Request) {
	item, err := a.jobs.Get(r.Context(), workspaceID(r), r.PathValue("jobID"))
	if errors.Is(err, job.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job_not_found", "The requested job does not exist.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "job_read_failed", "Could not load the job.")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) createJob(w http.ResponseWriter, r *http.Request) {
	var input job.CreateInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return
	}
	item, err := a.jobs.Create(r.Context(), workspaceID(r), input)
	if errors.Is(err, job.ErrInvalidInput) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_job", "Provide a title, company, valid status, and valid field values.")
		return
	}
	if errors.Is(err, job.ErrInvalidURL) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_job_url", "The source URL must be a valid HTTPS URL.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "job_create_failed", "Could not create the job.")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) updateJob(w http.ResponseWriter, r *http.Request) {
	var input job.UpdateInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return
	}
	item, err := a.jobs.Update(r.Context(), workspaceID(r), r.PathValue("jobID"), input)
	switch {
	case errors.Is(err, job.ErrInvalidInput), errors.Is(err, job.ErrInvalidURL):
		writeError(w, http.StatusUnprocessableEntity, "invalid_job", "Provide a title, company, valid status, and valid field values.")
	case errors.Is(err, job.ErrNotFound):
		writeError(w, http.StatusNotFound, "job_not_found", "The requested job does not exist.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "job_update_failed", "Could not update the job.")
	default:
		writeJSON(w, http.StatusOK, item)
	}
}

func (a *API) deleteJob(w http.ResponseWriter, r *http.Request) {
	err := a.jobs.Delete(r.Context(), workspaceID(r), r.PathValue("jobID"))
	if errors.Is(err, job.ErrNotFound) {
		writeError(w, http.StatusNotFound, "job_not_found", "The requested job does not exist.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "job_delete_failed", "Could not delete the job.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) importJobs(w http.ResponseWriter, r *http.Request) {
	var input job.ImportInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return
	}
	items, err := a.jobImports.Create(r.Context(), workspaceID(r), input)
	switch {
	case errors.Is(err, job.ErrInvalidURL):
		writeError(w, http.StatusUnprocessableEntity, "unsupported_job_url", "Use a supported public HTTPS job URL.")
	case errors.Is(err, job.ErrInvalidAIConfig):
		writeError(w, http.StatusUnprocessableEntity, "invalid_ai_import_config", "Select an LLM provider and model for AI-assisted import.")
	case errors.Is(err, job.ErrTooManyURLs):
		writeError(w, http.StatusUnprocessableEntity, "invalid_url_batch", "Provide between 1 and 50 job URLs.")
	case err != nil:
		a.logger.Error("queue job imports", "error", err)
		writeError(w, http.StatusInternalServerError, "job_import_failed", "Could not queue the job import.")
	default:
		writeJSON(w, http.StatusAccepted, map[string]any{"items": items})
	}
}

func workspaceID(r *http.Request) string {
	return requestcontext.WorkspaceID(r.Context())
}

func (a *API) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = id.New("req")
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(requestcontext.WithRequestID(r.Context(), requestID)))
	})
}

func (a *API) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.webOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", a.webOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Idempotency-Key, X-Request-ID, X-Workspace-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *API) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		a.logger.Info("HTTP request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(started),
			"request_id", requestcontext.RequestID(r.Context()),
			"trace_id", trace.SpanContextFromContext(r.Context()).TraceID().String(),
		)
	})
}

func (a *API) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recovered := recover(); recovered != nil {
				a.logger.Error("panic recovered", "value", recovered)
				writeError(w, http.StatusInternalServerError, "internal_error", "An unexpected error occurred.")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
