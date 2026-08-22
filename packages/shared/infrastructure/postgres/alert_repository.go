package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type AlertRepository struct {
	pool *pgxpool.Pool
}

func NewAlertRepository(pool *pgxpool.Pool) *AlertRepository {
	return &AlertRepository{pool: pool}
}

func scanAlert(row pgx.Row) (domain.Alert, error) {
	var alert domain.Alert
	var openedAt, resolvedAt *time.Time

	err := row.Scan(
		&alert.ID, &alert.OrganizationID, &alert.PolicyID, &alert.MonitorID,
		&alert.State, &alert.Severity,
		&alert.Title, &alert.Description, &alert.DedupKey,
		&alert.ConsecutiveFailures, &alert.ConsecutiveSuccesses,
		&openedAt, &resolvedAt, &alert.CreatedAt,
	)
	if err != nil {
		return domain.Alert{}, err
	}
	alert.OpenedAt = openedAt
	alert.ResolvedAt = resolvedAt
	return alert, nil
}

const alertColumns = `
	id::text,
	organization_id::text,
	policy_id::text,
	monitor_id::text,
	state,
	severity,
	title,
	description,
	dedup_key,
	consecutive_failures,
	consecutive_successes,
	opened_at,
	resolved_at,
	created_at
`

func (r *AlertRepository) ListActivePolicies(ctx context.Context, monitorID string) ([]domain.AlertPolicy, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text,
			organization_id::text,
			COALESCE(workspace_id::text, ''),
			name,
			scope,
			conditions,
			severity,
			opening_failures,
			resolving_successes,
			cooldown_seconds,
			renotify_seconds,
			enabled,
			channel_ids,
			created_at,
			updated_at
		FROM alert_policies
		WHERE enabled = TRUE
		  AND (
			scope IS NULL
			OR scope = '{}'
			OR scope->>'monitor_ids' IS NULL
			OR scope->'monitor_ids' @> to_jsonb($1::text)
		  )`, monitorID)
	if err != nil {
		return nil, fmt.Errorf("list active alert policies: %w", err)
	}
	defer rows.Close()

	var policies []domain.AlertPolicy
	for rows.Next() {
		var (
			p          domain.AlertPolicy
			scopeJSON  []byte
			condJSON   []byte
			channelIDs []string
		)

		if err := rows.Scan(
			&p.ID, &p.OrganizationID, &p.WorkspaceID, &p.Name,
			&scopeJSON, &condJSON, &p.Severity,
			&p.OpeningFailures, &p.ResolvingSuccesses,
			&p.CooldownSeconds, &p.RenotifySeconds,
			&p.Enabled, &channelIDs,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan alert policy: %w", err)
		}

		if err := json.Unmarshal(scopeJSON, &p.Scope); err != nil {
			p.Scope = domain.AlertPolicyScope{}
		}
		if err := json.Unmarshal(condJSON, &p.Conditions); err != nil {
			p.Conditions = domain.AlertConditions{}
		}
		p.ChannelIDs = channelIDs
		policies = append(policies, p)
	}

	return policies, rows.Err()
}

func (r *AlertRepository) FindByDedup(ctx context.Context, dedupKey string) (domain.Alert, error) {
	alert, err := scanAlert(r.pool.QueryRow(ctx, `
		SELECT `+alertColumns+`
		FROM alert_events
		WHERE dedup_key = $1`, dedupKey))

	if err == pgx.ErrNoRows {
		return domain.Alert{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Alert{}, fmt.Errorf("find alert by dedup: %w", err)
	}
	return alert, nil
}

func (r *AlertRepository) UpsertAlert(ctx context.Context, alert *domain.Alert) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO alert_events (
			id, organization_id, policy_id, monitor_id,
			state, severity, title, description, dedup_key,
			consecutive_failures, consecutive_successes,
			opened_at, resolved_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		ON CONFLICT (dedup_key) DO UPDATE SET
			state = EXCLUDED.state,
			severity = EXCLUDED.severity,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			consecutive_failures = EXCLUDED.consecutive_failures,
			consecutive_successes = EXCLUDED.consecutive_successes,
			opened_at = EXCLUDED.opened_at,
			resolved_at = EXCLUDED.resolved_at
	`, alert.ID, alert.OrganizationID, alert.PolicyID, alert.MonitorID,
		alert.State, alert.Severity, alert.Title, alert.Description, alert.DedupKey,
		alert.ConsecutiveFailures, alert.ConsecutiveSuccesses,
		alert.OpenedAt, alert.ResolvedAt)
	if err != nil {
		return fmt.Errorf("upsert alert: %w", err)
	}

	return nil
}

