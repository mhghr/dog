package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type ResourceHealthRepository struct {
	pool *pgxpool.Pool
}

func NewResourceHealthRepository(pool *pgxpool.Pool) *ResourceHealthRepository {
	return &ResourceHealthRepository{pool: pool}
}

func (r *ResourceHealthRepository) UpsertState(ctx context.Context, state *domain.ResourceHealthState) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO resource_health_state (resource_id, state, score, active_alerts, active_warnings, last_evaluated_at, state_changed_at)
		VALUES ($1::uuid, $2::resource_health, $3, $4, $5, $6, $7)
		ON CONFLICT (resource_id) DO UPDATE SET
			state = EXCLUDED.state,
			score = EXCLUDED.score,
			active_alerts = EXCLUDED.active_alerts,
			active_warnings = EXCLUDED.active_warnings,
			last_evaluated_at = EXCLUDED.last_evaluated_at,
			state_changed_at = EXCLUDED.state_changed_at`,
		state.ResourceID, string(state.State), state.Score, state.ActiveAlerts,
		state.ActiveWarnings, state.LastEvaluatedAt, state.StateChangedAt)
	if err != nil {
		return fmt.Errorf("upsert resource health state: %w", err)
	}
	return nil
}

func (r *ResourceHealthRepository) GetState(ctx context.Context, resourceID string) (domain.ResourceHealthState, error) {
	var state domain.ResourceHealthState
	var stateStr string
	err := r.pool.QueryRow(ctx, `
		SELECT resource_id::text, state::text, score, active_alerts, active_warnings,
		       last_evaluated_at, state_changed_at
		FROM resource_health_state WHERE resource_id = $1::uuid`, resourceID,
	).Scan(&state.ResourceID, &stateStr, &state.Score, &state.ActiveAlerts,
		&state.ActiveWarnings, &state.LastEvaluatedAt, &state.StateChangedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ResourceHealthState{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.ResourceHealthState{}, fmt.Errorf("get resource health state: %w", err)
	}
	state.State = domain.ResourceHealth(stateStr)
	return state, nil
}

func (r *ResourceHealthRepository) ListByWorkspace(ctx context.Context, workspaceID string) ([]domain.ResourceHealthState, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT rhs.resource_id::text, rhs.state::text, rhs.score, rhs.active_alerts, rhs.active_warnings,
		       rhs.last_evaluated_at, rhs.state_changed_at
		FROM resource_health_state rhs
		JOIN resources r ON r.id = rhs.resource_id
		WHERE r.workspace_id = $1::uuid
		ORDER BY rhs.score ASC`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list resource health states: %w", err)
	}
	defer rows.Close()

	var states []domain.ResourceHealthState
	for rows.Next() {
		var s domain.ResourceHealthState
		var stateStr string
		if err := rows.Scan(&s.ResourceID, &stateStr, &s.Score, &s.ActiveAlerts,
			&s.ActiveWarnings, &s.LastEvaluatedAt, &s.StateChangedAt); err != nil {
			return nil, fmt.Errorf("scan health state: %w", err)
		}
		s.State = domain.ResourceHealth(stateStr)
		states = append(states, s)
	}
	return states, rows.Err()
}
