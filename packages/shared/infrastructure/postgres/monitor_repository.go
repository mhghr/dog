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

type MonitorRepository struct {
	pool *pgxpool.Pool
}

func NewMonitorRepository(pool *pgxpool.Pool) *MonitorRepository {
	return &MonitorRepository{pool: pool}
}

const monitorColumns = `
	m.id::text,
	m.resource_id::text,
	m.monitor_type_id::text,
	m.health_profile_id::text,
	m.created_by::text,
	m.name,
	m.enabled,
	m.config,
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

func scanMonitor(row pgx.Row) (domain.Monitor, error) {
	var m domain.Monitor
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

func (r *MonitorRepository) Create(ctx context.Context, monitor *domain.Monitor) error {
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

	// The monitors.target column is NOT NULL; derive it from the owning
	// resource when the caller did not supply one.
	if monitor.ResourceTarget == "" {
		var target string
		if err := r.pool.QueryRow(ctx, `
			SELECT target FROM resources WHERE id = $1::uuid`,
			monitor.ResourceID,
		).Scan(&target); err != nil {
			return fmt.Errorf("load resource target: %w", err)
		}
		monitor.ResourceTarget = target
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO monitors (
			resource_id, monitor_type_id, health_profile_id, created_by,
			name, enabled, config, severity, target,
			interval_seconds, timeout_millis, retries, last_status, next_run_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING `+monitorColumns,
		monitor.ResourceID, monitor.MonitorTypeID, monitor.HealthProfileID, monitor.CreatedBy,
		monitor.Name, monitor.Enabled, configJSON, monitor.Severity, monitor.ResourceTarget,
		monitor.IntervalSeconds, monitor.TimeoutMillis, monitor.Retries,
		string(monitor.LastStatus), monitor.NextRunAt,
	).Scan(&monitor.ID, &monitor.ResourceID, &monitor.MonitorTypeID, &monitor.HealthProfileID, &monitor.CreatedBy,
		&monitor.Name, &monitor.Enabled, &configJSON, &monitor.Severity,
		&monitor.IntervalSeconds, &monitor.TimeoutMillis, &monitor.Retries,
		&monitor.LastStatus, &monitor.LastCheckedAt, &monitor.NextRunAt,
		&monitor.CreatedAt, &monitor.UpdatedAt)
	if err != nil {
		return fmt.Errorf("create resource monitor: %w", err)
	}
	return nil
}

