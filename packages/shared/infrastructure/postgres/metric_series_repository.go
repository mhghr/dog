package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type MetricSeriesRepository struct {
	pool *pgxpool.Pool
}

func NewMetricSeriesRepository(pool *pgxpool.Pool) *MetricSeriesRepository {
	return &MetricSeriesRepository{pool: pool}
}

func (r *MetricSeriesRepository) UpsertSeries(ctx context.Context, series *domain.MetricSeriesRow) error {
	labelsJSON := series.Labels
	if len(labelsJSON) == 0 {
		labelsJSON = json.RawMessage("{}")
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO metric_series (resource_id, metric_name, labels, unit)
		VALUES ($1::uuid, $2, $3::jsonb, $4)
		ON CONFLICT (resource_id, metric_name, labels) DO UPDATE SET unit = EXCLUDED.unit
		RETURNING id::text, created_at`,
		series.ResourceID, series.MetricName, labelsJSON, series.Unit,
	).Scan(&series.ID, &series.CreatedAt)
	if err != nil {
		return fmt.Errorf("upsert metric series: %w", err)
	}
	return nil
}

func (r *MetricSeriesRepository) GetSeries(ctx context.Context, id string) (domain.MetricSeriesRow, error) {
	return r.scanSeries(r.pool.QueryRow(ctx, `
		SELECT id::text, resource_id::text, metric_name, labels, unit, created_at
		FROM metric_series WHERE id = $1::uuid`, id))
}

func (r *MetricSeriesRepository) FindSeries(ctx context.Context, resourceID, metricName string) (*domain.MetricSeriesRow, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id::text, resource_id::text, metric_name, labels, unit, created_at
		FROM metric_series WHERE resource_id = $1::uuid AND metric_name = $2`, resourceID, metricName)

	var s domain.MetricSeriesRow
	var labelsJSON []byte
	err := row.Scan(&s.ID, &s.ResourceID, &s.MetricName, &labelsJSON, &s.Unit, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.Labels = labelsJSON
	return &s, nil
}

func (r *MetricSeriesRepository) ListByResource(ctx context.Context, resourceID string) ([]domain.MetricSeriesRow, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, resource_id::text, metric_name, labels, unit, created_at
		FROM metric_series WHERE resource_id = $1::uuid ORDER BY metric_name`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list metric series: %w", err)
	}
	defer rows.Close()

	var series []domain.MetricSeriesRow
	for rows.Next() {
		s, err := r.scanSeriesFromRows(rows)
		if err != nil {
			return nil, err
		}
		series = append(series, s)
	}
	return series, rows.Err()
}

func (r *MetricSeriesRepository) InsertPoints(ctx context.Context, points []domain.MetricPointDB) error {
	if len(points) == 0 {
		return nil
	}

	b := &strings.Builder{}
	b.WriteString("INSERT INTO metric_points (time, series_id, value, attributes) VALUES ")

	args := make([]any, 0, len(points)*4)
	idx := 0
	for i, p := range points {
		if i > 0 {
			b.WriteString(", ")
		}
		attrsJSON := p.Attributes
		if len(attrsJSON) == 0 {
			attrsJSON = json.RawMessage("{}")
		}
		fmt.Fprintf(b, "($%d::timestamptz, $%d::uuid, $%d, $%d::jsonb)",
			idx+1, idx+2, idx+3, idx+4)
		args = append(args, p.Time, p.SeriesID, p.Value, attrsJSON)
		idx += 4
	}

	b.WriteString(" ON CONFLICT (time, series_id) DO NOTHING")

	_, err := r.pool.Exec(ctx, b.String(), args...)
	if err != nil {
		return fmt.Errorf("insert metric points: %w", err)
	}
	return nil
}

func (r *MetricSeriesRepository) QueryPoints(ctx context.Context, seriesID string, from, to time.Time) ([]domain.MetricPointDB, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT time, series_id::text, value, attributes
		FROM metric_points
		WHERE series_id = $1::uuid AND time >= $2 AND time <= $3
		ORDER BY time`, seriesID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query metric points: %w", err)
	}
	defer rows.Close()

	return r.scanPoints(rows)
}

