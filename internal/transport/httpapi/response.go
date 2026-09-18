package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type errorResponse struct {
	Error apiError `json:"error"`
}

type apiError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Retryable bool   `json:"retryable"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		slog.Error("encode HTTP response", "error", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: apiError{
		Code:      code,
		Message:   message,
		Retryable: status >= http.StatusInternalServerError,
	}})
}

func decodeJSON(w http.ResponseWriter, r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

// decodeOrBadRequest decodes the request body into target, writing a 400
// invalid_request response and returning false on failure.
func decodeOrBadRequest(w http.ResponseWriter, r *http.Request, target any) bool {
	if err := decodeJSON(w, r, target); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "The request body is not valid JSON.")
		return false
	}
	return true
}
