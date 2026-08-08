package repository

import (
	"context"

	"monitoring-platform/packages/shared/domain"
)

type WorkspaceRepository interface {
	Create(ctx context.Context, ws *domain.Workspace) error
	GetByID(ctx context.Context, id string) (domain.Workspace, error)
	GetBySlug(ctx context.Context, orgID, slug string) (domain.Workspace, error)
	ListByOrganization(ctx context.Context, orgID string) ([]domain.Workspace, error)
	Update(ctx context.Context, ws *domain.Workspace) error
	Delete(ctx context.Context, id string) error

	AddMember(ctx context.Context, workspaceID string, input domain.WorkspaceMemberInput) error
	RemoveMember(ctx context.Context, workspaceID, userID string) error
	UpdateMemberRole(ctx context.Context, workspaceID, userID string, role domain.WorkspaceRole) error
	ListMembers(ctx context.Context, workspaceID string) ([]domain.WorkspaceMember, error)
	GetMemberRole(ctx context.Context, workspaceID, userID string) (domain.WorkspaceRole, error)
}
