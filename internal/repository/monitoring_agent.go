package repository

import (
	"context"
	"monitoring-platform/internal/domain"
)

// MonitoringAgentRepository persists monitoring agent records.
type MonitoringAgentRepository interface {
	// Create persists a new monitoring agent.
	Create(ctx context.Context, agent *domain.MonitoringAgent) error
	// GetByAgentID retrieves an agent by its public agent identifier.
	GetByAgentID(ctx context.Context, agentID string) (domain.MonitoringAgent, error)
	// GetByID retrieves an agent by its internal ID.
	GetByID(ctx context.Context, id string) (domain.MonitoringAgent, error)
	// ListByTenant returns a paginated list of agents for a tenant along with the total count.
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]domain.MonitoringAgent, int, error)
	// Update persists mutable fields (labels, private_ips, last_seen_at, status) of an agent.
	Update(ctx context.Context, agent *domain.MonitoringAgent) error
	// UpdateStatus updates the lifecycle status of an agent.
	UpdateStatus(ctx context.Context, agentID string, status domain.MonitoringAgentStatus) error
	// UpdateHeartbeat records a heartbeat report from an agent.
	UpdateHeartbeat(ctx context.Context, agentID string, hb domain.AgentHeartbeat) error
	// Delete removes an agent by its public agent identifier.
	Delete(ctx context.Context, agentID string) error
	// CountByTenant returns the total number of agents for a tenant.
	CountByTenant(ctx context.Context, tenantID string) (int, error)
}
