package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
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

	// SNMP collector: encrypt community/v3 secrets before they are stored so
	// they never appear in plaintext in the database or API responses.
	if h.monitorTypeIs(r.Context(), input.MonitorTypeID, domain.MonitorSNMP) {
		// Enforce tenant/target isolation: the SNMP target must be the
		// resource's own target (public network device), never an arbitrary
		// internal address reachable from the collector.
		if err := h.validateSnmpTarget(r.Context(), resourceID, monitor.Configuration); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_target", err.Error(), nil)
			return
		}
		h.ensureSNMPCredential(r.Context(), resourceID, monitor.Configuration)
		h.encryptSNMPConfigSecrets(monitor.Configuration, h.deps.Config.AgentSecretEncryptionKey)
	}

	if err := h.deps.MonitorRepo.Create(r.Context(), monitor); err != nil {
		h.deps.Logger.Error("create resource monitor failed", "error", err)
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, monitor)
}

// monitorTypeIs reports whether the given monitor type resolves to the given
// executor key.
func (h *Handler) monitorTypeIs(ctx context.Context, monitorTypeID string, executorKey domain.MonitorType) bool {
	if monitorTypeID == "" {
		return false
	}
	var key string
	if err := h.deps.Pool.QueryRow(ctx, `
		SELECT executor_key FROM monitor_types WHERE id = $1::uuid`, monitorTypeID).Scan(&key); err != nil {
		return false
	}
	return domain.MonitorType(key) == executorKey
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

	monitor, ok := h.applyMonitorUpdate(w, r, monitor, input)
	if !ok {
		return
	}
	if err := h.deps.MonitorRepo.Update(r.Context(), &monitor); err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, monitor)
}

// applyMonitorUpdate merges the update input onto an existing monitor, handling
// SNMP secret re-encryption when the configuration changes. Returns false after
// writing an error response when the update is rejected.
func (h *Handler) applyMonitorUpdate(w http.ResponseWriter, r *http.Request, monitor domain.Monitor, input struct {
	Name            string         `json:"name"`
	Enabled         *bool          `json:"enabled"`
	Configuration   map[string]any `json:"configuration"`
	Severity        string         `json:"severity"`
	IntervalSeconds *int           `json:"interval_seconds"`
	TimeoutMillis   *int           `json:"timeout_millis"`
	Retries         *int           `json:"retries"`
}) (domain.Monitor, bool) {
	if input.Name != "" {
		monitor.Name = input.Name
	}
	if input.Enabled != nil {
		monitor.Enabled = *input.Enabled
	}
	if input.Configuration != nil {
		monitor.Configuration = input.Configuration
		// SNMP collector: re-encrypt secrets on update (unchanged values are
		// masked by the UI and preserved).
		if monitor.Type == domain.MonitorSNMP {
			if err := h.validateSnmpTarget(r.Context(), monitor.ResourceID, monitor.Configuration); err != nil {
				writeError(w, r, http.StatusBadRequest, "invalid_target", err.Error(), nil)
				return monitor, false
			}
			h.ensureSNMPCredential(r.Context(), monitor.ResourceID, monitor.Configuration)
			h.encryptSNMPConfigSecrets(monitor.Configuration, h.deps.Config.AgentSecretEncryptionKey)
		}
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
	return monitor, true
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
	from, to, ok := parseMetricsRange(w, r, query)
	if !ok {
		return
	}
	stepSeconds := resolveStep(query.Get("step"), to.Sub(from))
	probeID := query.Get("probe_id")

	metricKey := query.Get("metric")
	series, err := h.querySeriesByMetric(r.Context(), monitorID, metricKey, from, to, stepSeconds)
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

	lastSuccessAt, err := h.deps.MetricQuery.LatestSuccessAt(r.Context(), monitorID)
	if err != nil {
		h.deps.Logger.Error("query latest success failed", "monitor_id", monitorID, "error", err)
		lastSuccessAt = nil
	}

	statusCodes, err := h.deps.MetricQuery.StatusCodeDistribution(r.Context(), monitorID, from, to)
	if err != nil {
		h.deps.Logger.Error("query status code distribution failed", "monitor_id", monitorID, "error", err)
		statusCodes = nil
	}

	// Range-scoped KPIs (aggregate or per-probe), computed in the metric layer
	// so the frontend consumes calculated values instead of raw samples.
	aggregate, err := h.deps.MetricQuery.AggregateMetrics(r.Context(), monitorID, probeID, from, to)
	if err != nil {
		h.deps.Logger.Error("query aggregate metrics failed", "monitor_id", monitorID, "error", err)
		aggregate = domain.MonitorAggregateMetrics{}
	}

	probeMetrics, err := h.deps.MetricQuery.ProbeMetrics(r.Context(), monitorID, from, to)
	if err != nil {
		h.deps.Logger.Error("query probe metrics failed", "monitor_id", monitorID, "error", err)
		probeMetrics = []domain.ProbeAggregateMetrics{}
	}

	// Attach last-status facts to each probe row from the latest result set.
	attachLastStatusToProbes(probeMetrics, latest)

	// Chart drill-down: when `at` is present, return the result closest to
	// that timestamp so the frontend can render its timing waterfall.
	selected := h.resolveSelectedResult(r, monitorID, probeID)

	writeJSON(w, http.StatusOK, map[string]any{
		"series":          series,
		"latest":          latest,
		"series_limit":    maxChartSeries,
		"step_seconds":    stepSeconds,
		"from":            from,
		"to":              to,
		"metric_key":      metricKey,
		"probe_id":        probeID,
		"monitor_type":    string(monitor.Type),
		"last_success_at": lastSuccessAt,
		"status_codes":    statusCodes,
		"aggregate":       aggregate,
		"probes":          probeMetrics,
		"selected":        selected,
	})
}

// parseMetricsRange resolves the from/to window from query params, defaulting
// to the trailing 24 hours. Returns false after writing an error response when
// the range is invalid.
func parseMetricsRange(w http.ResponseWriter, r *http.Request, query url.Values) (time.Time, time.Time, bool) {
	to := time.Now().UTC()
	if raw := query.Get("to"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_range", "to must be RFC3339", nil)
			return time.Time{}, time.Time{}, false
		}
		to = parsed
	}

	from := to.Add(-24 * time.Hour)
	if raw := query.Get("from"); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_range", "from must be RFC3339", nil)
			return time.Time{}, time.Time{}, false
		}
		from = parsed
	}

	if !from.Before(to) {
		writeError(w, r, http.StatusBadRequest, "invalid_range", "from must be before to", nil)
		return time.Time{}, time.Time{}, false
	}
	return from, to, true
}

