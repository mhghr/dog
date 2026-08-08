package repository

import (
	"context"
	"monitoring-platform/packages/shared/domain"
)

type AgentConfigRepository interface {
	Create(ctx context.Context, config *domain.AgentConfig) error
	GetActive(ctx context.Context, agentID string) (*domain.AgentConfig, error)
	GetByVersion(ctx context.Context, agentID string, version int) (*domain.AgentConfig, error)
	DeactivateOlder(ctx context.Context, agentID string, keepVersion int) error
	ListVersions(ctx context.Context, agentID string, limit int) ([]domain.AgentConfig, error)
}
