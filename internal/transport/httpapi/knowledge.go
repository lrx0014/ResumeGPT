package httpapi

import (
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/lrx0014/ResumeGPT/internal/identity"
	"github.com/lrx0014/ResumeGPT/internal/knowledge"
)

func (a *API) registerKnowledge(mux *http.ServeMux) {
	routes := []struct {
		pattern string
		role    identity.Role
		handler http.HandlerFunc
	}{
		{"GET /v1/profiles/{profileID}/knowledge", identity.RoleViewer, a.getKnowledge},
		{"GET /v1/profiles/{profileID}/knowledge/export", identity.RoleViewer, a.exportKnowledge},
		{"GET /v1/profiles/{profileID}/knowledge/search", identity.RoleViewer, a.searchKnowledge},
		{"POST /v1/profiles/{profileID}/sources", identity.RoleEditor, a.importKnowledge},
		{"POST /v1/profiles/{profileID}/sources/upload", identity.RoleEditor, a.uploadKnowledge},
		{"POST /v1/profiles/{profileID}/facts/{factID}/reviews", identity.RoleEditor, a.reviewKnowledge},
		{"DELETE /v1/profiles/{profileID}/sources/{sourceID}", identity.RoleEditor, a.deleteKnowledgeSource},
	}
	for _, route := range routes {
		mux.Handle(route.pattern, a.requireRole(route.role, func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			if a.knowledge == nil {
				writeError(w, 503, "knowledge_unavailable", "Profile knowledge requires PostgreSQL persistence. Start the database and select PostgreSQL mode.")
				return
			}
			route.handler(w, r)
		}))
	}
}

func (a *API) knowledgeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, knowledge.ErrInvalid):
		writeError(w, 422, "invalid_knowledge", "Use valid UTF-8 text: maximum 64 KiB, 200 non-empty lines, 4,000 bytes per line, and a name up to 200 bytes. Reviews require a valid status and version.")
	case errors.Is(err, knowledge.ErrNotFound):
		writeError(w, 404, "knowledge_not_found", "The profile or knowledge resource does not exist.")
	case errors.Is(err, knowledge.ErrConflict):
		writeError(w, 409, "version_conflict", "This fact has changed. Reload its latest version before reviewing.")
	default:
		// Database errors can contain source values; do not include them in application logs.
		a.logger.Error("knowledge operation failed")
		writeError(w, 500, "knowledge_failed", "Could not complete the knowledge operation.")
	}
}

func (a *API) getKnowledge(w http.ResponseWriter, r *http.Request) {
	result, err := a.knowledge.Snapshot(r.Context(), workspaceID(r), r.PathValue("profileID"))
	if err != nil {
		a.knowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) exportKnowledge(w http.ResponseWriter, r *http.Request) {
	result, err := a.knowledge.Snapshot(r.Context(), workspaceID(r), r.PathValue("profileID"))
	if err != nil {
		a.knowledgeError(w, err)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="profile-knowledge.json"`)
	writeJSON(w, 200, result)
}

func (a *API) importKnowledge(w http.ResponseWriter, r *http.Request) {
	var input knowledge.Import
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, 400, "invalid_request", "Expected a JSON object containing name and text.")
		return
	}
	a.saveKnowledge(w, r, input)
}

func (a *API) uploadKnowledge(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, knowledge.MaxSourceBytes+8192)
	if err := r.ParseMultipartForm(knowledge.MaxSourceBytes + 8192); err != nil {
		writeError(w, 400, "invalid_upload", "Upload one UTF-8 TXT file of at most 64 KiB.")
		return
	}
	defer r.MultipartForm.RemoveAll()
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, 400, "file_required", "The file field is required.")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, knowledge.MaxSourceBytes+1))
	if err != nil || len(data) > knowledge.MaxSourceBytes {
		writeError(w, 413, "file_too_large", "The file must not exceed 64 KiB.")
		return
	}
	if strings.ToLower(filepath.Ext(header.Filename)) != ".txt" || !strings.HasPrefix(http.DetectContentType(data), "text/plain") {
		writeError(w, 415, "unsupported_source", "Only plain-text UTF-8 TXT files are supported in this release. PDF, Word, TeX and image parsing are planned.")
		return
	}
	a.saveKnowledge(w, r, knowledge.Import{Name: header.Filename, Text: string(data)})
}

func (a *API) saveKnowledge(w http.ResponseWriter, r *http.Request, input knowledge.Import) {
	result, err := a.knowledge.Import(r.Context(), workspaceID(r), r.PathValue("profileID"), input)
	if err != nil {
		a.knowledgeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (a *API) reviewKnowledge(w http.ResponseWriter, r *http.Request) {
	var input knowledge.Review
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, 400, "invalid_request", "Expected a fact review object.")
		return
	}
	result, err := a.knowledge.Review(r.Context(), workspaceID(r), r.PathValue("profileID"), r.PathValue("factID"), input)
	if err != nil {
		a.knowledgeError(w, err)
		return
	}
	writeJSON(w, 201, result)
}

func (a *API) deleteKnowledgeSource(w http.ResponseWriter, r *http.Request) {
	if err := a.knowledge.DeleteSource(r.Context(), workspaceID(r), r.PathValue("profileID"), r.PathValue("sourceID")); err != nil {
		a.knowledgeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]bool{"deleted": true})
}

func (a *API) searchKnowledge(w http.ResponseWriter, r *http.Request) {
	result, err := a.knowledge.Search(r.Context(), workspaceID(r), r.PathValue("profileID"), r.URL.Query().Get("q"))
	if err != nil {
		a.knowledgeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"items": result})
}
