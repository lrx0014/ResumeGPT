package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/lrx0014/ResumeGPT/internal/generation"
	"github.com/lrx0014/ResumeGPT/internal/identity"
)

func (a *API) registerGenerations(mux *http.ServeMux) {
	if a.generations == nil {
		return
	}
	mux.Handle("GET /v1/generations", a.requireRole(identity.RoleViewer, a.listGenerations))
	mux.Handle("POST /v1/generations", a.requireRole(identity.RoleEditor, a.createGeneration))
	mux.Handle("GET /v1/generations/{generationID}", a.requireRole(identity.RoleViewer, a.getGeneration))
	mux.Handle("POST /v1/generations/{generationID}/retry", a.requireRole(identity.RoleEditor, a.retryGeneration))
	mux.Handle("GET /v1/generations/{generationID}/artifact", a.requireRole(identity.RoleViewer, a.generationArtifact))
}
func (a *API) listGenerations(w http.ResponseWriter, r *http.Request) {
	items, err := a.generations.List(r.Context(), workspaceID(r))
	if err != nil {
		writeError(w, 500, "generations_list_failed", "Could not load generations.")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (a *API) createGeneration(w http.ResponseWriter, r *http.Request) {
	var input generation.CreateInput
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid_request", "The request body is not valid JSON.")
		return
	}
	item, err := a.generations.Create(r.Context(), workspaceID(r), input)
	switch {
	case errors.Is(err, generation.ErrInvalid):
		writeError(w, 422, "invalid_generation", "Choose valid inputs and models for this generation.")
	case errors.Is(err, generation.ErrState):
		writeError(w, 409, "generation_input_not_ready", "Choose a saved profile, an Opportunity with a description, a ready matching LaTeX template, and available LLM connections.")
	case err != nil:
		a.logger.Error("create generation", "error", err)
		writeError(w, 500, "generation_create_failed", "Could not queue the generation.")
	default:
		writeJSON(w, 202, item)
	}
}
func (a *API) getGeneration(w http.ResponseWriter, r *http.Request) {
	item, err := a.generations.Get(r.Context(), workspaceID(r), r.PathValue("generationID"))
	if errors.Is(err, generation.ErrNotFound) {
		writeError(w, 404, "generation_not_found", "The generation does not exist.")
		return
	}
	if err != nil {
		writeError(w, 500, "generation_read_failed", "Could not load the generation.")
		return
	}
	writeJSON(w, 200, item)
}
func (a *API) retryGeneration(w http.ResponseWriter, r *http.Request) {
	item, err := a.generations.Retry(r.Context(), workspaceID(r), r.PathValue("generationID"))
	switch {
	case errors.Is(err, generation.ErrNotFound):
		writeError(w, 404, "generation_not_found", "The generation does not exist.")
	case errors.Is(err, generation.ErrState):
		writeError(w, 409, "generation_not_failed", "Only a failed generation can be retried.")
	case err != nil:
		a.logger.Error("retry generation", "error", err)
		writeError(w, 500, "generation_retry_failed", "Could not retry the generation.")
	default:
		writeJSON(w, 202, item)
	}
}
func (a *API) generationArtifact(w http.ResponseWriter, r *http.Request) {
	object, err := a.generations.Artifact(r.Context(), workspaceID(r), r.PathValue("generationID"))
	switch {
	case errors.Is(err, generation.ErrNotFound):
		writeError(w, 404, "generation_not_found", "The generation does not exist.")
	case errors.Is(err, generation.ErrState):
		writeError(w, 409, "generation_not_ready", "The generated PDF is not ready.")
	case err != nil:
		writeError(w, 500, "artifact_read_failed", "Could not load the generated PDF.")
	default:
		defer object.Body.Close()
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename=%q`, "generated-application.pdf"))
		w.WriteHeader(200)
		_, _ = io.Copy(w, object.Body)
	}
}
