package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/internal/alerting"
	"monitoring-platform/internal/domain"
)

type ChannelRepository struct {
	pool *pgxpool.Pool
}

func NewChannelRepository(pool *pgxpool.Pool) *ChannelRepository {
	return &ChannelRepository{pool: pool}
}

func (r *ChannelRepository) ListByIDs(ctx context.Context, ids []string) ([]alerting.NotificationChannel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text,
			organization_id::text,
			name,
			type,
			config,
			enabled,
			created_at,
			updated_at
		FROM notification_channels
		WHERE id = ANY($1::uuid[])
		  AND enabled = TRUE`, ids)
	if err != nil {
		return nil, fmt.Errorf("list notification channels by ids: %w", err)
	}
	defer rows.Close()

	var channels []alerting.NotificationChannel
	for rows.Next() {
		var (
			ch       alerting.NotificationChannel
			configJSON []byte
		)

		if err := rows.Scan(
			&ch.ID, &ch.OrganizationID, &ch.Name, &ch.Type,
			&configJSON, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}

		if err := json.Unmarshal(configJSON, &ch.Config); err != nil {
			ch.Config = map[string]string{}
		}
		channels = append(channels, ch)
	}

	return channels, rows.Err()
}

// ── CRUD methods for the API handler ──

func (r *ChannelRepository) ListByOrg(ctx context.Context, orgID string) ([]alerting.NotificationChannel, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT
			id::text,
			organization_id::text,
			name,
			type,
			config,
			enabled,
			created_at,
			updated_at
		FROM notification_channels
		WHERE organization_id = $1
		ORDER BY created_at DESC`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list notification channels: %w", err)
	}
	defer rows.Close()

	var channels []alerting.NotificationChannel
	for rows.Next() {
		var (
			ch         alerting.NotificationChannel
			configJSON []byte
		)

		if err := rows.Scan(
			&ch.ID, &ch.OrganizationID, &ch.Name, &ch.Type,
			&configJSON, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan channel: %w", err)
		}

		if err := json.Unmarshal(configJSON, &ch.Config); err != nil {
			ch.Config = map[string]string{}
		}
		channels = append(channels, ch)
	}

	return channels, rows.Err()
}

func (r *ChannelRepository) CreateChannel(ctx context.Context, ch *alerting.NotificationChannel) error {
	configJSON, err := json.Marshal(ch.Config)
	if err != nil {
		return fmt.Errorf("marshal channel config: %w", err)
	}

	var createdAt, updatedAt time.Time
	err = r.pool.QueryRow(ctx, `
		INSERT INTO notification_channels (
			organization_id, name, type, config, enabled
		)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id::text, created_at, updated_at`,
		ch.OrganizationID, ch.Name, ch.Type, configJSON, ch.Enabled,
	).Scan(&ch.ID, &createdAt, &updatedAt)
	if err != nil {
		return fmt.Errorf("insert notification channel: %w", err)
	}

	ch.CreatedAt = createdAt
	ch.UpdatedAt = updatedAt
	return nil
}

func (r *ChannelRepository) GetChannel(ctx context.Context, id string) (alerting.NotificationChannel, error) {
	var ch alerting.NotificationChannel
	var configJSON []byte

	err := r.pool.QueryRow(ctx, `
		SELECT
			id::text,
			organization_id::text,
			name,
			type,
			config,
			enabled,
			created_at,
			updated_at
		FROM notification_channels
		WHERE id = $1`, id,
	).Scan(
		&ch.ID, &ch.OrganizationID, &ch.Name, &ch.Type,
		&configJSON, &ch.Enabled, &ch.CreatedAt, &ch.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return alerting.NotificationChannel{}, domain.ErrNotFound
		}
		return alerting.NotificationChannel{}, fmt.Errorf("get channel: %w", err)
	}

	if err := json.Unmarshal(configJSON, &ch.Config); err != nil {
		ch.Config = map[string]string{}
	}
	return ch, nil
}
