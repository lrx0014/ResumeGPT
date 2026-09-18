package httpapi

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/lrx0014/ResumeGPT/internal/identity"
	resumetemplate "github.com/lrx0014/ResumeGPT/internal/template"
)

func (a *API) registerTemplates(mux *http.ServeMux) {
	if a.templates == nil {
		return
	}
	mux.Handle("GET /v1/templates", a.requireRole(identity.RoleViewer, a.listTemplates))
	mux.Handle("POST /v1/templates/uploads", a.requireRole(identity.RoleEditor, a.stageTemplate))
	mux.Handle("POST /v1/templates/{templateID}/complete", a.requireRole(identity.RoleEditor, a.queueTemplate))
	mux.Handle("POST /v1/templates/{templateID}/source-upload", a.requireRole(identity.RoleEditor, a.replaceTemplateSource))
	mux.Handle("GET /v1/templates/{templateID}", a.requireRole(identity.RoleViewer, a.getTemplate))
	mux.Handle("PUT /v1/templates/{templateID}", a.requireRole(identity.RoleEditor, a.updateTemplate))
	mux.Handle("DELETE /v1/templates/{templateID}", a.requireRole(identity.RoleEditor, a.deleteTemplate))
	mux.Handle("GET /v1/templates/{templateID}/file", a.requireRole(identity.RoleViewer, a.downloadTemplate))
	mux.Handle("GET /v1/templates/{templateID}/preview", a.requireRole(identity.RoleViewer, a.previewTemplate))
}

func (a *API) replaceTemplateSource(w http.ResponseWriter, r *http.Request) {
	var input resumetemplate.SourceInput
	if !decodeOrBadRequest(w, r, &input) {
		return
	}
	result, err := a.templates.ReplaceSource(r.Context(), workspaceID(r), r.PathValue("templateID"), input)
	switch {
	case errors.Is(err, resumetemplate.ErrInvalid):
		writeError(w, http.StatusUnprocessableEntity, "invalid_template_source", "Choose a TeX, LaTeX ZIP, DOC, or DOCX file and provide a valid optional ZIP entry file.")
	case errors.Is(err, resumetemplate.ErrBuiltIn):
		writeError(w, http.StatusConflict, "built_in_template", "The built-in template source cannot be replaced.")
	case errors.Is(err, resumetemplate.ErrNotFound):
		writeError(w, http.StatusNotFound, "template_not_found", "The requested template does not exist.")
	case errors.Is(err, resumetemplate.ErrState):
		writeError(w, http.StatusServiceUnavailable, "template_uploads_unavailable", "Template uploads are unavailable in this deployment.")
	case err != nil:
		a.logger.Error("replace template source", "error", err)
		writeError(w, http.StatusInternalServerError, "template_source_staging_failed", "Could not stage the replacement template source.")
	default:
		writeJSON(w, http.StatusCreated, result)
	}
}

