// Package ingestion persists worker results, updates monitor state, fans out
// live events, and forwards metrics to VictoriaMetrics.
package ingestion

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"monitoring-platform/internal/domain"
	"monitoring-platform/internal/events"
	"monitoring-platform/internal/metrics"
	"monitoring-platform/internal/repository"
)

type Service struct {
	results   repository.ResultRepository
	monitors  repository.MonitorRepository
	locations repository.LocationRepository
	victoria  *metrics.VictoriaClient
	bus       *events.Bus
	logger    *slog.Logger
	counters  *metrics.IngestionMetrics

	locationCodes sync.Map // location id -> code
}

func NewService(
	results repository.ResultRepository,
	monitors repository.MonitorRepository,
	locations repository.LocationRepository,
	victoria *metrics.VictoriaClient,
	bus *events.Bus,
	logger *slog.Logger,
	counters *metrics.IngestionMetrics,
) *Service {
	return &Service{
		results:   results,
		monitors:  monitors,
		locations: locations,
		victoria:  victoria,
		bus:       bus,
		logger:    logger,
		counters:  counters,
	}
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// Ingest validates and stores a probe result. Returns true when the result
// was inserted, false when it was an idempotent duplicate.
func (s *Service) Ingest(ctx context.Context, result *domain.ProbeResult) (bool, error) {
	if err := validateResult(result); err != nil {
		return false, err
	}

	monitor, err := s.monitors.GetByID(ctx, result.MonitorID)
	if err != nil {
		return false, err
	}

	if result.Metrics == nil {
		result.Metrics = map[string]any{}
	}
	if result.Attributes == nil {
		result.Attributes = map[string]any{}
	}

	if monitor.Type == domain.MonitorTLS {
		s.detectCertificateChange(ctx, result)
	}

	inserted, err := s.results.InsertAndUpdateMonitor(ctx, result)
	if err != nil {
		return false, err
	}

	if !inserted {
		s.counters.DuplicateResults.Inc()
		s.logger.Info("duplicate probe result ignored", "job_id", result.JobID, "monitor_id", result.MonitorID)
		return false, nil
	}

	s.counters.ResultsTotal.Inc()

	locationCode := s.locationCode(ctx, result.ProbeLocationID)
	s.victoria.Enqueue(result, string(monitor.Type), locationCode)
	s.publishEvent(result)

	s.logger.Info(
		"probe result ingested",
		"job_id", result.JobID,
		"monitor_id", result.MonitorID,
		"type", monitor.Type,
		"success", result.Success,
		"duration_ms", result.DurationMillis,
		"error_code", result.ErrorCode,
	)

	return true, nil
}

func (s *Service) detectCertificateChange(ctx context.Context, result *domain.ProbeResult) {
	currentFingerprint, _ := result.Attributes["fingerprint_sha256"].(string)
	if currentFingerprint == "" {
		return
	}

	previousFingerprint, err := s.results.LatestAttribute(ctx, result.MonitorID, "fingerprint_sha256")
	if err != nil {
		s.logger.Warn("fingerprint lookup failed", "monitor_id", result.MonitorID, "error", err)
		return
	}

	changed := previousFingerprint != "" && previousFingerprint != currentFingerprint
	result.Attributes["certificate_changed"] = changed
	result.Metrics["certificate_changed"] = boolToInt(changed)
}

func (s *Service) locationCode(ctx context.Context, locationID string) string {
	if locationID == "" {
		return "unknown"
	}

	if cached, ok := s.locationCodes.Load(locationID); ok {
		return cached.(string)
	}

	locations, err := s.locations.List(ctx)
	if err != nil {
		return "unknown"
	}

	code := "unknown"
	for _, location := range locations {
		s.locationCodes.Store(location.ID, location.Code)
		if location.ID == locationID {
			code = location.Code
		}
	}

	return code
}

func (s *Service) publishEvent(result *domain.ProbeResult) {
	payload, err := json.Marshal(map[string]any{
		"monitor_id":  result.MonitorID,
		"status":      result.Status,
		"success":     result.Success,
		"duration_ms": result.DurationMillis,
		"error_code":  result.ErrorCode,
		"timestamp":   result.FinishedAt.UTC().Format(time.RFC3339),
	})
	if err != nil {
		return
	}

	s.bus.Publish(events.Event{
		Name: "probe-result",
		ID:   result.ID,
		Data: payload,
	})
}

func validateResult(result *domain.ProbeResult) error {
	if result.ID == "" || result.JobID == "" || result.MonitorID == "" {
		return &ValidationError{Message: "id, job_id and monitor_id are required"}
	}

	if _, ok := domain.ParseMonitorStatus(string(result.Status)); !ok {
		return &ValidationError{Message: fmt.Sprintf("invalid status %q", result.Status)}
	}

	if result.StartedAt.IsZero() || result.FinishedAt.IsZero() {
		return &ValidationError{Message: "started_at and finished_at are required"}
	}

	if result.FinishedAt.Before(result.StartedAt) {
		return &ValidationError{Message: "finished_at must not be before started_at"}
	}

	if result.DurationMillis < 0 {
		return &ValidationError{Message: "duration_millis must not be negative"}
	}

	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
