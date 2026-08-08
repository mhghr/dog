package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/health"
)

type HealthRepository struct {
	pool *pgxpool.Pool
}

func NewHealthRepository(pool *pgxpool.Pool) *HealthRepository {
	return &HealthRepository{pool: pool}
}

func (r *HealthRepository) ListParameterCatalog(ctx context.Context, monitorType string) ([]health.ParameterDefinition, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT key, monitor_type::text, name, description,
		       data_type::text, unit, direction::text, default_profile::text
		FROM parameter_catalog
		WHERE monitor_type = $1
		ORDER BY key`, monitorType)
	if err != nil {
		return nil, fmt.Errorf("list parameter catalog: %w", err)
	}
	defer rows.Close()

	var defs []health.ParameterDefinition
	for rows.Next() {
		var d health.ParameterDefinition
		if err := rows.Scan(&d.Key, &d.MonitorType, &d.Name, &d.Description,
			&d.DataType, &d.Unit, &d.Direction, &d.DefaultProfile); err != nil {
			return nil, fmt.Errorf("scan parameter catalog: %w", err)
		}
		defs = append(defs, d)
	}

	return defs, rows.Err()
}

func (r *HealthRepository) GetParameterDefinition(ctx context.Context, monitorType, key string) (health.ParameterDefinition, error) {
	var d health.ParameterDefinition
	err := r.pool.QueryRow(ctx, `
		SELECT key, monitor_type::text, name, description,
		       data_type::text, unit, direction::text, default_profile::text
		FROM parameter_catalog
		WHERE monitor_type = $1 AND key = $2`, monitorType, key,
	).Scan(&d.Key, &d.MonitorType, &d.Name, &d.Description,
		&d.DataType, &d.Unit, &d.Direction, &d.DefaultProfile)
	if err == pgx.ErrNoRows {
		return health.ParameterDefinition{}, domain.ErrNotFound
	}
	if err != nil {
		return health.ParameterDefinition{}, fmt.Errorf("get parameter definition: %w", err)
	}
	return d, nil
}

func (r *HealthRepository) ListParameterRules(ctx context.Context, monitorID string) ([]health.ParameterRule, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, monitor_id::text, parameter_key, mode::text, profile::text,
		       aggregation, window_type, window_value,
		       warning_operator, warning_value, error_operator, error_value,
		       recovery_operator, recovery_value,
		       minimum_samples, consecutive_failures, consecutive_successes,
		       missing_data_policy, missed_checks, cooldown_seconds, enabled,
		       created_at, updated_at
		FROM parameter_rules
		WHERE monitor_id = $1
		ORDER BY parameter_key`, monitorID)
	if err != nil {
		return nil, fmt.Errorf("list parameter rules: %w", err)
	}
	defer rows.Close()

	var rules []health.ParameterRule
	for rows.Next() {
		var rule health.ParameterRule
		var profile *string
		if err := rows.Scan(
			&rule.ID, &rule.MonitorID, &rule.ParameterKey, &rule.Mode, &profile,
			&rule.Aggregation, &rule.WindowType, &rule.WindowValue,
			&rule.WarningOperator, &rule.WarningValue, &rule.ErrorOperator, &rule.ErrorValue,
			&rule.RecoveryOperator, &rule.RecoveryValue,
			&rule.MinimumSamples, &rule.ConsecutiveFailures, &rule.ConsecutiveSuccesses,
			&rule.MissingDataPolicy, &rule.MissedChecks, &rule.CooldownSeconds, &rule.Enabled,
			&rule.CreatedAt, &rule.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan parameter rule: %w", err)
		}
		if profile != nil {
			p := health.HealthRuleProfile(*profile)
			rule.Profile = &p
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

func (r *HealthRepository) GetParameterRule(ctx context.Context, monitorID, parameterKey string) (health.ParameterRule, error) {
	var rule health.ParameterRule
	var profile *string
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, monitor_id::text, parameter_key, mode::text, profile::text,
		       aggregation, window_type, window_value,
		       warning_operator, warning_value, error_operator, error_value,
		       recovery_operator, recovery_value,
		       minimum_samples, consecutive_failures, consecutive_successes,
		       missing_data_policy, missed_checks, cooldown_seconds, enabled,
		       created_at, updated_at
		FROM parameter_rules
		WHERE monitor_id = $1 AND parameter_key = $2`, monitorID, parameterKey,
	).Scan(
		&rule.ID, &rule.MonitorID, &rule.ParameterKey, &rule.Mode, &profile,
		&rule.Aggregation, &rule.WindowType, &rule.WindowValue,
		&rule.WarningOperator, &rule.WarningValue, &rule.ErrorOperator, &rule.ErrorValue,
		&rule.RecoveryOperator, &rule.RecoveryValue,
		&rule.MinimumSamples, &rule.ConsecutiveFailures, &rule.ConsecutiveSuccesses,
		&rule.MissingDataPolicy, &rule.MissedChecks, &rule.CooldownSeconds, &rule.Enabled,
		&rule.CreatedAt, &rule.UpdatedAt,
	)
	if err == pgx.ErrNoRows {
		return health.ParameterRule{}, domain.ErrNotFound
	}
	if err != nil {
		return health.ParameterRule{}, fmt.Errorf("get parameter rule: %w", err)
	}
	if profile != nil {
		p := health.HealthRuleProfile(*profile)
		rule.Profile = &p
	}
	return rule, nil
}

func (r *HealthRepository) UpsertParameterRule(ctx context.Context, rule *health.ParameterRule) error {
	var profile *string
	if rule.Profile != nil {
		s := string(*rule.Profile)
		profile = &s
	}

	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO parameter_rules (
			monitor_id, parameter_key, mode, profile,
			aggregation, window_type, window_value,
			warning_operator, warning_value, error_operator, error_value,
			recovery_operator, recovery_value,
			minimum_samples, consecutive_failures, consecutive_successes,
			missing_data_policy, missed_checks, cooldown_seconds, enabled
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT (monitor_id, parameter_key) DO UPDATE SET
			mode = EXCLUDED.mode,
			profile = EXCLUDED.profile,
			aggregation = EXCLUDED.aggregation,
			window_type = EXCLUDED.window_type,
			window_value = EXCLUDED.window_value,
			warning_operator = EXCLUDED.warning_operator,
			warning_value = EXCLUDED.warning_value,
			error_operator = EXCLUDED.error_operator,
			error_value = EXCLUDED.error_value,
			recovery_operator = EXCLUDED.recovery_operator,
			recovery_value = EXCLUDED.recovery_value,
			minimum_samples = EXCLUDED.minimum_samples,
			consecutive_failures = EXCLUDED.consecutive_failures,
			consecutive_successes = EXCLUDED.consecutive_successes,
			missing_data_policy = EXCLUDED.missing_data_policy,
			missed_checks = EXCLUDED.missed_checks,
			cooldown_seconds = EXCLUDED.cooldown_seconds,
			enabled = EXCLUDED.enabled,
			updated_at = NOW()
		RETURNING id::text, created_at, updated_at
	`, rule.MonitorID, rule.ParameterKey, rule.Mode, profile,
		rule.Aggregation, rule.WindowType, rule.WindowValue,
		rule.WarningOperator, rule.WarningValue, rule.ErrorOperator, rule.ErrorValue,
		rule.RecoveryOperator, rule.RecoveryValue,
		rule.MinimumSamples, rule.ConsecutiveFailures, rule.ConsecutiveSuccesses,
		rule.MissingDataPolicy, rule.MissedChecks, rule.CooldownSeconds, rule.Enabled,
	).Scan(&rule.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("upsert parameter rule: %w", err)
	}

	rule.CreatedAt = createdAt
	rule.UpdatedAt = updatedAt
	return nil
}

func (r *HealthRepository) DeleteParameterRule(ctx context.Context, monitorID, parameterKey string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM parameter_rules
		WHERE monitor_id = $1 AND parameter_key = $2`, monitorID, parameterKey)
	if err != nil {
		return fmt.Errorf("delete parameter rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HealthRepository) UpsertHealthState(ctx context.Context, state *health.ParameterHealthState) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO parameter_health_state (
			monitor_id, parameter_key, current_state, current_value,
			evaluated_at, previous_state, state_changed_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (monitor_id, parameter_key) DO UPDATE SET
			current_state = EXCLUDED.current_state,
			current_value = EXCLUDED.current_value,
			evaluated_at = EXCLUDED.evaluated_at,
			previous_state = EXCLUDED.previous_state,
			state_changed_at = EXCLUDED.state_changed_at
	`, state.MonitorID, state.ParameterKey, state.CurrentState, state.CurrentValue,
		state.EvaluatedAt, state.PreviousState, state.StateChangedAt)
	if err != nil {
		return fmt.Errorf("upsert health state: %w", err)
	}
	return nil
}

