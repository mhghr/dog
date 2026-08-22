// Package metricquery defines the canonical read port for monitor metric
// data. Consumers (HTTP handlers, future gRPC, realtime services) depend only
// on this interface; the storage backend behind it is an implementation
// detail. Today the PostgreSQL adapter serves reads; a VictoriaMetrics
// adapter can be swapped in without touching callers.
package metricquery

import (
	"context"
	"time"

	"monitoring-platform/packages/shared/domain"
)

// QueryService is the application-layer port for range-scoped metric reads.
//
// It intentionally exposes the same queries the monitoring detail page needs
// (per-probe series, KPIs, status-code distribution) and nothing about the
// underlying storage. Moving reads from PostgreSQL to VictoriaMetrics is a
// backend decision made inside the adapter, not by the API contract.
type QueryService interface {
	// SeriesByProbe returns time-bucketed series per probe location for a
	// monitor, used by the resource monitoring dashboard.
	SeriesByProbe(ctx context.Context, monitorID string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error)
	// SeriesByProbeMetric returns per-location time-bucketed series for a
	// specific metric key.
	SeriesByProbeMetric(ctx context.Context, monitorID, metricKey string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error)
	// StatusSeriesByProbe returns per-location time-bucketed success ratios
	// (0..1) for a monitor, including failed checks.
	StatusSeriesByProbe(ctx context.Context, monitorID string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error)
	// LatestSuccessAt returns the most recent successful check time for a
	// monitor, or nil when there is none.
	LatestSuccessAt(ctx context.Context, monitorID string) (*time.Time, error)
	// StatusCodeDistribution returns HTTP status-code counts over a window,
	// ordered by count descending.
	StatusCodeDistribution(ctx context.Context, monitorID string, from, to time.Time) ([]domain.StatusCodeCount, error)
	// AggregateMetrics returns range-scoped KPIs for a monitor, optionally
	// filtered to one probe (empty probeID = all probes).
	AggregateMetrics(ctx context.Context, monitorID, probeID string, from, to time.Time) (domain.MonitorAggregateMetrics, error)
	// ProbeMetrics returns range-scoped KPIs grouped per probe location.
	ProbeMetrics(ctx context.Context, monitorID string, from, to time.Time) ([]domain.ProbeAggregateMetrics, error)
}
