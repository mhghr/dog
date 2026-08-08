package repository

import (
	"context"

	"monitoring-platform/packages/shared/domain"
)

type AuditRepository interface {
	Log(ctx context.Context, entry *domain.AuditLog) error
	ListByOrganization(ctx context.Context, orgID string, limit, offset int) ([]domain.AuditLog, int, error)
	ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]domain.AuditLog, int, error)
	ListByResource(ctx context.Context, resourceID string, limit, offset int) ([]domain.AuditLog, int, error)
	ListByActor(ctx context.Context, userID string, limit, offset int) ([]domain.AuditLog, int, error)
}
