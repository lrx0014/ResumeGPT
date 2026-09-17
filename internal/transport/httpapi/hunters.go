package httpapi

import (
	"errors"
	"net/http"

	"github.com/lrx0014/ResumeGPT/internal/hunter"
	"github.com/lrx0014/ResumeGPT/internal/identity"
)

func (a *API) registerHunters(mux *http.ServeMux) {
	if a.hunters == nil {
		return
	}
	mux.Handle("GET /v1/job-hunters", a.requireRole(identity.RoleViewer, a.listHunters))
	mux.Handle("POST /v1/job-hunters", a.requireRole(identity.RoleEditor, a.createHunter))
	mux.Handle("GET /v1/job-hunters/{hunterID}", a.requireRole(identity.RoleViewer, a.getHunter))
	mux.Handle("PUT /v1/job-hunters/{hunterID}", a.requireRole(identity.RoleEditor, a.updateHunter))
	mux.Handle("DELETE /v1/job-hunters/{hunterID}", a.requireRole(identity.RoleEditor, a.deleteHunter))
	mux.Handle("POST /v1/job-hunters/{hunterID}/run", a.requireRole(identity.RoleEditor, a.runHunter))
	mux.Handle("GET /v1/job-hunters/{hunterID}/review-items", a.requireRole(identity.RoleViewer, a.listHunterReviewItems))
	mux.Handle("DELETE /v1/job-hunters/{hunterID}/review-items/{reviewID}", a.requireRole(identity.RoleEditor, a.dismissHunterReviewItem))
}

func (a *API) listHunterReviewItems(w http.ResponseWriter, r *http.Request) {
	items, err := a.hunters.ListReviewItems(r.Context(), workspaceID(r), r.PathValue("hunterID"))
	if errors.Is(err, hunter.ErrNotFound) {
		writeError(w, http.StatusNotFound, "hunter_not_found", "The requested Job Hunter does not exist.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hunter_review_list_failed", "Could not load jobs that need confirmation.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) dismissHunterReviewItem(w http.ResponseWriter, r *http.Request) {
	err := a.hunters.DismissReviewItem(r.Context(), workspaceID(r), r.PathValue("hunterID"), r.PathValue("reviewID"))
	if errors.Is(err, hunter.ErrNotFound) {
		writeError(w, http.StatusNotFound, "hunter_review_not_found", "The job awaiting confirmation does not exist.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hunter_review_dismiss_failed", "Could not dismiss the job awaiting confirmation.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) listHunters(w http.ResponseWriter, r *http.Request) {
	items, err := a.hunters.List(r.Context(), workspaceID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hunters_list_failed", "Could not load Job Hunters.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) getHunter(w http.ResponseWriter, r *http.Request) {
	item, err := a.hunters.Get(r.Context(), workspaceID(r), r.PathValue("hunterID"))
	if errors.Is(err, hunter.ErrNotFound) {
		writeError(w, http.StatusNotFound, "hunter_not_found", "The requested Job Hunter does not exist.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hunter_read_failed", "Could not load the Job Hunter.")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (a *API) createHunter(w http.ResponseWriter, r *http.Request) {
	var input hunter.SaveInput
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return
	}
	item, err := a.hunters.Create(r.Context(), workspaceID(r), input)
	if errors.Is(err, hunter.ErrInvalid) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_hunter", "Provide valid search criteria, Profile, result limit, schedule, LLM provider, and model.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hunter_create_failed", "Could not create the Job Hunter.")
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (a *API) updateHunter(w http.ResponseWriter, r *http.Request) {
	var input hunter.SaveInput
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return
	}
	item, err := a.hunters.Update(r.Context(), workspaceID(r), r.PathValue("hunterID"), input)
	switch {
	case errors.Is(err, hunter.ErrInvalid):
		writeError(w, http.StatusUnprocessableEntity, "invalid_hunter", "Provide valid search criteria, Profile, result limit, schedule, LLM provider, and model.")
	case errors.Is(err, hunter.ErrNotFound):
		writeError(w, http.StatusNotFound, "hunter_not_found", "The requested Job Hunter does not exist.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "hunter_update_failed", "Could not update the Job Hunter.")
	default:
		writeJSON(w, http.StatusOK, item)
	}
}

func (a *API) deleteHunter(w http.ResponseWriter, r *http.Request) {
	err := a.hunters.Delete(r.Context(), workspaceID(r), r.PathValue("hunterID"))
	if errors.Is(err, hunter.ErrNotFound) {
		writeError(w, http.StatusNotFound, "hunter_not_found", "The requested Job Hunter does not exist.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "hunter_delete_failed", "Could not delete the Job Hunter.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) runHunter(w http.ResponseWriter, r *http.Request) {
	err := a.hunters.RunNow(r.Context(), workspaceID(r), r.PathValue("hunterID"))
	switch {
	case errors.Is(err, hunter.ErrNotFound):
		writeError(w, http.StatusNotFound, "hunter_not_found", "The requested Job Hunter does not exist.")
	case errors.Is(err, hunter.ErrConflict):
		writeError(w, http.StatusConflict, "hunter_already_running", "This Job Hunter already has an active run.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "hunter_run_failed", "Could not queue the Job Hunter.")
	default:
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
	}
}