func (r *HealthRepository) GetHealthState(ctx context.Context, monitorID, parameterKey string) (health.ParameterHealthState, error) {
	var state health.ParameterHealthState
	err := r.pool.QueryRow(ctx, `
		SELECT monitor_id::text, parameter_key,
		       current_state::text, current_value,
		       evaluated_at, previous_state::text, state_changed_at
		FROM parameter_health_state
		WHERE monitor_id = $1 AND parameter_key = $2`, monitorID, parameterKey,
	).Scan(&state.MonitorID, &state.ParameterKey,
		&state.CurrentState, &state.CurrentValue,
		&state.EvaluatedAt, &state.PreviousState, &state.StateChangedAt)
	if err == pgx.ErrNoRows {
		return health.ParameterHealthState{}, domain.ErrNotFound
	}
	if err != nil {
		return health.ParameterHealthState{}, fmt.Errorf("get health state: %w", err)
	}
	return state, nil
}

func (r *HealthRepository) ListHealthStates(ctx context.Context, monitorID string) ([]health.ParameterHealthState, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT monitor_id::text, parameter_key,
		       current_state::text, current_value,
		       evaluated_at, previous_state::text, state_changed_at
		FROM parameter_health_state
		WHERE monitor_id = $1
		ORDER BY parameter_key`, monitorID)
	if err != nil {
		return nil, fmt.Errorf("list health states: %w", err)
	}
	defer rows.Close()

	var states []health.ParameterHealthState
	for rows.Next() {
		var state health.ParameterHealthState
		if err := rows.Scan(&state.MonitorID, &state.ParameterKey,
			&state.CurrentState, &state.CurrentValue,
			&state.EvaluatedAt, &state.PreviousState, &state.StateChangedAt); err != nil {
			return nil, fmt.Errorf("scan health state: %w", err)
		}
		states = append(states, state)
	}

	return states, rows.Err()
}

func (r *HealthRepository) ListNotificationChannels(ctx context.Context) ([]health.HealthNotificationChannel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, name, type, config::text, enabled, created_at, updated_at
		FROM health_notification_channels
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list health notification channels: %w", err)
	}
	defer rows.Close()

	var channels []health.HealthNotificationChannel
	for rows.Next() {
		var ch health.HealthNotificationChannel
		if err := rows.Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan health notification channel: %w", err)
		}
		channels = append(channels, ch)
	}

	return channels, rows.Err()
}

func (r *HealthRepository) GetNotificationChannel(ctx context.Context, id string) (health.HealthNotificationChannel, error) {
	var ch health.HealthNotificationChannel
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, name, type, config::text, enabled, created_at, updated_at
		FROM health_notification_channels
		WHERE id = $1`, id,
	).Scan(&ch.ID, &ch.Name, &ch.Type, &ch.Config, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt)
	if err == pgx.ErrNoRows {
		return health.HealthNotificationChannel{}, domain.ErrNotFound
	}
	if err != nil {
		return health.HealthNotificationChannel{}, fmt.Errorf("get health notification channel: %w", err)
	}
	return ch, nil
}

