package repository

import (
	"context"
	"monitoring-platform/internal/domain"
)

type MonitoringAgentRepository interface {
	Create(ctx context.Context, agent *domain.MonitoringAgent) error
	GetByAgentID(ctx context.Context, agentID string) (*domain.MonitoringAgent, error)
	GetByID(ctx context.Context, id string) (*domain.MonitoringAgent, error)
	ListByTenant(ctx context.Context, tenantID string) ([]domain.MonitoringAgent, error)
	UpdateStatus(ctx context.Context, agentID string, status domain.MonitoringAgentStatus) error
	UpdateLastSeen(ctx context.Context, agentID string, publicIP string) error
	UpdateHeartbeat(ctx context.Context, agentID string, cpu, memory, disk float64, uptime int64, publicIP string) error
	Delete(ctx context.Context, agentID string) error
	CountByTenant(ctx context.Context, tenantID string) (int, error)
}
