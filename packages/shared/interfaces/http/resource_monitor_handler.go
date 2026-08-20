package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"monitoring-platform/packages/shared/domain"
)

func (h *Handler) listResourceMonitors(w http.ResponseWriter, r *http.Request) {
	resourceID := chi.URLParam(r, "resourceID")
	if _, err := uuid.Parse(resourceID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "resource id must be a valid UUID", nil)
		return
	}

	// Tenant isolation: the resource must belong to the session organization.
	if !h.resourceBelongsToOrg(w, r, resourceID) {
		return
	}

	monitors, err := h.deps.MonitorRepo.ListByResource(r.Context(), resourceID)
	if err != nil {
		h.deps.Logger.Error("list resource monitors failed", "error", err)
		writeDomainError(w, r, err)
		return
	}
	if monitors == nil {
		monitors = []domain.Monitor{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": monitors})
}

func (h *Handler) createResourceMonitor(w http.ResponseWriter, r *http.Request) {
	resourceID := chi.URLParam(r, "resourceID")
	if _, err := uuid.Parse(resourceID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "resource id must be a valid UUID", nil)
		return
	}

	// Tenant isolation: a monitor may only be created under a resource the
	// session organization owns.
	if !h.resourceBelongsToOrg(w, r, resourceID) {
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

	monitor := &domain.Monitor{
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
	if err := h.deps.MonitorRepo.Create(r.Context(), monitor); err != nil {
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

	// Tenant isolation: the monitor's parent resource must belong to the
	// session organization. Resolving through the resource repo (org-owned)
	// prevents cross-tenant ID enumeration.
	if !h.monitorBelongsToOrg(w, r, monitorID) {
		return
	}

	monitor, err := h.deps.MonitorRepo.GetByID(r.Context(), monitorID)
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

	if !h.monitorBelongsToOrg(w, r, monitorID) {
		return
	}

	monitor, err := h.deps.MonitorRepo.GetByID(r.Context(), monitorID)
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
	if err := h.deps.MonitorRepo.Update(r.Context(), &monitor); err != nil {
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

	if !h.monitorBelongsToOrg(w, r, monitorID) {
		return
	}

	if err := h.deps.MonitorRepo.Delete(r.Context(), monitorID); err != nil {
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

	if !h.monitorBelongsToOrg(w, r, monitorID) {
		return
	}

	limit := 1
	results, _, err := h.deps.MonitorRepo.ListResults(r.Context(), monitorID, limit, 0)
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

func (h *Handler) resourceMonitorMetrics(w http.ResponseWriter, r *http.Request) {
	monitorID := chi.URLParam(r, "monitorID")
	if _, err := uuid.Parse(monitorID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "monitor id must be a valid UUID", nil)
		return
	}

	if !h.monitorBelongsToOrg(w, r, monitorID) {
		return
	}

	monitor, err := h.deps.MonitorRepo.GetByID(r.Context(), monitorID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	query := r.URL.Query()

	to := time.Now().UTC()
	if raw := query.Get("to"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_range", "to must be RFC3339", nil)
			return
		}
		to = parsed
	}

	from := to.Add(-24 * time.Hour)
	if raw := query.Get("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_range", "from must be RFC3339", nil)
			return
		}
		from = parsed
	}

	if !from.Before(to) {
		writeError(w, r, http.StatusBadRequest, "invalid_range", "from must be before to", nil)
		return
	}

	stepSeconds := resolveStep(query.Get("step"), to.Sub(from))

	metricKey := query.Get("metric")
	var series []domain.ProbeSeries
	switch metricKey {
	case "status":
		series, err = h.deps.Results.StatusSeriesByProbe(r.Context(), monitorID, from, to, stepSeconds, maxChartSeries)
	default:
		if metricKey != "" {
			series, err = h.deps.Results.SeriesByProbeMetric(r.Context(), monitorID, metricKey, from, to, stepSeconds, maxChartSeries)
		} else {
			series, err = h.deps.Results.SeriesByProbe(r.Context(), monitorID, from, to, stepSeconds, maxChartSeries)
		}
	}
	if err != nil {
		h.deps.Logger.Error("query per-probe series failed", "monitor_id", monitorID, "error", err)
		writeDomainError(w, r, err)
		return
	}

	// A detail page must remain bounded even when a monitor is assigned to many
	// probes. The UI renders a representative recent sample; full probe lists
	// belong to a dedicated, paginated endpoint.
	latest, err := h.deps.Results.LatestResultsByProbe(r.Context(), monitorID, maxChartSeries)
	if err != nil {
		h.deps.Logger.Error("query latest results by probe failed", "monitor_id", monitorID, "error", err)
		writeDomainError(w, r, err)
		return
	}

	if series == nil {
		series = []domain.ProbeSeries{}
	}
	if latest == nil {
		latest = []domain.ProbeResult{}
	}

	lastSuccessAt, err := h.deps.Results.LatestSuccessAt(r.Context(), monitorID)
	if err != nil {
		h.deps.Logger.Error("query latest success failed", "monitor_id", monitorID, "error", err)
		lastSuccessAt = nil
	}

	statusCodes, err := h.deps.Results.StatusCodeDistribution(r.Context(), monitorID, from, to)
	if err != nil {
		h.deps.Logger.Error("query status code distribution failed", "monitor_id", monitorID, "error", err)
		statusCodes = nil
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"series":          series,
		"latest":          latest,
		"series_limit":    maxChartSeries,
		"step_seconds":    stepSeconds,
		"from":            from,
		"to":              to,
		"metric_key":      metricKey,
		"monitor_type":    string(monitor.Type),
		"last_success_at": lastSuccessAt,
		"status_codes":    statusCodes,
	})
}

const (
	maxChartPoints = 1500
	maxChartSeries = 25
)

// resourceBelongsToOrg verifies that the given resource belongs to the
// session organization. Writes a 403/404 and returns false otherwise.
func (h *Handler) resourceBelongsToOrg(w http.ResponseWriter, r *http.Request, resourceID string) bool {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return false
	}

	res, err := h.deps.ResourceRepo.GetByID(r.Context(), resourceID)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "resource not found", nil)
		return false
	}
	if res.OrganizationID != orgID {
		writeError(w, r, http.StatusNotFound, "not_found", "resource not found", nil)
		return false
	}
	return true
}

