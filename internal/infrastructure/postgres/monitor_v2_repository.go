package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/internal/domain"
)

type MonitorV2Repository struct {
	pool *pgxpool.Pool
}

func NewMonitorV2Repository(pool *pgxpool.Pool) *MonitorV2Repository {
	return &MonitorV2Repository{pool: pool}
}

const monitorV2Columns = `
	m.id::text,
	m.resource_id::text,
	m.monitor_type_id::text,
	m.health_profile_id::text,
	m.created_by::text,
	m.name,
	m.enabled,
	m.configuration,
	m.severity,
	m.interval_seconds,
	m.timeout_millis,
	m.retries,
	m.last_status::text,
	m.last_checked_at,
	m.next_run_at,
	m.created_at,
	m.updated_at
`

func scanMonitorV2(row pgx.Row) (domain.MonitorV2, error) {
	var m domain.MonitorV2
	var healthProfileID, createdBy *string
	var configJSON []byte

	err := row.Scan(&m.ID, &m.ResourceID, &m.MonitorTypeID, &healthProfileID, &createdBy,
		&m.Name, &m.Enabled, &configJSON, &m.Severity,
		&m.IntervalSeconds, &m.TimeoutMillis, &m.Retries,
		&m.LastStatus, &m.LastCheckedAt, &m.NextRunAt,
		&m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return m, err
	}

	m.HealthProfileID = healthProfileID
	m.CreatedBy = createdBy
	json.Unmarshal(configJSON, &m.Configuration)
	return m, nil
}

