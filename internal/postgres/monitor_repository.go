package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/internal/domain"
)

type MonitorRepository struct {
	pool *pgxpool.Pool
}

func NewMonitorRepository(pool *pgxpool.Pool) *MonitorRepository {
	return &MonitorRepository{pool: pool}
}

const monitorColumns = `
	id::text,
	name,
	type::text,
	target,
	interval_seconds,
	timeout_millis,
	retries,
	enabled,
	config,
	last_status::text,
	last_checked_at,
	next_run_at,
	created_at,
	updated_at
`

func (r *MonitorRepository) Create(ctx context.Context, monitor *domain.Monitor) error {
	configJSON, err := json.Marshal(monitor.Config)
	if err != nil {
		return fmt.Errorf("marshal monitor config: %w", err)
	}

	row := r.pool.QueryRow(ctx, `
		INSERT INTO monitors (
			name, type, target, interval_seconds, timeout_millis,
			retries, enabled, config, last_status, next_run_at
		)
		VALUES ($1, $2::monitor_type, $3, $4, $5, $6, $7, $8::jsonb, $9::monitor_status, NOW())
		RETURNING `+monitorColumns,
		monitor.Name,
		string(monitor.Type),
		monitor.Target,
		monitor.IntervalSeconds,
		monitor.TimeoutMillis,
		monitor.Retries,
		monitor.Enabled,
		configJSON,
		string(monitor.LastStatus),
	)

	created, err := scanMonitor(row)
	if err != nil {
		return err
	}

	*monitor = created
	return nil
}

func (r *MonitorRepository) GetByID(ctx context.Context, id string) (domain.Monitor, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+monitorColumns+` FROM monitors WHERE id = $1::uuid`, id)

	monitor, err := scanMonitor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Monitor{}, domain.ErrNotFound
	}

	return monitor, err
}

var allowedSortColumns = map[string]string{
	"name":            "name",
	"status":          "last_status",
	"last_checked_at": "last_checked_at",
	"next_run_at":     "next_run_at",
	"created_at":      "created_at",
	"interval":        "interval_seconds",
}

func (r *MonitorRepository) List(ctx context.Context, filter domain.MonitorListFilter) ([]domain.MonitorWithLastResult, int, error) {
	where := make([]string, 0, 3)
	args := make([]any, 0, 4)

	if filter.Type != nil {
		args = append(args, string(*filter.Type))
		where = append(where, fmt.Sprintf("m.type = $%d::monitor_type", len(args)))
	}

	if filter.Status != nil {
		args = append(args, string(*filter.Status))
		where = append(where, fmt.Sprintf("m.last_status = $%d::monitor_status", len(args)))
	}

	if search := strings.TrimSpace(filter.Search); search != "" {
		args = append(args, "%"+escapeLike(search)+"%")
		where = append(where, fmt.Sprintf("(m.name ILIKE $%d OR m.target ILIKE $%d)", len(args), len(args)))
	}

	whereClause := ""
	if len(where) > 0 {
		whereClause = " WHERE " + strings.Join(where, " AND ")
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM monitors m"+whereClause, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count monitors: %w", err)
	}

	sortColumn, ok := allowedSortColumns[filter.Sort]
	if !ok {
		sortColumn = "created_at"
	}

	order := "DESC"
	if strings.EqualFold(filter.Order, "asc") {
		order = "ASC"
	}

	args = append(args, filter.PageSize)
	limitArg := len(args)
	args = append(args, (filter.Page-1)*filter.PageSize)
	offsetArg := len(args)

	query := fmt.Sprintf(`
		SELECT %s,
			lr.success,
			lr.duration_millis,
			lr.error_code,
			lr.metrics
		FROM monitors m
		LEFT JOIN LATERAL (
			SELECT success, duration_millis, error_code, metrics
			FROM probe_results pr
			WHERE pr.monitor_id = m.id
			ORDER BY pr.started_at DESC
			LIMIT 1
		) lr ON TRUE
		%s
		ORDER BY m.%s %s, m.id
		LIMIT $%d OFFSET $%d`,
		prefixedMonitorColumns("m"), whereClause, sortColumn, order, limitArg, offsetArg)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]domain.MonitorWithLastResult, 0)
	for rows.Next() {
		var (
			item        domain.MonitorWithLastResult
			configJSON  []byte
			success     *bool
			duration    *int64
			errorCode   *string
			metricsJSON []byte
		)

		if err := rows.Scan(
			&item.ID, &item.Name, &item.Type, &item.Target,
			&item.IntervalSeconds, &item.TimeoutMillis, &item.Retries, &item.Enabled,
			&configJSON, &item.LastStatus, &item.LastCheckedAt, &item.NextRunAt,
			&item.CreatedAt, &item.UpdatedAt,
			&success, &duration, &errorCode, &metricsJSON,
		); err != nil {
			return nil, 0, fmt.Errorf("scan monitor row: %w", err)
		}

		if err := json.Unmarshal(configJSON, &item.Config); err != nil {
			item.Config = map[string]any{}
		}

		if success != nil && duration != nil {
			item.LastResult = &domain.LastResultSummary{
				Success:        *success,
				DurationMillis: *duration,
				ErrorCode:      errorCode,
				Metrics:        map[string]any{},
			}
			_ = json.Unmarshal(metricsJSON, &item.LastResult.Metrics)
		}

		monitors = append(monitors, item)
	}

	return monitors, total, rows.Err()
}