func (r *HealthRepository) CreateNotificationChannel(ctx context.Context, ch *health.HealthNotificationChannel) error {
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO health_notification_channels (name, type, config, enabled)
		VALUES ($1, $2, $3, $4)
		RETURNING id::text, created_at, updated_at`,
		ch.Name, ch.Type, ch.Config, ch.Enabled,
	).Scan(&ch.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("insert health notification channel: %w", err)
	}

	ch.CreatedAt = createdAt
	ch.UpdatedAt = updatedAt
	return nil
}

func (r *HealthRepository) UpdateNotificationChannel(ctx context.Context, ch *health.HealthNotificationChannel) error {
	var updatedAt time.Time
	err := r.pool.QueryRow(ctx, `
		UPDATE health_notification_channels
		SET name = $2, type = $3, config = $4, enabled = $5, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`,
		ch.ID, ch.Name, ch.Type, ch.Config, ch.Enabled,
	).Scan(&updatedAt)
	if err == pgx.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update health notification channel: %w", err)
	}

	ch.UpdatedAt = updatedAt
	return nil
}

func (r *HealthRepository) DeleteNotificationChannel(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM health_notification_channels
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete health notification channel: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *HealthRepository) ListNotificationPolicies(ctx context.Context, monitorID string) ([]health.NotificationPolicy, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, monitor_id::text, parameter_key, channel_id::text,
		       triggers, delay_seconds, repeat_interval_seconds, cooldown_seconds,
		       enabled, created_at, updated_at
		FROM notification_policies
		WHERE monitor_id = $1
		ORDER BY created_at DESC`, monitorID)
	if err != nil {
		return nil, fmt.Errorf("list notification policies: %w", err)
	}
	defer rows.Close()

	var policies []health.NotificationPolicy
	for rows.Next() {
		var p health.NotificationPolicy
		var dbMonitorID *string
		if err := rows.Scan(
			&p.ID, &dbMonitorID, &p.ParameterKey, &p.ChannelID,
			&p.Triggers, &p.DelaySeconds, &p.RepeatIntervalSeconds, &p.CooldownSeconds,
			&p.Enabled, &p.CreatedAt, &p.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan notification policy: %w", err)
		}
		if dbMonitorID != nil {
			p.MonitorID = dbMonitorID
		}
		policies = append(policies, p)
	}

	return policies, rows.Err()
}

