package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/job"
	"github.com/lrx0014/ResumeGPT/internal/profile"
	"github.com/lrx0014/ResumeGPT/internal/shared/id"
)

type Dependencies struct {
	Profiles  *profile.Service
	Jobs      *job.Service
	Logger    *slog.Logger
	WebOrigin string
}

type API struct {
	profiles  *profile.Service
	jobs      *job.Service
	logger    *slog.Logger
	webOrigin string
}

func New(deps Dependencies) http.Handler {
	api := &API{
		profiles:  deps.Profiles,
		jobs:      deps.Jobs,
		logger:    deps.Logger,
		webOrigin: deps.WebOrigin,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", api.health)
	mux.HandleFunc("GET /readyz", api.health)
	mux.HandleFunc("GET /v1/system/capabilities", api.capabilities)
	mux.HandleFunc("GET /v1/profiles", api.listProfiles)
	mux.HandleFunc("POST /v1/profiles", api.createProfile)
	mux.HandleFunc("GET /v1/profiles/{profileID}", api.getProfile)
	mux.HandleFunc("GET /v1/jobs", api.listJobs)
	mux.HandleFunc("POST /v1/jobs", api.createJob)
	mux.HandleFunc("GET /v1/jobs/{jobID}", api.getJob)

	return api.requestID(api.recoverPanic(api.accessLog(api.cors(mux))))
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *API) capabilities(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"apiVersion": "v1",
		"features": map[string]bool{
			"profiles":   true,
			"jobs":       true,
			"generation": false,
			"rendering":  false,
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
	if errors.Is(err, profile.ErrInvalidName) {
		writeError(w, http.StatusUnprocessableEntity, "profile_name_required", err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profile_create_failed", "Could not create the profile.")
		return
	}
	writeJSON(w, http.StatusCreated, item)
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
		writeError(w, http.StatusUnprocessableEntity, "job_fields_required", err.Error())
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "job_create_failed", "Could not create the job.")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func workspaceID(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Workspace-ID")); value != "" {
		return value
	}
	return "ws_personal_dev"
}

type contextKey string

const requestIDKey contextKey = "request-id"

func (a *API) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = id.New("req")
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, requestID)))
	})
}

func (a *API) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.webOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", a.webOrigin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Idempotency-Key, X-Request-ID, X-Workspace-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
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
			"request_id", r.Context().Value(requestIDKey),
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
