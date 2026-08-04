package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"monitoring-platform/internal/domain"
)

func (h *Handler) listResourceMonitors(w http.ResponseWriter, r *http.Request) {
	resourceID := chi.URLParam(r, "resourceID")
	if _, err := uuid.Parse(resourceID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "resource id must be a valid UUID", nil)
		return
	}

	monitors, err := h.deps.MonitorV2Repo.ListByResource(r.Context(), resourceID)
	if err != nil {
		h.deps.Logger.Error("list resource monitors failed", "error", err)
		writeDomainError(w, r, err)
		return
	}
	if monitors == nil {
		monitors = []domain.MonitorV2{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": monitors})
}

func (h *Handler) createResourceMonitor(w http.ResponseWriter, r *http.Request) {
	resourceID := chi.URLParam(r, "resourceID")
	if _, err := uuid.Parse(resourceID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "resource id must be a valid UUID", nil)
		return
	}

	var input struct {
		MonitorTypeID   string         `json:"monitor_type_id"`
		Name            string         `json:"name"`
		Enabled         bool           `json:"enabled"`
		Configuration   map[string]any `json:"configuration"`
		Severity        string         `json:"severity"`
		IntervalSeconds int            `json:"interval_seconds"`
		TimeoutMillis   int            `json:"timeout_millis"`
		Retries         int            `json:"retries"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid body", nil)
		return
	}

	monitor := &domain.MonitorV2{
		ResourceID:      resourceID,
		MonitorTypeID:   input.MonitorTypeID,
		Name:            input.Name,
		Enabled:         input.Enabled,
		Configuration:   input.Configuration,
		Severity:        input.Severity,
		IntervalSeconds: input.IntervalSeconds,
		TimeoutMillis:   input.TimeoutMillis,
		Retries:         input.Retries,
		LastStatus:      domain.StatusUnknown,
	}
	if err := h.deps.MonitorV2Repo.Create(r.Context(), monitor); err != nil {
		h.deps.Logger.Error("create resource monitor failed", "error", err)
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, monitor)
}

func (h *Handler) getResourceMonitor(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "monitorID")
	if _, err := uuid.Parse(monitorID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "monitor id must be a valid UUID", nil)
		return
	}

	monitor, err := h.deps.MonitorV2Repo.GetByID(r.Context(), monitorID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, monitor)
}

func (h *Handler) updateResourceMonitor(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "monitorID")
	if _, err := uuid.Parse(monitorID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "monitor id must be a valid UUID", nil)
		return
	}

	monitor, err := h.deps.MonitorV2Repo.GetByID(r.Context(), monitorID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	var input struct {
		Name            string         `json:"name"`
		Enabled         *bool          `json:"enabled"`
		Configuration   map[string]any `json:"configuration"`
		Severity        string         `json:"severity"`
		IntervalSeconds *int           `json:"interval_seconds"`
		TimeoutMillis   *int           `json:"timeout_millis"`
		Retries         *int           `json:"retries"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "invalid body", nil)
		return
	}

	if input.Name != "" {
		monitor.Name = input.Name
	}
	if input.Enabled != nil {
		monitor.Enabled = *input.Enabled
	}
	if input.Configuration != nil {
		monitor.Configuration = input.Configuration
	}
	if input.Severity != "" {
		monitor.Severity = input.Severity
	}
	if input.IntervalSeconds != nil {
		monitor.IntervalSeconds = *input.IntervalSeconds
	}
	if input.TimeoutMillis != nil {
		monitor.TimeoutMillis = *input.TimeoutMillis
	}
	if input.Retries != nil {
		monitor.Retries = *input.Retries
	}
	if err := h.deps.MonitorV2Repo.Update(r.Context(), &monitor); err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, monitor)
}

func (h *Handler) deleteResourceMonitor(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "monitorID")
	if _, err := uuid.Parse(monitorID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "monitor id must be a valid UUID", nil)
		return
	}

	if err := h.deps.MonitorV2Repo.Delete(r.Context(), monitorID); err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listResourceMonitorResults(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "monitorID")
	if _, err := uuid.Parse(monitorID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "monitor id must be a valid UUID", nil)
		return
	}

	limit := 1
	results, _, err := h.deps.MonitorV2Repo.ListV2Results(r.Context(), monitorID, limit, 0)
	if err != nil {
		h.deps.Logger.Error("list monitor results failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	items := make([]map[string]any, 0, len(results))
	for _, result := range results {
		items = append(items, map[string]any{
			"id":              result.ID,
			"monitor_id":      result.MonitorID,
			"status":          result.Status,
			"success":         result.Success,
			"duration_millis": result.DurationMillis,
			"metrics":         result.Metrics,
			"attributes":      result.Attributes,
			"started_at":      result.StartedAt.Format(time.RFC3339),
			"finished_at":     result.FinishedAt.Format(time.RFC3339),
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}