func (r *HealthRepository) GetNotificationPolicy(ctx context.Context, id string) (health.NotificationPolicy, error) {
	var p health.NotificationPolicy
	var dbMonitorID *string
	err := r.pool.QueryRow(ctx, `
		SELECT id::text, monitor_id::text, parameter_key, channel_id::text,
		       triggers, delay_seconds, repeat_interval_seconds, cooldown_seconds,
		       enabled, created_at, updated_at
		FROM notification_policies
		WHERE id = $1`, id,
	).Scan(&p.ID, &dbMonitorID, &p.ParameterKey, &p.ChannelID,
		&p.Triggers, &p.DelaySeconds, &p.RepeatIntervalSeconds, &p.CooldownSeconds,
		&p.Enabled, &p.CreatedAt, &p.UpdatedAt)
	if err == pgx.ErrNoRows {
		return health.NotificationPolicy{}, domain.ErrNotFound
	}
	if err != nil {
		return health.NotificationPolicy{}, fmt.Errorf("get notification policy: %w", err)
	}
	if dbMonitorID != nil {
		p.MonitorID = dbMonitorID
	}
	return p, nil
}

func (r *HealthRepository) CreateNotificationPolicy(ctx context.Context, policy *health.NotificationPolicy) error {
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx, `
		INSERT INTO notification_policies (
			monitor_id, parameter_key, channel_id, triggers,
			delay_seconds, repeat_interval_seconds, cooldown_seconds, enabled
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, created_at, updated_at`,
		policy.MonitorID, policy.ParameterKey, policy.ChannelID, policy.Triggers,
		policy.DelaySeconds, policy.RepeatIntervalSeconds, policy.CooldownSeconds, policy.Enabled,
	).Scan(&policy.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("insert notification policy: %w", err)
	}

	policy.CreatedAt = createdAt
	policy.UpdatedAt = updatedAt
	return nil
}

func (r *HealthRepository) UpdateNotificationPolicy(ctx context.Context, policy *health.NotificationPolicy) error {
	var updatedAt time.Time
	err := r.pool.QueryRow(ctx, `
		UPDATE notification_policies
		SET monitor_id = $2, parameter_key = $3, channel_id = $4, triggers = $5,
		    delay_seconds = $6, repeat_interval_seconds = $7, cooldown_seconds = $8,
		    enabled = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`,
		policy.ID, policy.MonitorID, policy.ParameterKey, policy.ChannelID, policy.Triggers,
		policy.DelaySeconds, policy.RepeatIntervalSeconds, policy.CooldownSeconds, policy.Enabled,
	).Scan(&updatedAt)
	if err == pgx.ErrNoRows {
		return domain.ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("update notification policy: %w", err)
	}

	policy.UpdatedAt = updatedAt
	return nil
}

func (r *HealthRepository) DeleteNotificationPolicy(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM notification_policies
		WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete notification policy: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
