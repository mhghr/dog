package api

import (
	"encoding/json"
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"monitoring-platform/internal/domain"
)

type monitorResponse struct {
	ID              string                    `json:"id"`
	Name            string                    `json:"name"`
	Type            domain.MonitorType        `json:"type"`
	Target          string                    `json:"target"`
	IntervalSeconds int                       `json:"interval_seconds"`
	TimeoutMillis   int                       `json:"timeout_millis"`
	Retries         int                       `json:"retries"`
	Enabled         bool                      `json:"enabled"`
	Config          map[string]any            `json:"config"`
	LastStatus      domain.MonitorStatus      `json:"last_status"`
	LastCheckedAt   *time.Time                `json:"last_checked_at"`
	NextRunAt       time.Time                 `json:"next_run_at"`
	CreatedAt       time.Time                 `json:"created_at"`
	UpdatedAt       time.Time                 `json:"updated_at"`
	LastResult      *domain.LastResultSummary `json:"last_result,omitempty"`
}

func toMonitorResponse(monitor domain.Monitor, lastResult *domain.LastResultSummary) monitorResponse {
	return monitorResponse{
		ID:              monitor.ID,
		Name:            monitor.Name,
		Type:            monitor.Type,
		Target:          monitor.Target,
		IntervalSeconds: monitor.IntervalSeconds,
		TimeoutMillis:   monitor.TimeoutMillis,
		Retries:         monitor.Retries,
		Enabled:         monitor.Enabled,
		Config:          monitor.Config,
		LastStatus:      monitor.LastStatus,
		LastCheckedAt:   monitor.LastCheckedAt,
		NextRunAt:       monitor.NextRunAt,
		CreatedAt:       monitor.CreatedAt,
		UpdatedAt:       monitor.UpdatedAt,
		LastResult:      lastResult,
	}
}

type paginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

