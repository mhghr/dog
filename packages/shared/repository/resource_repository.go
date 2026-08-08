package repository

import (
	"context"
	"time"

	"monitoring-platform/packages/shared/domain"
)

type ResourceRepository interface {
	Create(ctx context.Context, res *domain.Resource) error
	GetByID(ctx context.Context, id string) (domain.Resource, error)
	List(ctx context.Context, filter domain.ResourceListFilter) ([]domain.Resource, int, error)
	Update(ctx context.Context, res *domain.Resource) error
	Delete(ctx context.Context, id string) error

	ListTypes(ctx context.Context) ([]domain.ResourceType, error)
	ListMonitorTypes(ctx context.Context) ([]domain.MonitorTypeDef, error)

	AttachTag(ctx context.Context, resourceID, key, value string) error
	RemoveTag(ctx context.Context, resourceID, tagID string) error
	ListTags(ctx context.Context, resourceID string) ([]domain.Tag, error)
	ListAllTags(ctx context.Context, orgID string) ([]domain.Tag, error)

	CreateWorkspace(ctx context.Context, ws *domain.Workspace) error
	ListWorkspaces(ctx context.Context, orgID string) ([]domain.Workspace, error)
}

// MonitorRepository is the port for resource-bound monitors.
type MonitorRepository interface {
	Create(ctx context.Context, monitor *domain.Monitor) error
	GetByID(ctx context.Context, id string) (domain.Monitor, error)
	ListByResource(ctx context.Context, resourceID string) ([]domain.Monitor, error)
	Update(ctx context.Context, monitor *domain.Monitor) error
	Delete(ctx context.Context, id string) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
	ClaimDue(ctx context.Context, batchSize int, fn func(domain.Monitor) error) (int, error)
	UpdateRunResult(ctx context.Context, monitorID string, status domain.MonitorStatus, checkedAt time.Time) error
	ListResults(ctx context.Context, monitorID string, limit, offset int) ([]domain.ProbeResult, int, error)
}
