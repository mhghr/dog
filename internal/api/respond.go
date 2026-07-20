package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"

	"monitoring-platform/internal/domain"
)

type errorBody struct {
	Code      string              `json:"code"`
	Message   string              `json:"message"`
	Fields    map[string][]string `json:"fields,omitempty"`
	RequestID string              `json:"request_id,omitempty"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if payload != nil {
		_ = json.NewEncoder(w).Encode(payload)
	}
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string, fields map[string][]string) {
	writeJSON(w, status, errorEnvelope{
		Error: errorBody{
			Code:      code,
			Message:   message,
			Fields:    fields,
			RequestID: middleware.GetReqID(r.Context()),
		},
	})
}

func writeDomainError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "The requested resource was not found", nil)
	case errors.Is(err, domain.ErrDuplicate):
		writeError(w, r, http.StatusConflict, "duplicate", "A resource with the same unique value already exists", nil)
	default:
		writeError(w, r, http.StatusInternalServerError, "internal_error", "An unexpected error occurred", nil)
	}
}
