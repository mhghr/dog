package postgres

import (
	"context"
	"time"

	"monitoring-platform/packages/shared/application/metricquery"
	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/repository"
)

// MetricQueryService is the PostgreSQL-backed implementation of the
// application metric query port. It currently reads time-series facts from
// probe_results and metric_points. When VictoriaMetrics becomes the primary
// read path, a sibling adapter can be substituted at wiring time without
// changing the API contract.
type MetricQueryService struct {
	results repository.ResultRepository
	series  repository.MetricSeriesRepository
}

// NewMetricQueryService builds the PostgreSQL metric query adapter.
func NewMetricQueryService(results repository.ResultRepository, series repository.MetricSeriesRepository) *MetricQueryService {
	return &MetricQueryService{results: results, series: series}
}

var _ metricquery.QueryService = (*MetricQueryService)(nil)

func (s *MetricQueryService) SeriesByProbe(ctx context.Context, monitorID string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error) {
	return s.results.SeriesByProbe(ctx, monitorID, from, to, stepSeconds, seriesLimit)
}

func (s *MetricQueryService) SeriesByProbeMetric(ctx context.Context, monitorID, metricKey string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error) {
	return s.results.SeriesByProbeMetric(ctx, monitorID, metricKey, from, to, stepSeconds, seriesLimit)
}

func (s *MetricQueryService) StatusSeriesByProbe(ctx context.Context, monitorID string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error) {
	return s.results.StatusSeriesByProbe(ctx, monitorID, from, to, stepSeconds, seriesLimit)
}

func (s *MetricQueryService) LatestSuccessAt(ctx context.Context, monitorID string) (*time.Time, error) {
	return s.results.LatestSuccessAt(ctx, monitorID)
}

func (s *MetricQueryService) StatusCodeDistribution(ctx context.Context, monitorID string, from, to time.Time) ([]domain.StatusCodeCount, error) {
	return s.results.StatusCodeDistribution(ctx, monitorID, from, to)
}

func (s *MetricQueryService) AggregateMetrics(ctx context.Context, monitorID, probeID string, from, to time.Time) (domain.MonitorAggregateMetrics, error) {
	return s.results.AggregateMetrics(ctx, monitorID, probeID, from, to)
}

func (s *MetricQueryService) ProbeMetrics(ctx context.Context, monitorID string, from, to time.Time) ([]domain.ProbeAggregateMetrics, error) {
	return s.results.ProbeMetrics(ctx, monitorID, from, to)
}
