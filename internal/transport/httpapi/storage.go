package httpapi

import (
	"net/http"
	"strings"
	"time"

	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
)

type createUploadRequest struct {
	ContentType string `json:"contentType"`
}

// presignTTL is how long a presigned upload/download URL remains valid.
const presignTTL = 15 * time.Minute

func (a *API) createUpload(w http.ResponseWriter, r *http.Request) {
	var input createUploadRequest
	if !decodeOrBadRequest(w, r, &input) {
		return
	}
	if strings.TrimSpace(input.ContentType) == "" {
		writeError(w, http.StatusUnprocessableEntity, "content_type_required", "Content type is required.")
		return
	}
	result, err := a.blobs.PresignUpload(r.Context(), workspaceID(r), blobstore.NewObjectID(), input.ContentType, presignTTL)
	if err != nil {
		a.logger.Error("presign upload", "error", err)
		writeError(w, http.StatusInternalServerError, "upload_signing_failed", "Could not create an upload URL.")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (a *API) createDownload(w http.ResponseWriter, r *http.Request) {
	if a.documents != nil {
		allowed, err := a.documents.DownloadAllowed(r.Context(), workspaceID(r), r.PathValue("objectID"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "download_authorization_failed", "Could not authorize the object download.")
			return
		}
		if !allowed {
			writeError(w, http.StatusLocked, "document_quarantined", "The document remains quarantined until scanning and extraction succeed.")
			return
		}
	}
	result, err := a.blobs.PresignDownload(r.Context(), workspaceID(r), r.PathValue("objectID"), presignTTL)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_object_id", "The object identifier is invalid.")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