func (r *AlertRepository) ListFiring(ctx context.Context) ([]domain.Alert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+alertColumns+`
		FROM alert_events
		WHERE state = 'firing'
		ORDER BY opened_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list firing alerts: %w", err)
	}
	defer rows.Close()

	return scanAlerts(rows)
}

func (r *AlertRepository) RecordNotification(ctx context.Context, alertID string) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE alert_events
		SET last_notified_at = NOW(),
		    notification_count = notification_count + 1
		WHERE id = $1`, alertID)
	if err != nil {
		return fmt.Errorf("record notification: %w", err)
	}
	return nil
}

// ── CRUD methods for the API handler ──

func (r *AlertRepository) ListPolicies(ctx context.Context, orgID string) ([]domain.AlertPolicy, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text,
			organization_id::text,
			COALESCE(workspace_id::text, ''),
			name,
			scope,
			conditions,
			severity,
			opening_failures,
			resolving_successes,
			cooldown_seconds,
			renotify_seconds,
			enabled,
			channel_ids,
			created_at,
			updated_at
		FROM alert_policies
		WHERE organization_id = $1
		ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list alert policies: %w", err)
	}
	defer rows.Close()

	var policies []domain.AlertPolicy
	for rows.Next() {
		var (
			p          domain.AlertPolicy
			scopeJSON  []byte
			condJSON   []byte
			channelIDs []string
		)

		if err := rows.Scan(
			&p.ID, &p.OrganizationID, &p.WorkspaceID, &p.Name,
			&scopeJSON, &condJSON, &p.Severity,
			&p.OpeningFailures, &p.ResolvingSuccesses,
			&p.CooldownSeconds, &p.RenotifySeconds,
			&p.Enabled, &channelIDs,
			&p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan alert policy: %w", err)
		}

		if err := json.Unmarshal(scopeJSON, &p.Scope); err != nil {
			p.Scope = domain.AlertPolicyScope{}
		}
		if err := json.Unmarshal(condJSON, &p.Conditions); err != nil {
			p.Conditions = domain.AlertConditions{}
		}
		p.ChannelIDs = channelIDs
		policies = append(policies, p)
	}

	return policies, rows.Err()
}

func (r *AlertRepository) CreatePolicy(ctx context.Context, policy *domain.AlertPolicy) error {
	scopeJSON, err := json.Marshal(policy.Scope)
	if err != nil {
		return fmt.Errorf("marshal scope: %w", err)
	}
	condJSON, err := json.Marshal(policy.Conditions)
	if err != nil {
		return fmt.Errorf("marshal conditions: %w", err)
	}

	err = r.pool.QueryRow(ctx, `
		INSERT INTO alert_policies (
			organization_id, workspace_id, name, scope, conditions,
			severity, opening_failures, resolving_successes,
			cooldown_seconds, renotify_seconds, enabled, channel_ids
		)
		VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id::text, created_at, updated_at`,
		policy.OrganizationID, policy.WorkspaceID, policy.Name,
		scopeJSON, condJSON, policy.Severity,
		policy.OpeningFailures, policy.ResolvingSuccesses,
		policy.CooldownSeconds, policy.RenotifySeconds,
		policy.Enabled, policy.ChannelIDs,
	).Scan(&policy.ID, &policy.CreatedAt, &policy.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert alert policy: %w", err)
	}

	return nil
}

func (r *AlertRepository) ListAlerts(ctx context.Context, orgID string) ([]domain.Alert, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+alertColumns+`
		FROM alert_events
		WHERE organization_id = $1
		ORDER BY created_at DESC
		LIMIT 200`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	return scanAlerts(rows)
}

func (r *AlertRepository) GetAlert(ctx context.Context, alertID string) (domain.Alert, error) {
	alert, err := scanAlert(r.pool.QueryRow(ctx, `
		SELECT `+alertColumns+`
		FROM alert_events
		WHERE id = $1`, alertID))

	if err == pgx.ErrNoRows {
		return domain.Alert{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Alert{}, fmt.Errorf("get alert: %w", err)
	}
	return alert, nil
}

func scanAlerts(rows pgx.Rows) ([]domain.Alert, error) {
	var alerts []domain.Alert
	for rows.Next() {
		var (
			alert      domain.Alert
			openedAt   *time.Time
			resolvedAt *time.Time
		)

		if err := rows.Scan(
			&alert.ID, &alert.OrganizationID, &alert.PolicyID, &alert.MonitorID,
			&alert.State, &alert.Severity,
			&alert.Title, &alert.Description, &alert.DedupKey,
			&alert.ConsecutiveFailures, &alert.ConsecutiveSuccesses,
			&openedAt, &resolvedAt, &alert.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan alert row: %w", err)
		}

		alert.OpenedAt = openedAt
		alert.ResolvedAt = resolvedAt
		alerts = append(alerts, alert)
	}
	return alerts, rows.Err()
}