func (r *MonitorRepository) GetByID(ctx context.Context, id string) (domain.Monitor, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT `+monitorColumns+`,
		       res.target, mt.executor_key
		FROM monitors m
		JOIN resources res ON res.id = m.resource_id
		JOIN monitor_types mt ON mt.id = m.monitor_type_id
		WHERE m.id = $1::uuid`)

	var m domain.Monitor
	var healthProfileID, createdBy *string
	var configJSON []byte
	var executorKey string

	err := row.Scan(&m.ID, &m.ResourceID, &m.MonitorTypeID, &healthProfileID, &createdBy,
		&m.Name, &m.Enabled, &configJSON, &m.Severity,
		&m.IntervalSeconds, &m.TimeoutMillis, &m.Retries,
		&m.LastStatus, &m.LastCheckedAt, &m.NextRunAt,
		&m.CreatedAt, &m.UpdatedAt, &m.ResourceTarget, &executorKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return m, domain.ErrNotFound
		}
		return m, fmt.Errorf("get resource monitor: %w", err)
	}
	m.HealthProfileID = healthProfileID
	m.CreatedBy = createdBy
	json.Unmarshal(configJSON, &m.Configuration)
	m.Type = domain.MonitorType(executorKey)
	return m, nil
}

func (r *MonitorRepository) ListByResource(ctx context.Context, resourceID string) ([]domain.Monitor, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+monitorColumns+`
		FROM monitors m
		WHERE m.resource_id = $1::uuid
		ORDER BY m.created_at DESC`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("list resource monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]domain.Monitor, 0)
	for rows.Next() {
		m, err := scanMonitor(rows)
		if err != nil {
			return nil, fmt.Errorf("scan resource monitor: %w", err)
		}
		monitors = append(monitors, m)
	}
	return monitors, rows.Err()
}

func (r *MonitorRepository) Update(ctx context.Context, monitor *domain.Monitor) error {
	configJSON, _ := json.Marshal(monitor.Configuration)
	if configJSON == nil {
		configJSON = []byte("{}")
	}

	tag, err := r.pool.Exec(ctx, `
		UPDATE monitors SET
			name=$2, enabled=$3, config=$4, severity=$5,
			interval_seconds=$6, timeout_millis=$7, retries=$8, updated_at=NOW()
		WHERE id=$1::uuid`,
		monitor.ID, monitor.Name, monitor.Enabled, configJSON, monitor.Severity,
		monitor.IntervalSeconds, monitor.TimeoutMillis, monitor.Retries)
	if err != nil {
		return fmt.Errorf("update resource monitor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MonitorRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM monitors WHERE id=$1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete resource monitor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MonitorRepository) SetEnabled(ctx context.Context, id string, enabled bool) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE monitors SET enabled=$2, updated_at=NOW() WHERE id=$1::uuid`, id, enabled)
	if err != nil {
		return fmt.Errorf("set resource monitor enabled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

// ClaimDue atomically claims due resource monitors and advances their
// next_run_at so the scheduler can publish probe jobs without double-scheduling.
func (r *MonitorRepository) ClaimDue(ctx context.Context, batchSize int, fn func(domain.Monitor) error) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT `+monitorColumns+`,
		       res.target, res.workspace_id, mt.name AS type_name
		FROM monitors m
		JOIN resources res ON res.id = m.resource_id
		JOIN monitor_types mt ON mt.id = m.monitor_type_id
		WHERE m.resource_id IS NOT NULL
		  AND m.enabled = TRUE AND m.next_run_at <= NOW()
		ORDER BY m.next_run_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, batchSize)
	if err != nil {
		return 0, fmt.Errorf("claim resource monitors: %w", err)
	}

	type claimed struct {
		monitor domain.Monitor
		code    domain.MonitorType
	}
	var claimedAll []claimed

	for rows.Next() {
		var m domain.Monitor
		var healthProfileID, createdBy *string
		var configJSON []byte
		var target, workspaceID, typeName string
		var lastChecked *time.Time

		if err := rows.Scan(&m.ID, &m.ResourceID, &m.MonitorTypeID, &healthProfileID, &createdBy,
			&m.Name, &m.Enabled, &configJSON, &m.Severity,
			&m.IntervalSeconds, &m.TimeoutMillis, &m.Retries,
			&m.LastStatus, &lastChecked, &m.NextRunAt,
			&m.CreatedAt, &m.UpdatedAt, &target, &workspaceID, &typeName); err != nil {
			return 0, fmt.Errorf("scan resource monitor claim: %w", err)
		}

		m.HealthProfileID = healthProfileID
		m.CreatedBy = createdBy
		m.LastCheckedAt = lastChecked
		json.Unmarshal(configJSON, &m.Configuration)
		m.ResourceTarget = target
		if workspaceID != "" {
			m.WorkspaceID = &workspaceID
		}
		m.ProbeType = domain.MonitorTypeCode(typeName)

		claimedAll = append(claimedAll, claimed{monitor: m, code: m.ProbeType})
	}
	rows.Close()

	for _, c := range claimedAll {
		// Advance next_run_at before invoking fn so a slow fn doesn't
		// cause a re-claim in the next tick.
		if _, err := tx.Exec(ctx, `UPDATE monitors SET next_run_at = NOW() + ($2 * INTERVAL '1 second') WHERE id=$1::uuid`,
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

func (r *MonitorRepository) UpdateRunResult(ctx context.Context, monitorID string, status domain.MonitorStatus, checkedAt time.Time) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE monitors SET last_status=$2, last_checked_at=$3, updated_at=NOW()
		WHERE id=$1::uuid`, monitorID, string(status), checkedAt)
	if err != nil {
		return fmt.Errorf("update resource monitor run result: %w", err)
	}
	return nil
}

func (r *MonitorRepository) ListResults(ctx context.Context, monitorID string, limit, offset int) ([]domain.ProbeResult, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM probe_results WHERE monitor_id = $1::uuid`, monitorID,
	).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count resource monitor results: %w", err)
	}

	rows, err := r.pool.Query(ctx, `
		SELECT
			pr.id::text, pr.job_id::text, pr.monitor_id::text,
			COALESCE(pr.probe_location_id::text, ''),
			pr.status::text, pr.success,
			COALESCE(pr.error_code, ''), COALESCE(pr.error_message, ''),
			pr.duration_millis, pr.metrics, pr.attributes, pr.started_at, pr.finished_at,
			COALESCE(pl.name, '')
		FROM probe_results pr
		LEFT JOIN probe_locations pl ON pl.id = pr.probe_location_id
		WHERE pr.monitor_id = $1::uuid
		ORDER BY pr.started_at DESC
		LIMIT $2 OFFSET $3`,
		monitorID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list resource monitor results: %w", err)
	}
	defer rows.Close()

	results := make([]domain.ProbeResult, 0, limit)
	for rows.Next() {
		var (
			result         domain.ProbeResult
			metricsJSON    []byte
			attributesJSON []byte
			probeName      string
		)

		if err := rows.Scan(
			&result.ID, &result.JobID, &result.MonitorID, &result.ProbeLocationID,
			&result.Status, &result.Success,
			&result.ErrorCode, &result.ErrorMessage,
			&result.DurationMillis, &metricsJSON, &attributesJSON,
			&result.StartedAt, &result.FinishedAt, &probeName,
		); err != nil {
			return nil, 0, fmt.Errorf("scan resource monitor result: %w", err)
		}

		if err := json.Unmarshal(metricsJSON, &result.Metrics); err != nil {
			result.Metrics = map[string]any{}
		}
		if err := json.Unmarshal(attributesJSON, &result.Attributes); err != nil {
			result.Attributes = map[string]any{}
		}
		if result.Attributes == nil {
			result.Attributes = map[string]any{}
		}
		if probeName != "" {
			result.Attributes["probe_name"] = probeName
		}

		results = append(results, result)
	}

	return results, total, rows.Err()
}
