package httpapi

import (
	"errors"
	"net/http"

	"github.com/lrx0014/ResumeGPT/internal/identity"
	"github.com/lrx0014/ResumeGPT/internal/settings"
)

func (a *API) registerSettings(mux *http.ServeMux) {
	if a.settings == nil {
		return
	}
	mux.Handle("GET /v1/settings", a.requireRole(identity.RoleViewer, a.getSettings))
	mux.Handle("PUT /v1/settings", a.requireRole(identity.RoleOwner, a.updateSettings))
	mux.Handle("GET /v1/settings/llm-connections", a.requireRole(identity.RoleViewer, a.listLLMConnections))
	mux.Handle("POST /v1/settings/llm-connections", a.requireRole(identity.RoleOwner, a.createLLMConnection))
	mux.Handle("PUT /v1/settings/llm-connections/{connectionID}", a.requireRole(identity.RoleOwner, a.updateLLMConnection))
	mux.Handle("DELETE /v1/settings/llm-connections/{connectionID}", a.requireRole(identity.RoleOwner, a.deleteLLMConnection))
	mux.Handle("POST /v1/settings/llm-connections/{connectionID}/test", a.requireRole(identity.RoleOwner, a.testLLMConnection))
}

func (a *API) getSettings(w http.ResponseWriter, r *http.Request) {
	value, err := a.settings.GetPreferences(r.Context(), workspaceID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "settings_read_failed", "Could not load settings.")
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (a *API) updateSettings(w http.ResponseWriter, r *http.Request) {
	var input settings.PreferencesInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return
	}
	value, err := a.settings.SavePreferences(r.Context(), workspaceID(r), input)
	if errors.Is(err, settings.ErrInvalid) {
		writeError(w, http.StatusUnprocessableEntity, "invalid_settings", "Choose a supported interface language and theme.")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "settings_update_failed", "Could not save settings.")
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (a *API) listLLMConnections(w http.ResponseWriter, r *http.Request) {
	items, err := a.settings.ListConnections(r.Context(), workspaceID(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "llm_connections_read_failed", "Could not load LLM connections.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (a *API) createLLMConnection(w http.ResponseWriter, r *http.Request) {
	var input settings.LLMConnectionInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return
	}
	value, err := a.settings.CreateConnection(r.Context(), workspaceID(r), input)
	if handleSettingsMutationError(w, err, "create") {
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (a *API) updateLLMConnection(w http.ResponseWriter, r *http.Request) {
	var input settings.LLMConnectionInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return
	}
	value, err := a.settings.UpdateConnection(r.Context(), workspaceID(r), r.PathValue("connectionID"), input)
	if handleSettingsMutationError(w, err, "update") {
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (a *API) deleteLLMConnection(w http.ResponseWriter, r *http.Request) {
	err := a.settings.DeleteConnection(r.Context(), workspaceID(r), r.PathValue("connectionID"))
	if handleSettingsMutationError(w, err, "delete") {
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) testLLMConnection(w http.ResponseWriter, r *http.Request) {
	result, err := a.settings.TestConnection(r.Context(), workspaceID(r), r.PathValue("connectionID"))
	switch {
	case errors.Is(err, settings.ErrNotFound):
		writeError(w, http.StatusNotFound, "llm_connection_not_found", "The requested LLM connection does not exist.")
	case errors.Is(err, settings.ErrConnectionFailed):
		writeError(w, http.StatusBadGateway, "llm_connection_test_failed", err.Error())
	case err != nil:
		writeError(w, http.StatusInternalServerError, "llm_connection_test_failed", "Could not test the LLM connection.")
	default:
		writeJSON(w, http.StatusOK, result)
	}
}

func handleSettingsMutationError(w http.ResponseWriter, err error, operation string) bool {
	switch {
	case errors.Is(err, settings.ErrInvalid):
		writeError(w, http.StatusUnprocessableEntity, "invalid_llm_connection", "Provide a valid name, mode, provider, and Base URL.")
	case errors.Is(err, settings.ErrNotFound):
		writeError(w, http.StatusNotFound, "llm_connection_not_found", "The requested LLM connection does not exist.")
	case err != nil:
		writeError(w, http.StatusInternalServerError, "llm_connection_"+operation+"_failed", "Could not "+operation+" the LLM connection.")
	default:
		return false
	}
	return true
}
