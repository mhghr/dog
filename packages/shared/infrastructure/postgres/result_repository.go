package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type ResultRepository struct {
	pool *pgxpool.Pool
}

func NewResultRepository(pool *pgxpool.Pool) *ResultRepository {
	return &ResultRepository{pool: pool}
}

func (r *ResultRepository) InsertAndUpdateMonitor(ctx context.Context, result *domain.ProbeResult) (bool, error) {
	metricsJSON, err := json.Marshal(result.Metrics)
	if err != nil {
		return false, fmt.Errorf("marshal result metrics: %w", err)
	}

	attributesJSON, err := json.Marshal(result.Attributes)
	if err != nil {
		return false, fmt.Errorf("marshal result attributes: %w", err)
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin ingestion transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	var locationID any
	if result.ProbeLocationID != "" {
		locationID = result.ProbeLocationID
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO probe_results (
			id, job_id, monitor_id, probe_location_id, status, success,
			error_code, error_message, duration_millis, metrics, attributes,
			started_at, finished_at, attempt
		)
		VALUES (
			$1::uuid, $2::uuid, $3::uuid, $4::uuid, $5::monitor_status, $6,
			NULLIF($7, ''), NULLIF($8, ''), $9, $10::jsonb, $11::jsonb, $12, $13, $14
		)
		ON CONFLICT (job_id, COALESCE(probe_location_id, '00000000-0000-0000-0000-000000000000'::uuid), attempt) DO NOTHING`,
		result.ID,
		result.JobID,
		result.MonitorID,
		locationID,
		string(result.Status),
		result.Success,
		result.ErrorCode,
		result.ErrorMessage,
		result.DurationMillis,
		metricsJSON,
		attributesJSON,
		result.StartedAt,
		result.FinishedAt,
		getAttemptFromAttributes(result.Attributes),
	)
	if err != nil {
		return false, fmt.Errorf("insert probe result: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return false, nil // duplicate job_id: idempotent no-op
	}

	if _, err := tx.Exec(ctx, `
		UPDATE monitors
		SET
			last_status = $2::monitor_status,
			last_checked_at = $3,
			updated_at = NOW()
		WHERE id = $1::uuid
		  AND enabled = TRUE
		  AND (last_checked_at IS NULL OR last_checked_at < $3)`,
		result.MonitorID,
		string(result.Status),
		result.FinishedAt,
	); err != nil {
		return false, fmt.Errorf("update monitor status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit ingestion transaction: %w", err)
	}

	return true, nil
}

func (r *ResultRepository) ListByMonitor(ctx context.Context, monitorID string, limit, offset int) ([]domain.ProbeResult, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM probe_results WHERE monitor_id = $1::uuid`, monitorID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count probe results: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text, job_id::text, monitor_id::text,
			COALESCE(probe_location_id::text, ''),
			status::text, success,
			COALESCE(error_code, ''), COALESCE(error_message, ''),
			duration_millis, metrics, attributes, started_at, finished_at
		FROM probe_results
		WHERE monitor_id = $1::uuid
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3`,
		monitorID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list probe results: %w", err)
	}
	defer rows.Close()

	results := make([]domain.ProbeResult, 0, limit)
	for rows.Next() {
		var (
			result         domain.ProbeResult
			metricsJSON    []byte
			attributesJSON []byte
		)

		if err := rows.Scan(
			&result.ID, &result.JobID, &result.MonitorID, &result.ProbeLocationID,
			&result.Status, &result.Success,
			&result.ErrorCode, &result.ErrorMessage,
			&result.DurationMillis, &metricsJSON, &attributesJSON,
			&result.StartedAt, &result.FinishedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan probe result: %w", err)
		}

		if err := json.Unmarshal(metricsJSON, &result.Metrics); err != nil {
			result.Metrics = map[string]any{}
		}
		if err := json.Unmarshal(attributesJSON, &result.Attributes); err != nil {
			result.Attributes = map[string]any{}
		}

		results = append(results, result)
	}

	return results, total, rows.Err()
}

func (r *ResultRepository) LatestAttribute(ctx context.Context, monitorID, key string) (string, error) {
	var value *string
	err := r.pool.QueryRow(ctx, `
		SELECT attributes->>$2
		FROM probe_results
		WHERE monitor_id = $1::uuid
		ORDER BY started_at DESC
		LIMIT 1`,
		monitorID, key,
	).Scan(&value)

	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("fetch latest attribute: %w", err)
	}

	if value == nil {
		return "", nil
	}

	return *value, nil
}

func (r *ResultRepository) Series(ctx context.Context, monitorID string, from, to time.Time, stepSeconds int) (domain.MetricSeries, error) {
	series := domain.MetricSeries{
		Latency: []domain.MetricPoint{},
		Success: []domain.MetricPoint{},
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			date_bin(make_interval(secs => $4::int), started_at, TIMESTAMPTZ 'epoch') AS bucket,
			AVG(duration_millis)::float8 AS latency,
			AVG(CASE WHEN success THEN 1 ELSE 0 END)::float8 AS success_ratio
		FROM probe_results
		WHERE monitor_id = $1::uuid
		  AND started_at >= $2
		  AND started_at < $3
		GROUP BY bucket
		ORDER BY bucket`,
		monitorID, from, to, stepSeconds)
	if err != nil {
		return series, fmt.Errorf("query metric series: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			bucket       time.Time
			latency      float64
			successRatio float64
		)

		if err := rows.Scan(&bucket, &latency, &successRatio); err != nil {
			return series, fmt.Errorf("scan metric bucket: %w", err)
		}

		series.Latency = append(series.Latency, domain.MetricPoint{Timestamp: bucket, Value: latency})
		series.Success = append(series.Success, domain.MetricPoint{Timestamp: bucket, Value: successRatio})
	}
	if err := rows.Err(); err != nil {
		return series, err
	}

	err = r.pool.QueryRow(ctx, `
		SELECT
			AVG(CASE WHEN success THEN 1 ELSE 0 END)::float8 * 100,
			PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY duration_millis) FILTER (WHERE success),
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_millis) FILTER (WHERE success),
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_millis) FILTER (WHERE success)
		FROM probe_results
		WHERE monitor_id = $1::uuid
		  AND started_at >= $2
		  AND started_at < $3`,
		monitorID, from, to,
	).Scan(
		&series.Summary.UptimePercent,
		&series.Summary.P50LatencyMS,
		&series.Summary.P95LatencyMS,
		&series.Summary.P99LatencyMS,
	)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return series, fmt.Errorf("query metric summary: %w", err)
	}

	return series, nil
}

