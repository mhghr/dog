package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"monitoring-platform/internal/domain"
	"monitoring-platform/internal/ingestion"
)

// workerAuth guards internal endpoints with the shared worker bearer token.
func (h *Handler) workerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		token, found := strings.CutPrefix(header, "Bearer ")

		expected := h.deps.Config.WorkerToken
		if !found || expected == "" ||
			subtle.ConstantTimeCompare([]byte(token), []byte(expected)) != 1 {
			writeError(w, r, http.StatusUnauthorized, "unauthorized", "A valid worker token is required", nil)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *Handler) ingestResult(w http.ResponseWriter, r *http.Request) {
	var result domain.ProbeResult
	if err := decodeJSON(r, &result); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	inserted, err := h.deps.Ingestion.Ingest(r.Context(), &result)
	if err != nil {
		var validationErr *ingestion.ValidationError
		switch {
		case isValidationError(err, &validationErr):
			writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", validationErr.Message, nil)
		default:
			h.deps.Logger.Error("ingest result failed", "job_id", result.JobID, "error", err)
			writeDomainError(w, r, err)
		}
		return
	}

	if !inserted {
		writeJSON(w, http.StatusOK, map[string]string{"status": "duplicate"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "accepted"})
}

func isValidationError(err error, target **ingestion.ValidationError) bool {
	validationErr, ok := err.(*ingestion.ValidationError)
	if ok {
		*target = validationErr
	}
	return ok
}
