package api

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/ingestion"
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

	events := h.deps.AlertEngine.Evaluate(r.Context(), result)
	for _, evt := range events {
		h.deps.Notifier.Dispatch(r.Context(), evt, evt.ChannelIDs)
	}

	writeJSON(w, http.StatusCreated, map[string]string{"status": "accepted"})
}

func (h *Handler) ingestResultBatch(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Results []*domain.ProbeResult `json:"results"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	stored := make([]string, 0, len(req.Results))
	for _, result := range req.Results {
		inserted, err := h.deps.Ingestion.Ingest(r.Context(), result)
		if err != nil {
			h.deps.Logger.Error("batch ingest result failed", "job_id", result.JobID, "error", err)
			continue
		}
		if inserted {
			events := h.deps.AlertEngine.Evaluate(r.Context(), *result)
			for _, evt := range events {
				h.deps.Notifier.Dispatch(r.Context(), evt, evt.ChannelIDs)
			}
			stored = append(stored, result.ID)
		}
	}

	writeJSON(w, http.StatusCreated, map[string]any{"stored": stored})
}

func isValidationError(err error, target **ingestion.ValidationError) bool {
	validationErr, ok := err.(*ingestion.ValidationError)
	if ok {
		*target = validationErr
	}
	return ok
}
