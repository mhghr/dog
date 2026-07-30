package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/internal/domain"
)

type AgentConfigRepository struct {
	pool *pgxpool.Pool
}

func NewAgentConfigRepository(pool *pgxpool.Pool) *AgentConfigRepository {
	return &AgentConfigRepository{pool: pool}
}

func (r *AgentConfigRepository) Create(ctx context.Context, config *domain.AgentConfig) error {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshal agent config: %w", err)
	}
	query := `INSERT INTO monitoring_agent_configs (id, agent_id, tenant_id, version, config_json, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)`
	_, err = r.pool.Exec(ctx, query, config.ID, config.AgentID, config.TenantID, config.Version, configJSON, config.IsActive)
	if err != nil {
		return fmt.Errorf("insert agent config: %w", err)
	}
	return nil
}

func (r *AgentConfigRepository) GetActive(ctx context.Context, agentID string) (*domain.AgentConfig, error) {
	query := `SELECT id::text, agent_id, tenant_id::text, version, config_json, is_active, created_at
		FROM monitoring_agent_configs WHERE agent_id = $1 AND is_active = TRUE ORDER BY version DESC LIMIT 1`
	config, err := r.scanConfig(ctx, query, agentID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &config, nil
}

func (r *AgentConfigRepository) GetByVersion(ctx context.Context, agentID string, version int) (*domain.AgentConfig, error) {
	query := `SELECT id::text, agent_id, tenant_id::text, version, config_json, is_active, created_at
		FROM monitoring_agent_configs WHERE agent_id = $1 AND version = $2`
	config, err := r.scanConfig(ctx, query, agentID, version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return &config, nil
}

func (r *AgentConfigRepository) DeactivateOlder(ctx context.Context, agentID string, keepVersion int) error {
	query := `UPDATE monitoring_agent_configs SET is_active = FALSE WHERE agent_id = $1 AND version < $2`
	_, err := r.pool.Exec(ctx, query, agentID, keepVersion)
	if err != nil {
		return fmt.Errorf("deactivate older configs: %w", err)
	}
	return nil
}

func (r *AgentConfigRepository) ListVersions(ctx context.Context, agentID string, limit int) ([]domain.AgentConfig, error) {
	query := `SELECT id::text, agent_id, tenant_id::text, version, config_json, is_active, created_at
		FROM monitoring_agent_configs WHERE agent_id = $1 ORDER BY version DESC LIMIT $2`
	rows, err := r.pool.Query(ctx, query, agentID, limit)
	if err != nil {
		return nil, fmt.Errorf("list config versions: %w", err)
	}
	defer rows.Close()

	configs := make([]domain.AgentConfig, 0)
	for rows.Next() {
		c, err := scanConfigRow(rows)
		if err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	return configs, rows.Err()
}

func (r *AgentConfigRepository) scanConfig(ctx context.Context, query string, args ...any) (domain.AgentConfig, error) {
	row := r.pool.QueryRow(ctx, query, args...)
	return scanConfigRow(row)
}

func scanConfigRow(row rowScanner) (domain.AgentConfig, error) {
	var c domain.AgentConfig
	var configJSON []byte
	err := row.Scan(&c.ID, &c.AgentID, &c.TenantID, &c.Version, &configJSON, &c.IsActive, &c.CreatedAt)
	if err != nil {
		return c, fmt.Errorf("scan config: %w", err)
	}
	if err := json.Unmarshal(configJSON, &c); err != nil {
		return c, fmt.Errorf("unmarshal config: %w", err)
	}
	return c, nil
}
