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
	mux.Handle("PUT /v1/generations/{generationID}", a.requireRole(identity.RoleEditor, a.reconfigureGeneration))
	mux.Handle("DELETE /v1/generations/{generationID}", a.requireRole(identity.RoleEditor, a.deleteGeneration))
	mux.Handle("GET /v1/generations/{generationID}/steps", a.requireRole(identity.RoleViewer, a.generationSteps))
	mux.Handle("GET /v1/generations/{generationID}/steps/{stepID}/artifact", a.requireRole(identity.RoleViewer, a.generationStepArtifact))
	mux.Handle("POST /v1/generations/{generationID}/retry", a.requireRole(identity.RoleEditor, a.retryGeneration))
	mux.Handle("POST /v1/generations/{generationID}/revisions", a.requireRole(identity.RoleEditor, a.reviseGeneration))
	mux.Handle("GET /v1/generations/{generationID}/artifact", a.requireRole(identity.RoleViewer, a.generationArtifact))
}
func (a *API) reconfigureGeneration(w http.ResponseWriter, r *http.Request) {
	var input generation.CreateInput
	if decodeJSON(w, r, &input) != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return
	}
	item, err := a.generations.Reconfigure(r.Context(), workspaceID(r), r.PathValue("generationID"), input)
	switch {
	case errors.Is(err, generation.ErrInvalid):
		writeError(w, http.StatusUnprocessableEntity, "invalid_generation", "Choose valid inputs and models for this application.")
	case errors.Is(err, generation.ErrNotFound):
		writeError(w, http.StatusNotFound, "generation_not_found", "The generation does not exist.")
	case errors.Is(err, generation.ErrState):
		writeError(w, http.StatusConflict, "generation_active", "Wait until the current generation finishes before changing its configuration.")
	case err != nil:
		a.logger.Error("reconfigure generation", "error", err)
		writeError(w, http.StatusInternalServerError, "generation_reconfigure_failed", "Could not update and regenerate the application.")
	default:
		writeJSON(w, http.StatusAccepted, item)
	}
}
func (a *API) deleteGeneration(w http.ResponseWriter, r *http.Request) {
	err := a.generations.Delete(r.Context(), workspaceID(r), r.PathValue("generationID"))
	switch {
	case errors.Is(err, generation.ErrNotFound):
		writeError(w, http.StatusNotFound, "generation_not_found", "The generation does not exist.")
	case err != nil:
		a.logger.Error("delete generation", "error", err)
		writeError(w, http.StatusInternalServerError, "generation_delete_failed", "Could not delete the generation.")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}
func (a *API) generationSteps(w http.ResponseWriter, r *http.Request) {
	items, err := a.generations.Steps(r.Context(), workspaceID(r), r.PathValue("generationID"))
	if errors.Is(err, generation.ErrNotFound) {
		writeError(w, 404, "generation_not_found", "The generation does not exist.")
		return
	}
	if err != nil {
		writeError(w, 500, "generation_steps_failed", "Could not load the generation timeline.")
		return
	}
	writeJSON(w, 200, map[string]any{"items": items})
}
func (a *API) reviseGeneration(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Prompt string `json:"prompt"`
	}
	if decodeJSON(w, r, &input) != nil {
		writeError(w, 400, "invalid_request", "The request body is not valid JSON.")
		return
	}
	item, err := a.generations.Revise(r.Context(), workspaceID(r), r.PathValue("generationID"), input.Prompt)
	switch {
	case errors.Is(err, generation.ErrInvalid):
		writeError(w, 422, "invalid_revision_prompt", "Enter a revision instruction of up to 4,000 characters.")
	case errors.Is(err, generation.ErrNotFound):
		writeError(w, 404, "generation_not_found", "The generation does not exist.")
	case errors.Is(err, generation.ErrState):
		writeError(w, 409, "generation_not_ready", "Wait until the current generation is ready before requesting a revision.")
	case err != nil:
		a.logger.Error("revise generation", "error", err)
		writeError(w, 500, "generation_revision_failed", "Could not queue the requested revision.")
	default:
		writeJSON(w, 202, item)
	}
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
		writeError(w, 409, "generation_input_not_ready", "Choose a saved profile, an Opportunity with a description, a ready matching LaTeX template, and available LLM providers.")
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
func (a *API) generationStepArtifact(w http.ResponseWriter, r *http.Request) {
	object, err := a.generations.StepArtifact(r.Context(), workspaceID(r), r.PathValue("generationID"), r.PathValue("stepID"))
	switch {
	case errors.Is(err, generation.ErrNotFound):
		writeError(w, 404, "generation_step_not_found", "The generation step does not exist.")
	case errors.Is(err, generation.ErrState):
		writeError(w, 409, "generation_step_has_no_pdf", "This generation step does not contain a PDF.")
	case err != nil:
		writeError(w, 500, "step_artifact_read_failed", "Could not load the generated stage PDF.")
	default:
		defer object.Body.Close()
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename=%q`, "generation-stage.pdf"))
		w.WriteHeader(200)
		_, _ = io.Copy(w, object.Body)
	}
}