func (r *MonitorV2Repository) Create(ctx context.Context, monitor *domain.MonitorV2) error {
	configJSON, _ := json.Marshal(monitor.Configuration)
	if configJSON == nil {
		configJSON = []byte("{}")
	}
	if monitor.Severity == "" {
		monitor.Severity = "warning"
	}
	if monitor.LastStatus == "" {
		monitor.LastStatus = domain.StatusUnknown
	}
	if monitor.NextRunAt.IsZero() {
		monitor.NextRunAt = time.Now().UTC()
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO monitors_v2 (
			resource_id, monitor_type_id, health_profile_id, created_by,
			name, enabled, configuration, severity,
			interval_seconds, timeout_millis, retries, last_status, next_run_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING `+monitorV2Columns,
		monitor.ResourceID, monitor.MonitorTypeID, monitor.HealthProfileID, monitor.CreatedBy,
		monitor.Name, monitor.Enabled, configJSON, monitor.Severity,
		monitor.IntervalSeconds, monitor.TimeoutMillis, monitor.Retries,
		string(monitor.LastStatus), monitor.NextRunAt,
	).Scan(&monitor.ID, &monitor.ResourceID, &monitor.MonitorTypeID, &monitor.HealthProfileID, &monitor.CreatedBy,
		&monitor.Name, &monitor.Enabled, &configJSON, &monitor.Severity,
		&monitor.IntervalSeconds, &monitor.TimeoutMillis, &monitor.Retries,
		&monitor.LastStatus, &monitor.LastCheckedAt, &monitor.NextRunAt,
		&monitor.CreatedAt, &monitor.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create monitor v2: %w", err)
	}
	return nil
}

func (r *MonitorV2Repository) GetByID(ctx context.Context, id string) (domain.MonitorV2, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+monitorV2Columns+`,
		       res.target
		FROM monitors_v2 m
		JOIN resources res ON res.id = m.resource_id
		WHERE m.id = $1::uuid`)

	var m domain.MonitorV2
	var healthProfileID, createdBy *string
	var configJSON []byte

	err := row.Scan(&m.ID, &m.ResourceID, &m.MonitorTypeID, &healthProfileID, &createdBy,
		&m.Name, &m.Enabled, &configJSON, &m.Severity,
		&m.IntervalSeconds, &m.TimeoutMillis, &m.Retries,
		&m.LastStatus, &m.LastCheckedAt, &m.NextRunAt,
		&m.CreatedAt, &m.UpdatedAt, &m.ResourceTarget)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m, domain.ErrNotFound
		}
		return m, fmt.Errorf("get monitor v2: %w", err)
	}
	m.HealthProfileID = healthProfileID
	m.CreatedBy = createdBy
	json.Unmarshal(configJSON, &m.Configuration)
	return m, nil
}

func (r *MonitorV2Repository) ListByResource(ctx context.Context, resourceID string) ([]domain.MonitorV2, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+monitorV2Columns+`
		FROM monitors_v2 m
		WHERE m.resource_id = $1::uuid
		ORDER BY m.created_at DESC`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list monitors v2: %w", err)
	}
	defer rows.Close()

	monitors := make([]domain.MonitorV2, 0)
	for rows.Next() {
		m, err := scanMonitorV2(rows)
		if err != nil {
			return nil, fmt.Errorf("scan monitor v2: %w", err)
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

func (r *MonitorV2Repository) Update(ctx context.Context, monitor *domain.MonitorV2) error {
	configJSON, _ := json.Marshal(monitor.Configuration)
	if configJSON == nil {
		configJSON = []byte("{}")
	}

	tag, err := r.pool.Exec(ctx, `
		UPDATE monitors_v2 SET
			name=$2, enabled=$3, configuration=$4, severity=$5,
			interval_seconds=$6, timeout_millis=$7, retries=$8, updated_at=NOW()
		WHERE id=$1::uuid`,
		monitor.ID, monitor.Name, monitor.Enabled, configJSON, monitor.Severity,
		monitor.IntervalSeconds, monitor.TimeoutMillis, monitor.Retries)
	if err != nil {
		return fmt.Errorf("update monitor v2: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MonitorV2Repository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM monitors_v2 WHERE id=$1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete monitor v2: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MonitorV2Repository) SetEnabled(ctx context.Context, id string, enabled bool) error {
	tag, err := r.pool.Exec(ctx, `UPDATE monitors_v2 SET enabled=$2, updated_at=NOW() WHERE id=$1::uuid`, id, enabled)
	if err != nil {
		return fmt.Errorf("set monitor v2 enabled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ClaimDue atomically claims due monitors and advances their next_run_at so
// the scheduler can publish probe jobs without double-scheduling.
func (r *MonitorV2Repository) ClaimDue(ctx context.Context, batchSize int, fn func(domain.MonitorV2) error) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT `+monitorV2Columns+`,
		       res.target, mt.name AS type_name
		FROM monitors_v2 m
		JOIN resources res ON res.id = m.resource_id
		JOIN monitor_types mt ON mt.id = m.monitor_type_id
		WHERE m.enabled = TRUE AND m.next_run_at <= NOW()
		ORDER BY m.next_run_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, batchSize)
	if err != nil {
		return 0, fmt.Errorf("claim monitors v2: %w", err)
	}

	type claimed struct {
		monitor domain.MonitorV2
		code    domain.MonitorType
	}
	var claimedAll []claimed

	for rows.Next() {
		var m domain.MonitorV2
		var healthProfileID, createdBy *string
		var configJSON []byte
		var target, typeName string
		var lastChecked *time.Time

		if err := rows.Scan(&m.ID, &m.ResourceID, &m.MonitorTypeID, &healthProfileID, &createdBy,
			&m.Name, &m.Enabled, &configJSON, &m.Severity,
			&m.IntervalSeconds, &m.TimeoutMillis, &m.Retries,
			&m.LastStatus, &lastChecked, &m.NextRunAt,
			&m.CreatedAt, &m.UpdatedAt, &target, &typeName); err != nil {
			return 0, fmt.Errorf("scan monitor v2 claim: %w", err)
		}

		m.HealthProfileID = healthProfileID
		m.CreatedBy = createdBy
		m.LastCheckedAt = lastChecked
		json.Unmarshal(configJSON, &m.Configuration)
		m.ResourceTarget = target
		m.ProbeType = domain.MonitorTypeCode(typeName)

		claimedAll = append(claimedAll, claimed{monitor: m, code: m.ProbeType})
	}
	rows.Close()

	for _, c := range claimedAll {
		// Advance next_run_at before invoking fn so a slow fn doesn't
		// cause a re-claim in the next tick.
		if _, err := tx.Exec(ctx, `UPDATE monitors_v2 SET next_run_at = NOW() + ($2 * INTERVAL '1 second') WHERE id=$1::uuid`,
			c.monitor.ID, c.monitor.IntervalSeconds); err != nil {
			return 0, err
		}
		if err := fn(c.monitor); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(claimedAll), nil
}

func (r *MonitorV2Repository) UpdateRunResult(ctx context.Context, monitorID string, status domain.MonitorStatus, checkedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE monitors_v2 SET last_status=$2, last_checked_at=$3, updated_at=NOW()
		WHERE id=$1::uuid`, monitorID, string(status), checkedAt)
	if err != nil {
		return fmt.Errorf("update monitor v2 run result: %w", err)
	}
	return nil
}

func (r *MonitorV2Repository) ListV2Results(ctx context.Context, monitorID string, limit, offset int) ([]domain.ProbeResult, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM probe_results WHERE monitor_v2_id = $1::uuid`, monitorID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count v2 probe results: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text, job_id::text, monitor_v2_id::text,
			COALESCE(probe_location_id::text, ''),
			status::text, success,
			COALESCE(error_code, ''), COALESCE(error_message, ''),
			duration_millis, metrics, attributes, started_at, finished_at
		FROM probe_results
		WHERE monitor_v2_id = $1::uuid
		ORDER BY started_at DESC
		LIMIT $2 OFFSET $3`,
		monitorID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list v2 probe results: %w", err)
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
			return nil, 0, fmt.Errorf("scan v2 probe result: %w", err)
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