// monitorBelongsToOrg resolves the monitor's parent resource and verifies it
// belongs to the session organization. Prevents cross-tenant ID enumeration
// on monitor, result and metric endpoints.
func (h *Handler) monitorBelongsToOrg(w http.ResponseWriter, r *http.Request, monitorID string) bool {
	orgID, ok := domain.OrgIDFromContext(r.Context())
	if !ok {
		writeError(w, r, http.StatusForbidden, "forbidden", "No organization in session", nil)
		return false
	}

	monitor, err := h.deps.MonitorRepo.GetByID(r.Context(), monitorID)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "monitor not found", nil)
		return false
	}

	if monitor.ResourceID == "" {
		writeError(w, r, http.StatusNotFound, "not_found", "monitor not found", nil)
		return false
	}

	res, err := h.deps.ResourceRepo.GetByID(r.Context(), monitor.ResourceID)
	if err != nil {
		writeError(w, r, http.StatusNotFound, "not_found", "monitor not found", nil)
		return false
	}
	if res.OrganizationID != orgID {
		writeError(w, r, http.StatusNotFound, "not_found", "monitor not found", nil)
		return false
	}
	return true
}

// resolveStep implements the spec downsampling table with a hard cap on the
// number of returned points.
func resolveStep(raw string, window time.Duration) int {
	var step time.Duration

	if raw != "" && raw != "auto" {
		if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
			step = parsed
		}
	}

	if step == 0 {
		switch {
		case window <= 15*time.Minute:
			step = 5 * time.Second
		case window <= time.Hour:
			step = 15 * time.Second
		case window <= 6*time.Hour:
			step = time.Minute
		case window <= 24*time.Hour:
			step = 5 * time.Minute
		case window <= 7*24*time.Hour:
			step = 30 * time.Minute
		case window <= 30*24*time.Hour:
			step = 2 * time.Hour
		default:
			step = 24 * time.Hour
		}
	}

	if minStep := window / maxChartPoints; step < minStep {
		step = minStep.Round(time.Second)
		if step < time.Second {
			step = time.Second
		}
	}

	return int(step.Seconds())
}
