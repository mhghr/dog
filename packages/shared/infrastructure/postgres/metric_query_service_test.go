package postgres

import (
	"context"
	"testing"
	"time"

	"monitoring-platform/packages/shared/application/metricquery"
	"monitoring-platform/packages/shared/domain"
)

// fakeMetricQueryResults stubs the repository.ResultRepository surface used by
// MetricQueryService so the adapter's delegation is exercised without a DB.
type fakeMetricQueryResults struct {
	aggregate domain.MonitorAggregateMetrics
	probes    []domain.ProbeAggregateMetrics
	status    []domain.StatusCodeCount
	series    []domain.ProbeSeries
}

func (f *fakeMetricQueryResults) SeriesByProbe(ctx context.Context, monitorID string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error) {
	return f.series, nil
}

func (f *fakeMetricQueryResults) SeriesByProbeMetric(ctx context.Context, monitorID, metricKey string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error) {
	return f.series, nil
}

func (f *fakeMetricQueryResults) StatusSeriesByProbe(ctx context.Context, monitorID string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error) {
	return f.series, nil
}

func (f *fakeMetricQueryResults) LatestSuccessAt(ctx context.Context, monitorID string) (*time.Time, error) {
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return &at, nil
}

func (f *fakeMetricQueryResults) StatusCodeDistribution(ctx context.Context, monitorID string, from, to time.Time) ([]domain.StatusCodeCount, error) {
	return f.status, nil
}

func (f *fakeMetricQueryResults) AggregateMetrics(ctx context.Context, monitorID, probeID string, from, to time.Time) (domain.MonitorAggregateMetrics, error) {
	return f.aggregate, nil
}

func (f *fakeMetricQueryResults) ProbeMetrics(ctx context.Context, monitorID string, from, to time.Time) ([]domain.ProbeAggregateMetrics, error) {
	return f.probes, nil
}

func (f *fakeMetricQueryResults) DashboardSummary(ctx context.Context) (domain.DashboardSummary, error) {
	return domain.DashboardSummary{}, nil
}

func (f *fakeMetricQueryResults) InsertAndUpdateMonitor(ctx context.Context, result *domain.ProbeResult) (bool, error) {
	return false, nil
}

func (f *fakeMetricQueryResults) ListByMonitor(ctx context.Context, monitorID string, limit, offset int) ([]domain.ProbeResult, int, error) {
	return nil, 0, nil
}

func (f *fakeMetricQueryResults) LatestAttribute(ctx context.Context, monitorID, key string) (string, error) {
	return "", nil
}

func (f *fakeMetricQueryResults) Series(ctx context.Context, monitorID string, from, to time.Time, stepSeconds int) (domain.MetricSeries, error) {
	return domain.MetricSeries{}, nil
}

func (f *fakeMetricQueryResults) LatestResultsByProbe(ctx context.Context, monitorID string, limit int) ([]domain.ProbeResult, error) {
	return nil, nil
}

func (f *fakeMetricQueryResults) ResultAt(ctx context.Context, monitorID, probeID string, at time.Time) (domain.ProbeResult, error) {
	return domain.ProbeResult{}, nil
}

// TestMetricQueryServiceDelegates verifies the PostgreSQL adapter forwards each
// port method to the underlying result repository unchanged. This keeps the
// application layer the single read path for metrics (section 17).
func TestMetricQueryServiceDelegates(t *testing.T) {
	availability := 99.5
	fake := &fakeMetricQueryResults{
		aggregate: domain.MonitorAggregateMetrics{Availability: &availability},
		probes:    []domain.ProbeAggregateMetrics{{ProbeID: "p1"}},
		status:    []domain.StatusCodeCount{{Code: 200, Count: 10}},
		series:    []domain.ProbeSeries{{ProbeID: "p1"}},
	}

	svc := NewMetricQueryService(fake, nil)
	ctx := context.Background()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(time.Hour)

	var _ metricquery.QueryService = svc

	series, err := svc.SeriesByProbe(ctx, "m1", from, to, 60, 25)
	if err != nil || len(series) != 1 {
		t.Fatalf("SeriesByProbe: series=%v err=%v", series, err)
	}

	series, err = svc.SeriesByProbeMetric(ctx, "m1", "rtt_ms", from, to, 60, 25)
	if err != nil || len(series) != 1 {
		t.Fatalf("SeriesByProbeMetric: series=%v err=%v", series, err)
	}

	series, err = svc.StatusSeriesByProbe(ctx, "m1", from, to, 60, 25)
	if err != nil || len(series) != 1 {
		t.Fatalf("StatusSeriesByProbe: series=%v err=%v", series, err)
	}

	latest, err := svc.LatestSuccessAt(ctx, "m1")
	if err != nil || latest == nil {
		t.Fatalf("LatestSuccessAt: latest=%v err=%v", latest, err)
	}

	status, err := svc.StatusCodeDistribution(ctx, "m1", from, to)
	if err != nil || len(status) != 1 || status[0].Count != 10 {
		t.Fatalf("StatusCodeDistribution: status=%v err=%v", status, err)
	}

	agg, err := svc.AggregateMetrics(ctx, "m1", "", from, to)
	if err != nil || agg.Availability == nil || *agg.Availability != 99.5 {
		t.Fatalf("AggregateMetrics: agg=%v err=%v", agg, err)
	}

	probes, err := svc.ProbeMetrics(ctx, "m1", from, to)
	if err != nil || len(probes) != 1 || probes[0].ProbeID != "p1" {
		t.Fatalf("ProbeMetrics: probes=%v err=%v", probes, err)
	}
}
