package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type WorkspaceRepository struct {
	pool *pgxpool.Pool
}

func NewWorkspaceRepository(pool *pgxpool.Pool) *WorkspaceRepository {
	return &WorkspaceRepository{pool: pool}
}

func (r *WorkspaceRepository) Create(ctx context.Context, ws *domain.Workspace) error {
	settingsJSON, err := json.Marshal(ws.Settings)
	if err != nil {
		settingsJSON = []byte("{}")
	}

	err = r.pool.QueryRow(ctx, `
		INSERT INTO workspaces (organization_id, name, slug, description, plan, settings)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
		RETURNING id::text, created_at, updated_at`,
		ws.OrganizationID, ws.Name, ws.Slug, ws.Description, string(ws.Plan), settingsJSON,
	).Scan(&ws.ID, &ws.CreatedAt, &ws.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return domain.ErrDuplicate
		}
		return fmt.Errorf("insert workspace: %w", err)
	}
	return nil
}

func (r *WorkspaceRepository) GetByID(ctx context.Context, id string) (domain.Workspace, error) {
	return r.scanWorkspace(r.pool.QueryRow(ctx, `
		SELECT id::text, organization_id::text, name, slug, description, plan, settings, created_at, updated_at
		FROM workspaces WHERE id = $1::uuid`, id))
}

func (r *WorkspaceRepository) GetBySlug(ctx context.Context, orgID, slug string) (domain.Workspace, error) {
	return r.scanWorkspace(r.pool.QueryRow(ctx, `
		SELECT id::text, organization_id::text, name, slug, description, plan, settings, created_at, updated_at
		FROM workspaces WHERE organization_id = $1::uuid AND slug = $2`, orgID, slug))
}

func (r *WorkspaceRepository) ListByOrganization(ctx context.Context, orgID string) ([]domain.Workspace, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id::text, organization_id::text, name, slug, description, plan, settings, created_at, updated_at
		FROM workspaces WHERE organization_id = $1::uuid ORDER BY created_at`, orgID)
	if err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	defer rows.Close()

	var workspaces []domain.Workspace
	for rows.Next() {
		ws, err := r.scanWorkspaceFromRows(rows)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, ws)
	}
	return workspaces, rows.Err()
}

func (r *WorkspaceRepository) Update(ctx context.Context, ws *domain.Workspace) error {
	settingsJSON, err := json.Marshal(ws.Settings)
	if err != nil {
		settingsJSON = []byte("{}")
	}

	row := r.pool.QueryRow(ctx, `
		UPDATE workspaces
		SET name = $2, description = $3, plan = $4, settings = $5::jsonb, updated_at = NOW()
		WHERE id = $1::uuid
		RETURNING updated_at`,
		ws.ID, ws.Name, ws.Description, string(ws.Plan), settingsJSON,
	)
	if err := row.Scan(&ws.UpdatedAt); errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrNotFound
	} else if err != nil {
		return fmt.Errorf("update workspace: %w", err)
	}
	return nil
}

func (r *WorkspaceRepository) Delete(ctx context.Context, id string) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM workspaces WHERE id = $1::uuid`, id)
	if err != nil {
		return fmt.Errorf("delete workspace: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *WorkspaceRepository) AddMember(ctx context.Context, workspaceID string, input domain.WorkspaceMemberInput) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO workspace_members (workspace_id, user_id, role)
		VALUES ($1::uuid, $2::uuid, $3::workspace_role)
		ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = $3::workspace_role`,
		workspaceID, input.UserID, string(input.Role))
	if err != nil {
		return fmt.Errorf("add workspace member: %w", err)
	}
	return nil
}

func (r *WorkspaceRepository) RemoveMember(ctx context.Context, workspaceID, userID string) error {
	tag, err := r.pool.Exec(ctx, `
		DELETE FROM workspace_members WHERE workspace_id = $1::uuid AND user_id = $2::uuid`,
		workspaceID, userID)
	if err != nil {
		return fmt.Errorf("remove workspace member: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *WorkspaceRepository) UpdateMemberRole(ctx context.Context, workspaceID, userID string, role domain.WorkspaceRole) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE workspace_members SET role = $3::workspace_role
		WHERE workspace_id = $1::uuid AND user_id = $2::uuid`,
		workspaceID, userID, string(role))
	if err != nil {
		return fmt.Errorf("update member role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}

func (r *WorkspaceRepository) ListMembers(ctx context.Context, workspaceID string) ([]domain.WorkspaceMember, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT workspace_id::text, user_id::text, role::text, joined_at
		FROM workspace_members WHERE workspace_id = $1::uuid ORDER BY joined_at`, workspaceID)
	if err != nil {
		return nil, fmt.Errorf("list members: %w", err)
	}
	defer rows.Close()

	var members []domain.WorkspaceMember
	for rows.Next() {
		var m domain.WorkspaceMember
		var role string
		if err := rows.Scan(&m.WorkspaceID, &m.UserID, &role, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("scan member: %w", err)
		}
		m.Role = domain.WorkspaceRole(role)
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *WorkspaceRepository) GetMemberRole(ctx context.Context, workspaceID, userID string) (domain.WorkspaceRole, error) {
	var role string
	err := r.pool.QueryRow(ctx, `
		SELECT role::text FROM workspace_members
		WHERE workspace_id = $1::uuid AND user_id = $2::uuid`,
		workspaceID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", domain.ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get member role: %w", err)
	}
	return domain.WorkspaceRole(role), nil
}

func (r *WorkspaceRepository) scanWorkspace(row pgx.Row) (domain.Workspace, error) {
	var ws domain.Workspace
	var plan string
	var settingsJSON []byte
	err := row.Scan(&ws.ID, &ws.OrganizationID, &ws.Name, &ws.Slug, &ws.Description,
		&plan, &settingsJSON, &ws.CreatedAt, &ws.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Workspace{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Workspace{}, err
	}
	ws.Plan = domain.WorkspacePlan(plan)
	if err := json.Unmarshal(settingsJSON, &ws.Settings); err != nil {
		ws.Settings = map[string]any{}
	}
	return ws, nil
}

func (r *WorkspaceRepository) scanWorkspaceFromRows(rows pgx.Rows) (domain.Workspace, error) {
	var ws domain.Workspace
	var plan string
	var settingsJSON []byte
	if err := rows.Scan(&ws.ID, &ws.OrganizationID, &ws.Name, &ws.Slug, &ws.Description,
		&plan, &settingsJSON, &ws.CreatedAt, &ws.UpdatedAt); err != nil {
		return domain.Workspace{}, fmt.Errorf("scan workspace: %w", err)
	}
	ws.Plan = domain.WorkspacePlan(plan)
	if err := json.Unmarshal(settingsJSON, &ws.Settings); err != nil {
		ws.Settings = map[string]any{}
	}
	return ws, nil
}