func (r *MonitorRepository) Update(ctx context.Context, monitor *domain.Monitor) error {
	configJSON, err := json.Marshal(monitor.Config)
	if err != nil {
		return fmt.Errorf("marshal monitor config: %w", err)
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE monitors
		SET
			name = $2,
			type = $3::monitor_type,
			target = $4,
			interval_seconds = $5,
			timeout_millis = $6,
			retries = $7,
			enabled = $8,
			config = $9::jsonb,
			next_run_at = LEAST(next_run_at, NOW() + ($5 * INTERVAL '1 second')),
			updated_at = NOW()
		WHERE id = $1::uuid
		RETURNING `+monitorColumns,
		monitor.ID,
		monitor.Name,
		string(monitor.Type),
		monitor.Target,
		monitor.IntervalSeconds,
		monitor.TimeoutMillis,
		monitor.Retries,
		monitor.Enabled,
		configJSON,
	)

	updated, err := scanMonitor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	if err != nil {
		return err
	}

	*monitor = updated
	return nil
}

func (r *MonitorRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM monitors WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete monitor: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *MonitorRepository) SetEnabled(ctx context.Context, id string, enabled bool) (domain.Monitor, error) {
	query := `
		UPDATE monitors
		SET
			enabled = $2,
			last_status = CASE WHEN $2 THEN 'unknown'::monitor_status ELSE 'paused'::monitor_status END,
			next_run_at = CASE WHEN $2 THEN NOW() ELSE next_run_at END,
			updated_at = NOW()
		WHERE id = $1::uuid
		RETURNING ` + monitorColumns

	row := r.pool.QueryRow(ctx, query, id, enabled)

	monitor, err := scanMonitor(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Monitor{}, domain.ErrNotFound
	}

	return monitor, err
}

func (r *MonitorRepository) ClaimDue(ctx context.Context, limit int, publish func(domain.Monitor) error) (int, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin scheduler transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	rows, err := tx.Query(ctx, `
		SELECT `+monitorColumns+`
		FROM monitors
		WHERE enabled = TRUE
		  AND next_run_at <= NOW()
		ORDER BY next_run_at
		LIMIT $1
		FOR UPDATE SKIP LOCKED`, limit)
	if err != nil {
		return 0, fmt.Errorf("select due monitors: %w", err)
	}

	dueMonitors := make([]domain.Monitor, 0)
	for rows.Next() {
		monitor, err := scanMonitorFromRows(rows)
		if err != nil {
			rows.Close()
			return 0, err
		}
		dueMonitors = append(dueMonitors, monitor)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	publishedIDs := make([]string, 0, len(dueMonitors))
	for _, monitor := range dueMonitors {
		if err := publish(monitor); err != nil {
			// Leave next_run_at untouched so the monitor is retried on
			// the next tick; the scheduler records the error metric.
			continue
		}
		publishedIDs = append(publishedIDs, monitor.ID)
	}

	if len(publishedIDs) > 0 {
		if _, err := tx.Exec(ctx, `
			UPDATE monitors
			SET
				next_run_at = NOW() + make_interval(secs => interval_seconds),
				updated_at = NOW()
			WHERE id = ANY($1::uuid[])`, publishedIDs); err != nil {
			return 0, fmt.Errorf("advance next_run_at: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit scheduler transaction: %w", err)
	}

	return len(publishedIDs), nil
}

func prefixedMonitorColumns(alias string) string {
	columns := strings.Split(monitorColumns, ",")
	prefixed := make([]string, 0, len(columns))
	for _, column := range columns {
		prefixed = append(prefixed, alias+"."+strings.TrimSpace(column))
	}
	return strings.Join(prefixed, ", ")
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanMonitor(row rowScanner) (domain.Monitor, error) {
	var (
		monitor    domain.Monitor
		configJSON []byte
	)

	if err := row.Scan(
		&monitor.ID, &monitor.Name, &monitor.Type, &monitor.Target,
		&monitor.IntervalSeconds, &monitor.TimeoutMillis, &monitor.Retries, &monitor.Enabled,
		&configJSON, &monitor.LastStatus, &monitor.LastCheckedAt, &monitor.NextRunAt,
		&monitor.CreatedAt, &monitor.UpdatedAt,
	); err != nil {
		return domain.Monitor{}, err
	}

	if err := json.Unmarshal(configJSON, &monitor.Config); err != nil {
		monitor.Config = map[string]any{}
	}

	return monitor, nil
}

func scanMonitorFromRows(rows pgx.Rows) (domain.Monitor, error) {
	monitor, err := scanMonitor(rows)
	if err != nil {
		return domain.Monitor{}, fmt.Errorf("scan monitor: %w", err)
	}
	return monitor, nil
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}
