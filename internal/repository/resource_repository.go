package repository

import (
	"context"
	"time"

	"monitoring-platform/internal/domain"
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

// MonitorV2Repository is the port for monitors attached to resources.
type MonitorV2Repository interface {
	Create(ctx context.Context, monitor *domain.MonitorV2) error
	GetByID(ctx context.Context, id string) (domain.MonitorV2, error)
	ListByResource(ctx context.Context, resourceID string) ([]domain.MonitorV2, error)
	Update(ctx context.Context, monitor *domain.MonitorV2) error
	Delete(ctx context.Context, id string) error
	SetEnabled(ctx context.Context, id string, enabled bool) error
	ClaimDue(ctx context.Context, batchSize int, fn func(domain.MonitorV2) error) (int, error)
	UpdateRunResult(ctx context.Context, monitorID string, status domain.MonitorStatus, checkedAt time.Time) error
	ListV2Results(ctx context.Context, monitorID string, limit, offset int) ([]domain.ProbeResult, int, error)
}