func (r *ResultRepository) DashboardSummary(ctx context.Context) (domain.DashboardSummary, error) {
	summary := domain.DashboardSummary{
		StatusCounts:    map[string]int{"up": 0, "down": 0, "unknown": 0, "paused": 0},
		RecentFailures:  []domain.RecentFailure{},
		SlowestMonitors: []domain.SlowMonitor{},
	}

	rows, err := r.pool.Query(ctx,
		`SELECT last_status::text, COUNT(*) FROM monitors GROUP BY last_status`)
	if err != nil {
		return summary, fmt.Errorf("query status counts: %w", err)
	}

	for rows.Next() {
		var (
			status string
			count  int
		)
		if err := rows.Scan(&status, &count); err != nil {
			rows.Close()
			return summary, err
		}
		summary.StatusCounts[status] = count
		summary.TotalMonitors += count
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return summary, err
	}

	err = r.pool.QueryRow(ctx, `
		SELECT
			COUNT(*) FILTER (WHERE success),
			COUNT(*) FILTER (WHERE NOT success),
			AVG(CASE WHEN success THEN 1 ELSE 0 END)::float8 * 100
		FROM probe_results
		WHERE started_at > NOW() - INTERVAL '24 hours'`,
	).Scan(&summary.Checks24h.Successful, &summary.Checks24h.Failed, &summary.Availability24h)
	if err != nil {
		return summary, fmt.Errorf("query 24h checks: %w", err)
	}

	failureRows, err := r.pool.Query(ctx, `
		SELECT
			m.id::text, m.name, m.type::text,
			pr.error_code, pr.error_message, pr.started_at
		FROM probe_results pr
		JOIN monitors m ON m.id = pr.monitor_id
		WHERE pr.success = FALSE
		ORDER BY pr.started_at DESC
		LIMIT 10`)
	if err != nil {
		return summary, fmt.Errorf("query recent failures: %w", err)
	}

	for failureRows.Next() {
		var failure domain.RecentFailure
		if err := failureRows.Scan(
			&failure.MonitorID, &failure.MonitorName, &failure.MonitorType,
			&failure.ErrorCode, &failure.ErrorMessage, &failure.StartedAt,
		); err != nil {
			failureRows.Close()
			return summary, err
		}
		summary.RecentFailures = append(summary.RecentFailures, failure)
	}
	failureRows.Close()
	if err := failureRows.Err(); err != nil {
		return summary, err
	}

	slowRows, err := r.pool.Query(ctx, `
		SELECT m.id::text, m.name, m.type::text, lr.duration_millis
		FROM monitors m
		JOIN LATERAL (
			SELECT duration_millis
			FROM probe_results pr
			WHERE pr.monitor_id = m.id AND pr.success = TRUE
			ORDER BY pr.started_at DESC
			LIMIT 1
		) lr ON TRUE
		WHERE m.enabled = TRUE
		ORDER BY lr.duration_millis DESC
		LIMIT 5`)
	if err != nil {
		return summary, fmt.Errorf("query slowest monitors: %w", err)
	}

	for slowRows.Next() {
		var slow domain.SlowMonitor
		if err := slowRows.Scan(&slow.MonitorID, &slow.MonitorName, &slow.MonitorType, &slow.DurationMillis); err != nil {
			slowRows.Close()
			return summary, err
		}
		summary.SlowestMonitors = append(summary.SlowestMonitors, slow)
	}
	slowRows.Close()
	if err := slowRows.Err(); err != nil {
		return summary, err
	}

	err = r.pool.QueryRow(ctx, `
		WITH latest AS (
			SELECT DISTINCT ON (pr.monitor_id)
				pr.monitor_id,
				m.type,
				m.config,
				pr.error_code,
				pr.metrics
			FROM probe_results pr
			JOIN monitors m ON m.id = pr.monitor_id
			WHERE m.enabled = TRUE
			ORDER BY pr.monitor_id, pr.started_at DESC
		)
		SELECT
			COUNT(*) FILTER (
				WHERE type = 'tls'
				  AND (metrics->>'days_remaining') IS NOT NULL
				  AND (metrics->>'days_remaining')::numeric <= COALESCE((config->>'warning_days')::numeric, 30)
			),
			COUNT(*) FILTER (
				WHERE type = 'domain_expiration'
				  AND (metrics->>'days_remaining') IS NOT NULL
				  AND (metrics->>'days_remaining')::numeric <= COALESCE((config->>'warning_days')::numeric, 45)
			),
			COUNT(*) FILTER (
				WHERE type = 'smtp'
				  AND error_code IN ('smtp_starttls_unavailable', 'smtp_starttls_failed', 'smtp_tls_invalid')
			),
			COUNT(*) FILTER (
				WHERE type = 'ntp'
				  AND error_code = 'ntp_offset_too_high'
			)
		FROM latest`,
	).Scan(
		&summary.Attention.CertificatesExpiring30d,
		&summary.Attention.DomainsExpiring45d,
		&summary.Attention.SMTPStartTLSFailures,
		&summary.Attention.NTPHighOffset,
	)
	if err != nil {
		return summary, fmt.Errorf("query attention summary: %w", err)
	}

	return summary, nil
}