func (h *Handler) createMonitor(w http.ResponseWriter, r *http.Request) {
	var input domain.MonitorInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	monitor, fieldErrors := domain.ValidateMonitorInput(input)
	if len(fieldErrors) > 0 {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "The request contains invalid fields", fieldErrors)
		return
	}

	if err := h.deps.Monitors.Create(r.Context(), &monitor); err != nil {
		h.deps.Logger.Error("create monitor failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, toMonitorResponse(monitor, nil))
}

func (h *Handler) listMonitors(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()

	filter := domain.MonitorListFilter{
		Page:     parsePositiveInt(query.Get("page"), 1, 1000000),
		PageSize: parsePositiveInt(query.Get("page_size"), 20, 100),
		Search:   strings.TrimSpace(query.Get("search")),
		Sort:     query.Get("sort"),
		Order:    query.Get("order"),
	}

	if rawType := query.Get("type"); rawType != "" {
		monitorType, ok := domain.ParseMonitorType(rawType)
		if !ok {
			writeError(w, r, http.StatusBadRequest, "invalid_filter", "Unknown monitor type filter", nil)
			return
		}
		filter.Type = &monitorType
	}

	if rawStatus := query.Get("status"); rawStatus != "" {
		status, ok := domain.ParseMonitorStatus(rawStatus)
		if !ok {
			writeError(w, r, http.StatusBadRequest, "invalid_filter", "Unknown monitor status filter", nil)
			return
		}
		filter.Status = &status
	}

	monitors, total, err := h.deps.Monitors.List(r.Context(), filter)
	if err != nil {
		h.deps.Logger.Error("list monitors failed", "error", err)
		writeDomainError(w, r, err)
		return
	}

	items := make([]monitorResponse, 0, len(monitors))
	for _, monitor := range monitors {
		items = append(items, toMonitorResponse(monitor.Monitor, monitor.LastResult))
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(filter.PageSize)))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"pagination": paginationResponse{
			Page:       filter.Page,
			PageSize:   filter.PageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func (h *Handler) getMonitor(w http.ResponseWriter, r *http.Request) {
	monitorID, ok := h.monitorID(w, r)
	if !ok {
		return
	}

	monitor, err := h.deps.Monitors.GetByID(r.Context(), monitorID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toMonitorResponse(monitor, nil))
}

func (h *Handler) updateMonitor(w http.ResponseWriter, r *http.Request) {
	monitorID, ok := h.monitorID(w, r)
	if !ok {
		return
	}

	existing, err := h.deps.Monitors.GetByID(r.Context(), monitorID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	var input domain.MonitorInput
	if err := decodeJSON(r, &input); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_json", "Request body is not valid JSON", nil)
		return
	}

	validated, fieldErrors := domain.ValidateMonitorInput(input)
	if len(fieldErrors) > 0 {
		writeError(w, r, http.StatusUnprocessableEntity, "validation_failed", "The request contains invalid fields", fieldErrors)
		return
	}

	validated.ID = existing.ID
	if err := h.deps.Monitors.Update(r.Context(), &validated); err != nil {
		h.deps.Logger.Error("update monitor failed", "monitor_id", monitorID, "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toMonitorResponse(validated, nil))
}

func (h *Handler) deleteMonitor(w http.ResponseWriter, r *http.Request) {
	monitorID, ok := h.monitorID(w, r)
	if !ok {
		return
	}

	if err := h.deps.Monitors.Delete(r.Context(), monitorID); err != nil {
		writeDomainError(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) pauseMonitor(w http.ResponseWriter, r *http.Request) {
	h.setMonitorEnabled(w, r, false)
}

func (h *Handler) resumeMonitor(w http.ResponseWriter, r *http.Request) {
	h.setMonitorEnabled(w, r, true)
}

func (h *Handler) setMonitorEnabled(w http.ResponseWriter, r *http.Request, enabled bool) {
	monitorID, ok := h.monitorID(w, r)
	if !ok {
		return
	}

	monitor, err := h.deps.Monitors.SetEnabled(r.Context(), monitorID, enabled)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toMonitorResponse(monitor, nil))
}

type resultResponse struct {
	ID              string               `json:"id"`
	JobID           string               `json:"job_id"`
	MonitorID       string               `json:"monitor_id"`
	ProbeLocationID string               `json:"probe_location_id"`
	Status          domain.MonitorStatus `json:"status"`
	Success         bool                 `json:"success"`
	ErrorCode       string               `json:"error_code,omitempty"`
	ErrorMessage    string               `json:"error_message,omitempty"`
	DurationMillis  int64                `json:"duration_millis"`
	Metrics         map[string]any       `json:"metrics"`
	Attributes      map[string]any       `json:"attributes"`
	StartedAt       time.Time            `json:"started_at"`
	FinishedAt      time.Time            `json:"finished_at"`
}

func (h *Handler) listMonitorResults(w http.ResponseWriter, r *http.Request) {
	monitorID, ok := h.monitorID(w, r)
	if !ok {
		return
	}

	if _, err := h.deps.Monitors.GetByID(r.Context(), monitorID); err != nil {
		writeDomainError(w, r, err)
		return
	}

	query := r.URL.Query()
	limit := parsePositiveInt(query.Get("limit"), 50, 500)
	page := parsePositiveInt(query.Get("page"), 1, 1000000)

	results, total, err := h.deps.Results.ListByMonitor(r.Context(), monitorID, limit, (page-1)*limit)
	if err != nil {
		h.deps.Logger.Error("list results failed", "monitor_id", monitorID, "error", err)
		writeDomainError(w, r, err)
		return
	}

	items := make([]resultResponse, 0, len(results))
	for _, result := range results {
		items = append(items, resultResponse(result))
	}

	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"items": items,
		"pagination": paginationResponse{
			Page:       page,
			PageSize:   limit,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

// monitorMetrics serves downsampled series for charts. Buckets are computed
// from stored probe results; step=auto follows the spec's downsampling table.
func (h *Handler) monitorMetrics(w http.ResponseWriter, r *http.Request) {
	monitorID, ok := h.monitorID(w, r)
	if !ok {
		return
	}

	if _, err := h.deps.Monitors.GetByID(r.Context(), monitorID); err != nil {
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

	series, err := h.deps.Results.Series(r.Context(), monitorID, from, to, stepSeconds)
	if err != nil {
		h.deps.Logger.Error("query metric series failed", "monitor_id", monitorID, "error", err)
		writeDomainError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"series": map[string]any{
			"latency": series.Latency,
			"success": series.Success,
		},
		"summary": map[string]any{
			"uptime_percent": series.Summary.UptimePercent,
			"p50_latency_ms": series.Summary.P50LatencyMS,
			"p95_latency_ms": series.Summary.P95LatencyMS,
			"p99_latency_ms": series.Summary.P99LatencyMS,
		},
		"step_seconds": stepSeconds,
		"from":         from,
		"to":           to,
	})
}

const maxChartPoints = 1500

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

func (h *Handler) monitorID(w http.ResponseWriter, r *http.Request) (string, bool) {
	monitorID := chi.URLParam(r, "monitorID")
	if _, err := uuid.Parse(monitorID); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid_id", "monitor id must be a valid UUID", nil)
		return "", false
	}

	return monitorID, true
}

func decodeJSON(r *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	if err := decoder.Decode(target); err != nil {
		return err
	}

	if decoder.More() {
		return errors.New("unexpected trailing data")
	}

	return nil
}

func parsePositiveInt(raw string, fallback, max int) int {
	if raw == "" {
		return fallback
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return fallback
	}

	if value > max {
		return max
	}

	return value
}
