package repository

import (
	"context"
	"time"

	"monitoring-platform/packages/shared/domain"
)

type MetricSeriesRepository interface {
	UpsertSeries(ctx context.Context, series *domain.MetricSeriesRow) error
	GetSeries(ctx context.Context, id string) (domain.MetricSeriesRow, error)
	FindSeries(ctx context.Context, resourceID, metricName string) (*domain.MetricSeriesRow, error)
	ListByResource(ctx context.Context, resourceID string) ([]domain.MetricSeriesRow, error)

	InsertPoints(ctx context.Context, points []domain.MetricPointDB) error
	QueryPoints(ctx context.Context, seriesID string, from, to time.Time) ([]domain.MetricPointDB, error)
	LatestPoint(ctx context.Context, seriesID string) (*domain.MetricPointDB, error)
	DeletePointsBefore(ctx context.Context, cutoff time.Time) (int64, error)

	UpsertRollup(ctx context.Context, rollup *domain.MetricRollup) error
	QueryRollups(ctx context.Context, seriesID string, from, to time.Time) ([]domain.MetricRollup, error)
}