func getAttemptFromAttributes(attrs map[string]any) int {
	if attrs == nil {
		return 1
	}
	switch v := attrs["attempt"].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	default:
		return 1
	}
}

type rawPoint struct {
	probeID   string
	probeName string
	location  string
	ts        time.Time
	value     float64
}

// scanProbeSeriesRows reads raw (probe_id, probe_name, location, bucket,
// value) rows produced by the per-probe series queries.
func scanProbeSeriesRows(rows pgx.Rows, scanErr string) ([]rawPoint, error) {
	var raw []rawPoint
	for rows.Next() {
		var (
			pid, pname, loc string
			bucket          time.Time
			value           float64
		)
		if err := rows.Scan(&pid, &pname, &loc, &bucket, &value); err != nil {
			return nil, fmt.Errorf("%s: %w", scanErr, err)
		}
		raw = append(raw, rawPoint{probeID: pid, probeName: pname, location: loc, ts: bucket, value: value})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return raw, nil
}

// groupProbeSeries groups raw points by probe location, preserving name
// ordering, and returns one time-bucketed series per probe.
func groupProbeSeries(raw []rawPoint, metricKey string) []domain.ProbeSeries {
	var series []domain.ProbeSeries
	var current *domain.ProbeSeries
	for _, p := range raw {
		if current == nil || current.ProbeID != p.probeID {
			series = append(series, domain.ProbeSeries{
				ProbeID:   p.probeID,
				ProbeName: p.probeName,
				Location:  p.location,
				MetricKey: metricKey,
				Points:    []domain.MetricPoint{},
				Values:    []domain.MetricPoint{},
			})
			current = &series[len(series)-1]
		}
		current.Points = append(current.Points, domain.MetricPoint{Timestamp: p.ts, Value: p.value})
		current.Values = append(current.Values, domain.MetricPoint{Timestamp: p.ts, Value: p.value})
	}
	return series
}

// SeriesByProbe returns one time-bucketed series per probe location for a
// monitor. Buckets are built over the [from, to) window; each series holds
// the average latency (ms) per bucket.
func (r *ResultRepository) SeriesByProbe(ctx context.Context, monitorID string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error) {
	rows, err := r.pool.Query(ctx, `
		WITH selected_probes AS (
			SELECT DISTINCT ON (probe_location_id) probe_location_id
			FROM probe_results
			WHERE monitor_id = $1::uuid AND started_at >= $2 AND started_at < $3
			ORDER BY probe_location_id, started_at DESC
			LIMIT $5
		)
		SELECT
			pl.id::text,
			COALESCE(pl.name, ''),
			COALESCE(pl.code, ''),
			date_bin(make_interval(secs => $4::int), pr.started_at, TIMESTAMPTZ 'epoch') AS bucket,
			AVG(pr.duration_millis)::float8 AS latency
		FROM probe_results pr
		LEFT JOIN probe_locations pl ON pl.id = pr.probe_location_id
		JOIN selected_probes sp ON sp.probe_location_id = pr.probe_location_id
		WHERE pr.monitor_id = $1::uuid
		  AND pr.success = TRUE
		  AND pr.started_at >= $2
		  AND pr.started_at < $3
		GROUP BY pl.id, pl.name, pl.code, bucket
		ORDER BY pl.name, bucket`,
		monitorID, from, to, stepSeconds, seriesLimit)
	if err != nil {
		return nil, fmt.Errorf("query per-probe series: %w", err)
	}
	defer rows.Close()

	raw, err := scanProbeSeriesRows(rows, "scan per-probe bucket")
	if err != nil {
		return nil, err
	}

	return groupProbeSeries(raw, ""), nil
}

// SeriesByProbeMetric returns one time-bucketed series per probe location for
// a specific metric key in the probe_results.metrics JSONB column.
func (r *ResultRepository) SeriesByProbeMetric(ctx context.Context, monitorID, metricKey string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error) {
	rows, err := r.pool.Query(ctx, `
		WITH selected_probes AS (
			SELECT DISTINCT ON (probe_location_id) probe_location_id
			FROM probe_results
			WHERE monitor_id = $1::uuid AND started_at >= $2 AND started_at < $3
			ORDER BY probe_location_id, started_at DESC
			LIMIT $6
		)
		SELECT
			pl.id::text,
			COALESCE(pl.name, ''),
			COALESCE(pl.code, ''),
			date_bin(make_interval(secs => $4::int), pr.started_at, TIMESTAMPTZ 'epoch') AS bucket,
			AVG((pr.metrics->>$5)::numeric)::float8 AS value
		FROM probe_results pr
		LEFT JOIN probe_locations pl ON pl.id = pr.probe_location_id
		JOIN selected_probes sp ON sp.probe_location_id = pr.probe_location_id
		WHERE pr.monitor_id = $1::uuid
		  AND pr.success = TRUE
		  AND pr.started_at >= $2
		  AND pr.started_at < $3
		  AND pr.metrics->>$5 IS NOT NULL
		GROUP BY pl.id, pl.name, pl.code, bucket
		ORDER BY pl.name, bucket`,
		monitorID, from, to, stepSeconds, metricKey, seriesLimit)
	if err != nil {
		return nil, fmt.Errorf("query per-probe metric series: %w", err)
	}
	defer rows.Close()

	raw, err := scanProbeSeriesRows(rows, "scan per-probe metric bucket")
	if err != nil {
		return nil, err
	}

	return groupProbeSeries(raw, metricKey), nil
}

// LatestResultsByProbe returns the most recent probe result for each probe
// location assigned to a monitor.
func (r *ResultRepository) LatestResultsByProbe(ctx context.Context, monitorID string, limit int) ([]domain.ProbeResult, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT DISTINCT ON (pr.probe_location_id)
			pr.id::text, pr.job_id::text, pr.monitor_id::text,
			COALESCE(pr.probe_location_id::text, ''),
			pr.status::text, pr.success,
			COALESCE(pr.error_code, ''), COALESCE(pr.error_message, ''),
			pr.duration_millis, pr.metrics, pr.attributes, pr.started_at, pr.finished_at,
			COALESCE(pl.name, ''), COALESCE(pl.code, '')
		FROM probe_results pr
		LEFT JOIN probe_locations pl ON pl.id = pr.probe_location_id
		WHERE pr.monitor_id = $1::uuid
		ORDER BY pr.probe_location_id, pr.started_at DESC
		LIMIT NULLIF($2, 0)`,
		monitorID, limit)
	if err != nil {
		return nil, fmt.Errorf("query latest results by probe: %w", err)
	}
	defer rows.Close()

	results := make([]domain.ProbeResult, 0)
	for rows.Next() {
		var (
			result         domain.ProbeResult
			metricsJSON    []byte
			attributesJSON []byte
			probeName      string
			probeCode      string
		)
		if err := rows.Scan(
			&result.ID, &result.JobID, &result.MonitorID, &result.ProbeLocationID,
			&result.Status, &result.Success,
			&result.ErrorCode, &result.ErrorMessage,
			&result.DurationMillis, &metricsJSON, &attributesJSON,
			&result.StartedAt, &result.FinishedAt, &probeName, &probeCode,
		); err != nil {
			return nil, fmt.Errorf("scan latest result: %w", err)
		}
		json.Unmarshal(metricsJSON, &result.Metrics)
		if err := json.Unmarshal(attributesJSON, &result.Attributes); err != nil {
			result.Attributes = map[string]any{}
		}
		if result.Attributes == nil {
			result.Attributes = map[string]any{}
		}
		result.Attributes["probe_name"] = probeName
		result.Attributes["probe_code"] = probeCode
		results = append(results, result)
	}

	return results, rows.Err()
}

// StatusSeriesByProbe returns one time-bucketed success-ratio series per
// probe location for a monitor, including failed checks. This is the explicit
// availability signal: 1.0 means fully up in that bucket, 0.0 fully down, and
// an absent bucket means no data (never inferred as downtime by consumers).
func (r *ResultRepository) StatusSeriesByProbe(ctx context.Context, monitorID string, from, to time.Time, stepSeconds, seriesLimit int) ([]domain.ProbeSeries, error) {
	rows, err := r.pool.Query(ctx, `
		WITH selected_probes AS (
			SELECT DISTINCT ON (probe_location_id) probe_location_id
			FROM probe_results
			WHERE monitor_id = $1::uuid AND started_at >= $2 AND started_at < $3
			ORDER BY probe_location_id, started_at DESC
			LIMIT $5
		)
		SELECT
			pl.id::text,
			COALESCE(pl.name, ''),
			COALESCE(pl.code, ''),
			date_bin(make_interval(secs => $4::int), pr.started_at, TIMESTAMPTZ 'epoch') AS bucket,
			AVG(CASE WHEN pr.success THEN 1 ELSE 0 END)::float8 AS value
		FROM probe_results pr
		LEFT JOIN probe_locations pl ON pl.id = pr.probe_location_id
		JOIN selected_probes sp ON sp.probe_location_id = pr.probe_location_id
		WHERE pr.monitor_id = $1::uuid
		  AND pr.started_at >= $2
		  AND pr.started_at < $3
		GROUP BY pl.id, pl.name, pl.code, bucket
		ORDER BY pl.name, bucket`,
		monitorID, from, to, stepSeconds, seriesLimit)
	if err != nil {
		return nil, fmt.Errorf("query status series: %w", err)
	}
	defer rows.Close()

	raw, err := scanProbeSeriesRows(rows, "scan status bucket")
	if err != nil {
		return nil, err
	}

	return groupProbeSeries(raw, "status"), nil
}

// LatestSuccessAt returns the most recent successful check time, or nil.
func (r *ResultRepository) LatestSuccessAt(ctx context.Context, monitorID string) (*time.Time, error) {
	var at time.Time
	err := r.pool.QueryRow(ctx, `
		SELECT started_at
		FROM probe_results
		WHERE monitor_id = $1::uuid AND success = TRUE
		ORDER BY started_at DESC
		LIMIT 1`,
		monitorID,
	).Scan(&at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query latest success: %w", err)
	}
	return &at, nil
}

// StatusCodeDistribution returns HTTP status-code counts for a monitor over a
// time window, ordered by count descending. Only results that recorded a
// status code are counted.
func (r *ResultRepository) StatusCodeDistribution(ctx context.Context, monitorID string, from, to time.Time) ([]domain.StatusCodeCount, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT (attributes->>'status_code')::int AS code, COUNT(*)::bigint
		FROM probe_results
		WHERE monitor_id = $1::uuid
		  AND started_at >= $2
		  AND started_at < $3
		  AND attributes ? 'status_code'
		GROUP BY code
		ORDER BY count DESC`,
		monitorID, from, to)
	if err != nil {
		return nil, fmt.Errorf("query status code distribution: %w", err)
	}
	defer rows.Close()

	distribution := make([]domain.StatusCodeCount, 0)
	for rows.Next() {
		var entry domain.StatusCodeCount
		if err := rows.Scan(&entry.Code, &entry.Count); err != nil {
			return nil, fmt.Errorf("scan status code distribution: %w", err)
		}
		distribution = append(distribution, entry)
	}
	return distribution, rows.Err()
}
