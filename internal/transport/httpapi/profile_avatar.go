package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/lrx0014/ResumeGPT/internal/platform/blobstore"
	"github.com/lrx0014/ResumeGPT/internal/profile"
)

type avatarUploadRequest struct {
	ContentType string `json:"contentType"`
}

func (a *API) createAvatarUpload(w http.ResponseWriter, r *http.Request) {
	var input avatarUploadRequest
	if !decodeOrBadRequest(w, r, &input) {
		return
	}
	if input.ContentType != "image/jpeg" && input.ContentType != "image/png" {
		writeError(w, http.StatusUnprocessableEntity, "unsupported_avatar", "Avatar images must be JPEG or PNG.")
		return
	}
	if _, err := a.profiles.Get(r.Context(), workspaceID(r), r.PathValue("profileID")); errors.Is(err, profile.ErrNotFound) {
		writeError(w, http.StatusNotFound, "profile_not_found", "The requested profile does not exist.")
		return
	} else if err != nil {
		writeError(w, http.StatusInternalServerError, "profile_read_failed", "Could not load the profile.")
		return
	}
	result, err := a.blobs.PresignUpload(r.Context(), workspaceID(r), blobstore.NewObjectID(), input.ContentType, presignTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "avatar_upload_failed", "Could not create an avatar upload URL.")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (a *API) getProfileAvatar(w http.ResponseWriter, r *http.Request) {
	item, err := a.profiles.Get(r.Context(), workspaceID(r), r.PathValue("profileID"))
	if errors.Is(err, profile.ErrNotFound) || (err == nil && strings.TrimSpace(item.AvatarObjectID) == "") {
		writeError(w, http.StatusNotFound, "avatar_not_found", "The profile does not have an avatar.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "profile_read_failed", "Could not load the profile.")
		return
	}
	result, err := a.blobs.PresignDownload(r.Context(), workspaceID(r), item.AvatarObjectID, presignTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "avatar_download_failed", "Could not create an avatar URL.")
		return
	}
	writeJSON(w, http.StatusOK, result)
}