func (r *MetricSeriesRepository) LatestPoint(ctx context.Context, seriesID string) (*domain.MetricPointDB, error) {
	var p domain.MetricPointDB
	err := r.pool.QueryRow(ctx, `
		SELECT time, series_id::text, value, attributes
		FROM metric_points
		WHERE series_id = $1::uuid
		ORDER BY time DESC LIMIT 1`, seriesID,
	).Scan(&p.Time, &p.SeriesID, &p.Value, &p.Attributes)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get latest point: %w", err)
	}
	return &p, nil
}

func (r *MetricSeriesRepository) DeletePointsBefore(ctx context.Context, cutoff time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM metric_points WHERE time < $1`, cutoff)
	if err != nil {
		return 0, fmt.Errorf("delete old points: %w", err)
	}
	return tag.RowsAffected(), nil
}

func (r *MetricSeriesRepository) UpsertRollup(ctx context.Context, rollup *domain.MetricRollup) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO metric_rollups (series_id, bucket, bucket_width_seconds, count, sum, min, max, p50, p95, p99, last_value)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (series_id, bucket, bucket_width_seconds) DO UPDATE SET
			count = EXCLUDED.count, sum = EXCLUDED.sum, min = EXCLUDED.min, max = EXCLUDED.max,
			p50 = EXCLUDED.p50, p95 = EXCLUDED.p95, p99 = EXCLUDED.p99, last_value = EXCLUDED.last_value
		RETURNING id::text, created_at`,
		rollup.SeriesID, rollup.Bucket, rollup.BucketWidthSeconds, rollup.Count, rollup.Sum,
		rollup.Min, rollup.Max, rollup.P50, rollup.P95, rollup.P99, rollup.LastValue,
	).Scan(&rollup.ID, &rollup.CreatedAt)
	if err != nil {
		return fmt.Errorf("upsert metric rollup: %w", err)
	}
	return nil
}

func (r *MetricSeriesRepository) QueryRollups(ctx context.Context, seriesID string, from, to time.Time) ([]domain.MetricRollup, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, series_id::text, bucket, bucket_width_seconds, count, sum, min, max, p50, p95, p99, last_value, created_at
		FROM metric_rollups
		WHERE series_id = $1::uuid AND bucket >= $2 AND bucket <= $3
		ORDER BY bucket`, seriesID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query metric rollups: %w", err)
	}
	defer rows.Close()

	var rollups []domain.MetricRollup
	for rows.Next() {
		var r domain.MetricRollup
		if err := rows.Scan(&r.ID, &r.SeriesID, &r.Bucket, &r.BucketWidthSeconds, &r.Count, &r.Sum,
			&r.Min, &r.Max, &r.P50, &r.P95, &r.P99, &r.LastValue, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan rollup: %w", err)
		}
		rollups = append(rollups, r)
	}
	return rollups, rows.Err()
}

func (r *MetricSeriesRepository) scanSeries(row pgx.Row) (domain.MetricSeriesRow, error) {
	var s domain.MetricSeriesRow
	var labelsJSON []byte
	err := row.Scan(&s.ID, &s.ResourceID, &s.MetricName, &labelsJSON, &s.Unit, &s.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MetricSeriesRow{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MetricSeriesRow{}, err
	}
	s.Labels = labelsJSON
	return s, nil
}

func (r *MetricSeriesRepository) scanSeriesFromRows(rows pgx.Rows) (domain.MetricSeriesRow, error) {
	var s domain.MetricSeriesRow
	var labelsJSON []byte
	if err := rows.Scan(&s.ID, &s.ResourceID, &s.MetricName, &labelsJSON, &s.Unit, &s.CreatedAt); err != nil {
		return domain.MetricSeriesRow{}, fmt.Errorf("scan metric series: %w", err)
	}
	s.Labels = labelsJSON
	return s, nil
}

func (r *MetricSeriesRepository) scanPoints(rows pgx.Rows) ([]domain.MetricPointDB, error) {
	var points []domain.MetricPointDB
	for rows.Next() {
		var p domain.MetricPointDB
		if err := rows.Scan(&p.Time, &p.SeriesID, &p.Value, &p.Attributes); err != nil {
			return nil, fmt.Errorf("scan metric point: %w", err)
		}
		points = append(points, p)
	}
	return points, rows.Err()
}
