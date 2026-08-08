package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type ProbeAssignmentRepository struct {
	pool *pgxpool.Pool
}

func NewProbeAssignmentRepository(pool *pgxpool.Pool) *ProbeAssignmentRepository {
	return &ProbeAssignmentRepository{pool: pool}
}

func (r *ProbeAssignmentRepository) Create(ctx context.Context, assign *domain.ProbeAssignment) error {
	err := r.pool.QueryRow(ctx, `
		INSERT INTO probe_assignments (monitor_id, probe_id, priority)
		VALUES ($1::uuid, $2::uuid, $3)
		RETURNING id::text, created_at`,
		assign.MonitorID, assign.ProbeID, assign.Priority,
	).Scan(&assign.ID, &assign.CreatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert probe assignment: %w", err)
	}
	return nil
}

func (r *ProbeAssignmentRepository) ListByMonitor(ctx context.Context, monitorID string) ([]domain.ProbeAssignment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, monitor_id::text, probe_id::text, priority, created_at
		FROM probe_assignments WHERE monitor_id = $1::uuid ORDER BY priority DESC`, monitorID)
	if err != nil {
		return nil, fmt.Errorf("list probe assignments by monitor: %w", err)
	}
	defer rows.Close()

	return scanProbeAssignments(rows)
}

func (r *ProbeAssignmentRepository) ListByProbe(ctx context.Context, probeID string) ([]domain.ProbeAssignment, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, monitor_id::text, probe_id::text, priority, created_at
		FROM probe_assignments WHERE probe_id = $1::uuid ORDER BY priority DESC`, probeID)
	if err != nil {
		return nil, fmt.Errorf("list probe assignments by probe: %w", err)
	}
	defer rows.Close()

	return scanProbeAssignments(rows)
}

func (r *ProbeAssignmentRepository) Delete(ctx context.Context, monitorID, probeID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM probe_assignments WHERE monitor_id = $1::uuid AND probe_id = $2::uuid`,
		monitorID, probeID)
	if err != nil {
		return fmt.Errorf("delete probe assignment: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func scanProbeAssignments(rows pgx.Rows) ([]domain.ProbeAssignment, error) {
	var assignments []domain.ProbeAssignment
	for rows.Next() {
		var a domain.ProbeAssignment
		if err := rows.Scan(&a.ID, &a.MonitorID, &a.ProbeID, &a.Priority, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan probe assignment: %w", err)
		}
		assignments = append(assignments, a)
	}
	return assignments, rows.Err()
}
