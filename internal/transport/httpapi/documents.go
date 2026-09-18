package httpapi

import (
	"errors"
	"net/http"

	"github.com/lrx0014/ResumeGPT/internal/document"
	"github.com/lrx0014/ResumeGPT/internal/identity"
)

func (a *API) registerDocuments(mux *http.ServeMux) {
	if a.documents == nil {
		return
	}
	mux.Handle("POST /v1/profiles/{profileID}/document-uploads", a.requireRole(identity.RoleEditor, a.stageDocument))
	mux.Handle("POST /v1/profiles/{profileID}/document-uploads/{uploadID}/complete", a.requireRole(identity.RoleEditor, a.queueDocument))
	mux.Handle("GET /v1/profiles/{profileID}/document-uploads/{uploadID}", a.requireRole(identity.RoleViewer, a.getDocumentUpload))
}

func (a *API) stageDocument(w http.ResponseWriter, r *http.Request) {
	var input document.StageInput
	if !decodeOrBadRequest(w, r, &input) {
		return
	}
	result, err := a.documents.Stage(r.Context(), workspaceID(r), r.PathValue("profileID"), input)
	switch {
	case errors.Is(err, document.ErrInvalid):
		writeError(w, http.StatusUnprocessableEntity, "unsupported_document", "Choose a PDF, DOC, DOCX, TeX, Markdown, TXT, PNG, JPG, or JPEG file and provide its content type.")
	case errors.Is(err, document.ErrNotFound):
		writeError(w, http.StatusNotFound, "profile_not_found", "The requested profile does not exist.")
	case err != nil:
		a.logger.Error("stage document upload", "error", err)
		writeError(w, http.StatusInternalServerError, "document_staging_failed", "Could not stage the document upload.")
	default:
		writeJSON(w, http.StatusCreated, result)
	}
}

func (a *API) queueDocument(w http.ResponseWriter, r *http.Request) {
	result, err := a.documents.Queue(r.Context(), workspaceID(r), r.PathValue("profileID"), r.PathValue("uploadID"))
	switch {
	case errors.Is(err, document.ErrNotFound):
		writeError(w, http.StatusNotFound, "upload_not_found", "The staged document upload does not exist.")
	case errors.Is(err, document.ErrState):
		writeError(w, http.StatusConflict, "upload_state_conflict", "The staged document has already been submitted.")
	case err != nil:
		a.logger.Error("queue document extraction", "error", err)
		writeError(w, http.StatusInternalServerError, "document_queue_failed", "Could not queue document extraction.")
	default:
		writeJSON(w, http.StatusAccepted, result)
	}
}

func (a *API) getDocumentUpload(w http.ResponseWriter, r *http.Request) {
	result, err := a.documents.Get(r.Context(), workspaceID(r), r.PathValue("profileID"), r.PathValue("uploadID"))
	if errors.Is(err, document.ErrNotFound) {
		writeError(w, http.StatusNotFound, "upload_not_found", "The staged document upload does not exist.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "upload_read_failed", "Could not read document processing status.")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
