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

type MonitoringAgentRepository struct {
	pool *pgxpool.Pool
}

func NewMonitoringAgentRepository(pool *pgxpool.Pool) *MonitoringAgentRepository {
	return &MonitoringAgentRepository{pool: pool}
}

const monitoringAgentColumns = `
	id::text,
	tenant_id::text,
	external_id,
	hostname,
	os,
	arch,
	version,
	agent_id,
	secret_hash,
	secret_encrypted,
	status,
	last_seen_at,
	registered_at,
	updated_at,
	labels,
	capabilities,
	private_ips,
	bootstrap_token_id::text
`

func (r *MonitoringAgentRepository) Create(ctx context.Context, agent *domain.MonitoringAgent) error {
	labelsJSON, err := json.Marshal(agent.Labels)
	if err != nil {
		return fmt.Errorf("marshal agent labels: %w", err)
	}

	query := `INSERT INTO monitoring_agents (tenant_id, external_id, hostname, os, arch, version, agent_id, secret_hash, secret_encrypted, status, labels, capabilities, private_ips, bootstrap_token_id)
	           VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
	           RETURNING ` + monitoringAgentColumns

	row := r.pool.QueryRow(ctx, query,
		agent.TenantID, agent.ExternalID, agent.Hostname, agent.OS, agent.Arch,
		agent.Version, agent.AgentID, agent.SecretHash, agent.SecretEncrypted, string(agent.Status),
		labelsJSON, agent.Capabilities, agent.PrivateIPs, agent.BootstrapTokenID,
	)

	created, err := scanMonitoringAgent(row)
	if err != nil {
		return fmt.Errorf("insert monitoring agent: %w", err)
	}

	*agent = created
	return nil
}

func (r *MonitoringAgentRepository) GetByAgentID(ctx context.Context, agentID string) (domain.MonitoringAgent, error) {
	query := `SELECT ` + monitoringAgentColumns + ` FROM monitoring_agents WHERE agent_id = $1`
	row := r.pool.QueryRow(ctx, query, agentID)

	agent, err := scanMonitoringAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MonitoringAgent{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MonitoringAgent{}, fmt.Errorf("get agent by agent_id: %w", err)
	}

	return agent, nil
}

func (r *MonitoringAgentRepository) GetByID(ctx context.Context, id string) (domain.MonitoringAgent, error) {
	query := `SELECT ` + monitoringAgentColumns + ` FROM monitoring_agents WHERE id = $1::uuid`
	row := r.pool.QueryRow(ctx, query, id)

	agent, err := scanMonitoringAgent(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.MonitoringAgent{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.MonitoringAgent{}, fmt.Errorf("get agent by id: %w", err)
	}

	return agent, nil
}

func (r *MonitoringAgentRepository) ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]domain.MonitoringAgent, int, error) {
	countQuery := `SELECT COUNT(*) FROM monitoring_agents WHERE tenant_id = $1`
	var total int
	err := r.pool.QueryRow(ctx, countQuery, tenantID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count agents by tenant: %w", err)
	}

	query := `SELECT ` + monitoringAgentColumns + ` FROM monitoring_agents WHERE tenant_id = $1
	           ORDER BY registered_at DESC LIMIT $2 OFFSET $3`
	rows, err := r.pool.Query(ctx, query, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list agents by tenant: %w", err)
	}
	defer rows.Close()

	agents := make([]domain.MonitoringAgent, 0)
	for rows.Next() {
		agent, err := scanMonitoringAgent(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan agent row: %w", err)
		}
		agents = append(agents, agent)
	}

	return agents, total, rows.Err()
}

func (r *MonitoringAgentRepository) Update(ctx context.Context, agent *domain.MonitoringAgent) error {
	labelsJSON, err := json.Marshal(agent.Labels)
	if err != nil {
		return fmt.Errorf("marshal agent labels: %w", err)
	}

	query := `UPDATE monitoring_agents
	           SET labels = $1, private_ips = $2, last_seen_at = $3, status = $4
	           WHERE agent_id = $5`
	tag, err := r.pool.Exec(ctx, query, labelsJSON, agent.PrivateIPs, agent.LastSeenAt, string(agent.Status), agent.AgentID)
	if err != nil {
		return fmt.Errorf("update monitoring agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MonitoringAgentRepository) UpdateStatus(ctx context.Context, agentID string, status domain.MonitoringAgentStatus) error {
	query := `UPDATE monitoring_agents SET status = $1 WHERE agent_id = $2`
	tag, err := r.pool.Exec(ctx, query, string(status), agentID)
	if err != nil {
		return fmt.Errorf("update agent status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MonitoringAgentRepository) UpdateHeartbeat(ctx context.Context, agentID string, hb domain.AgentHeartbeat) error {
	query := `INSERT INTO monitoring_agent_heartbeats (agent_id, tenant_id, cpu_percent, memory_percent, disk_percent, uptime_seconds, metrics_sent, metrics_queued, collector_uptime_seconds, public_ip, recorded_at)
	           VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := r.pool.Exec(ctx, query,
		agentID, hb.TenantID, hb.CPUPercent, hb.MemoryPercent, hb.DiskPercent,
		hb.UptimeSeconds, hb.MetricsSent, hb.MetricsQueued, hb.CollectorUptimeSeconds,
		hb.PublicIP, time.Now(),
	)
	if err != nil {
		return fmt.Errorf("insert agent heartbeat: %w", err)
	}

	updateQuery := `UPDATE monitoring_agents SET last_seen_at = $1 WHERE agent_id = $2`
	_, err = r.pool.Exec(ctx, updateQuery, time.Now(), agentID)
	if err != nil {
		return fmt.Errorf("update agent last_seen_at: %w", err)
	}

	return nil
}

func (r *MonitoringAgentRepository) Delete(ctx context.Context, agentID string) error {
	query := `DELETE FROM monitoring_agents WHERE agent_id = $1`
	tag, err := r.pool.Exec(ctx, query, agentID)
	if err != nil {
		return fmt.Errorf("delete agent: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *MonitoringAgentRepository) CountByTenant(ctx context.Context, tenantID string) (int, error) {
	query := `SELECT COUNT(*) FROM monitoring_agents WHERE tenant_id = $1`
	var count int
	err := r.pool.QueryRow(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count agents by tenant: %w", err)
	}
	return count, nil
}

func scanMonitoringAgent(row rowScanner) (domain.MonitoringAgent, error) {
	var (
		agent      domain.MonitoringAgent
		labelsJSON []byte
	)

	if err := row.Scan(
		&agent.ID, &agent.TenantID, &agent.ExternalID,
		&agent.Hostname, &agent.OS, &agent.Arch, &agent.Version,
		&agent.AgentID, &agent.SecretHash, &agent.SecretEncrypted, &agent.Status,
		&agent.LastSeenAt, &agent.RegisteredAt, &agent.UpdatedAt,
		&labelsJSON, &agent.Capabilities, &agent.PrivateIPs,
		&agent.BootstrapTokenID,
	); err != nil {
		return domain.MonitoringAgent{}, err
	}

	if err := json.Unmarshal(labelsJSON, &agent.Labels); err != nil {
		agent.Labels = map[string]string{}
	}

	return agent, nil
}