// querySeriesByMetric routes the per-probe series query to the metric layer
// based on the requested metric key.
func (h *Handler) querySeriesByMetric(ctx context.Context, monitorID, metricKey string, from, to time.Time, stepSeconds int) ([]domain.ProbeSeries, error) {
	switch metricKey {
	case "status":
		return h.deps.MetricQuery.StatusSeriesByProbe(ctx, monitorID, from, to, stepSeconds, maxChartSeries)
	case "":
		return h.deps.MetricQuery.SeriesByProbe(ctx, monitorID, from, to, stepSeconds, maxChartSeries)
	default:
		return h.deps.MetricQuery.SeriesByProbeMetric(ctx, monitorID, metricKey, from, to, stepSeconds, maxChartSeries)
	}
}

// attachLastStatusToProbes decorates each probe aggregate with the latest
// result's status facts so the frontend can show per-probe up/down state.
func attachLastStatusToProbes(probeMetrics []domain.ProbeAggregateMetrics, latest []domain.ProbeResult) {
	latestByProbe := make(map[string]domain.ProbeResult, len(latest))
	for _, res := range latest {
		if res.ProbeLocationID != "" {
			latestByProbe[res.ProbeLocationID] = res
		}
	}
	for i := range probeMetrics {
		latestRes, ok := latestByProbe[probeMetrics[i].ProbeID]
		if !ok {
			continue
		}
		probeMetrics[i].LastStatusCode = attributeInt(latestRes.Attributes["status_code"])
		probeMetrics[i].LastSuccess = latestRes.Success
		at := latestRes.StartedAt
		probeMetrics[i].LastCheckedAt = &at
	}
}

// resolveSelectedResult returns the probe result closest to `at` for the
// drill-down waterfall, or nil when absent or invalid.
func (h *Handler) resolveSelectedResult(r *http.Request, monitorID, probeID string) *domain.ProbeResult {
	raw := r.URL.Query().Get("at")
	if raw == "" {
		return nil
	}
	at, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil
	}
	res, err := h.deps.Results.ResultAt(r.Context(), monitorID, probeID, at)
	if err != nil {
		return nil
	}
	return &res
}

// attributeInt converts a JSON-decoded status_code attribute into *int, or nil
// when absent/unparseable (transport failures have no HTTP status code).
func attributeInt(raw any) *int {
	switch v := raw.(type) {
	case float64:
		value := int(v)
		return &value
	case float32:
		value := int(v)
		return &value
	case int:
		return &v
	case int64:
		value := int(v)
		return &value
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return &parsed
		}
	}
	return nil
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