func (a *API) listTemplates(w http.ResponseWriter, r *http.Request) {
	items, err := a.templates.List(r.Context(), workspaceID(r))
	if err != nil {
		a.logger.Error("list templates", "error", err)
		writeError(w, http.StatusInternalServerError, "templates_list_failed", "Could not load templates.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
func (a *API) getTemplate(w http.ResponseWriter, r *http.Request) {
	item, err := a.templates.Get(r.Context(), workspaceID(r), r.PathValue("templateID"))
	if errors.Is(err, resumetemplate.ErrNotFound) {
		writeError(w, http.StatusNotFound, "template_not_found", "The requested template does not exist.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "template_read_failed", "Could not load the template.")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) stageTemplate(w http.ResponseWriter, r *http.Request) {
	var input resumetemplate.StageInput
	if !decodeOrBadRequest(w, r, &input) {
		return
	}
	result, err := a.templates.Stage(r.Context(), workspaceID(r), input)
	switch {
	case errors.Is(err, resumetemplate.ErrInvalid):
		writeError(w, http.StatusUnprocessableEntity, "invalid_template", "Choose a TeX, LaTeX ZIP, DOC, or DOCX file and provide a name, template type, and valid optional ZIP entry file.")
	case errors.Is(err, resumetemplate.ErrState):
		writeError(w, http.StatusServiceUnavailable, "template_uploads_unavailable", "Template uploads are unavailable in this deployment.")
	case err != nil:
		a.logger.Error("stage template upload", "error", err)
		writeError(w, http.StatusInternalServerError, "template_staging_failed", "Could not stage the template upload.")
	default:
		writeJSON(w, http.StatusCreated, result)
	}
}
func (a *API) queueTemplate(w http.ResponseWriter, r *http.Request) {
	item, err := a.templates.Queue(r.Context(), workspaceID(r), r.PathValue("templateID"))
	switch {
	case errors.Is(err, resumetemplate.ErrNotFound):
		writeError(w, http.StatusNotFound, "template_not_found", "The staged template does not exist.")
	case errors.Is(err, resumetemplate.ErrState), errors.Is(err, resumetemplate.ErrBuiltIn):
		writeError(w, http.StatusConflict, "template_state_conflict", "This template cannot be submitted.")
	case err != nil:
		a.logger.Error("queue template extraction", "error", err)
		writeError(w, http.StatusInternalServerError, "template_queue_failed", "Could not queue template extraction.")
	default:
		writeJSON(w, http.StatusAccepted, item)
	}
}
func (a *API) updateTemplate(w http.ResponseWriter, r *http.Request) {
	var input resumetemplate.UpdateInput
	if !decodeOrBadRequest(w, r, &input) {
		return
	}
	item, err := a.templates.Update(r.Context(), workspaceID(r), r.PathValue("templateID"), input)
	switch {
	case errors.Is(err, resumetemplate.ErrInvalid):
		writeError(w, http.StatusUnprocessableEntity, "invalid_template", "Provide a name, résumé or cover-letter type, and a valid description.")
	case errors.Is(err, resumetemplate.ErrBuiltIn):
		writeError(w, http.StatusConflict, "built_in_template", "Built-in templates cannot be edited.")
	case errors.Is(err, resumetemplate.ErrNotFound):
		writeError(w, http.StatusNotFound, "template_not_found", "The requested template does not exist.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "template_update_failed", "Could not update the template.")
	default:
		writeJSON(w, http.StatusOK, item)
	}
}
func (a *API) deleteTemplate(w http.ResponseWriter, r *http.Request) {
	err := a.templates.Delete(r.Context(), workspaceID(r), r.PathValue("templateID"))
	switch {
	case errors.Is(err, resumetemplate.ErrBuiltIn):
		writeError(w, http.StatusConflict, "built_in_template", "Built-in templates cannot be deleted.")
	case errors.Is(err, resumetemplate.ErrNotFound):
		writeError(w, http.StatusNotFound, "template_not_found", "The requested template does not exist.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "template_delete_failed", "Could not delete the template.")
	default:
		w.WriteHeader(204)
	}
}
func (a *API) downloadTemplate(w http.ResponseWriter, r *http.Request) {
	source, signed, err := a.templates.Download(r.Context(), workspaceID(r), r.PathValue("templateID"))
	switch {
	case errors.Is(err, resumetemplate.ErrNotFound):
		writeError(w, http.StatusNotFound, "template_not_found", "The requested template does not exist.")
	case errors.Is(err, resumetemplate.ErrState):
		writeError(w, http.StatusConflict, "template_not_ready", "The template file is not ready for download.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "template_download_failed", "Could not prepare the template download.")
	case source != nil:
		w.Header().Set("Content-Type", "application/x-tex")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, "rezume.tex"))
		w.WriteHeader(200)
		_, _ = w.Write(source)
	default:
		http.Redirect(w, r, signed.URL, http.StatusTemporaryRedirect)
	}
}

func (a *API) previewTemplate(w http.ResponseWriter, r *http.Request) {
	preview, err := a.templates.Preview(r.Context(), workspaceID(r), r.PathValue("templateID"))
	switch {
	case errors.Is(err, resumetemplate.ErrNotFound):
		writeError(w, http.StatusNotFound, "template_not_found", "The requested template does not exist.")
	case errors.Is(err, resumetemplate.ErrState):
		writeError(w, http.StatusConflict, "template_preview_unavailable", "A PDF preview is not available for this template.")
	case err != nil:
		a.logger.Warn("load template preview", "template_id", r.PathValue("templateID"), "error", err)
		writeError(w, http.StatusInternalServerError, "template_preview_failed", "Could not load the stored PDF preview.")
	default:
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Cache-Control", "private, max-age=300")
		w.Header().Set("Content-Disposition", `inline; filename="template-preview.pdf"`)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(preview)
	}
}
