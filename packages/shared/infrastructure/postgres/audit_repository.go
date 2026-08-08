package postgres

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/packages/shared/domain"
)

type AuditRepository struct {
	pool *pgxpool.Pool
}

func NewAuditRepository(pool *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{pool: pool}
}

func (r *AuditRepository) Log(ctx context.Context, entry *domain.AuditLog) error {
	detailsJSON := entry.Details
	if len(detailsJSON) == 0 {
		detailsJSON = json.RawMessage("{}")
	}

	err := r.pool.QueryRow(ctx, `
		INSERT INTO audit_logs (organization_id, workspace_id, actor_user_id, actor_agent_id,
		                        action, resource_type, resource_id, details, ip_address, user_agent)
		VALUES ($1::uuid, $2, $3, $4, $5::audit_action, $6, $7, $8::jsonb, $9, $10)
		RETURNING id::text, created_at`,
		entry.OrganizationID, entry.WorkspaceID, entry.ActorUserID, entry.ActorAgentID,
		string(entry.Action), entry.ResourceType, entry.ResourceID, detailsJSON,
		entry.IPAddress, entry.UserAgent,
	).Scan(&entry.ID, &entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}
	return nil
}

func (r *AuditRepository) ListByOrganization(ctx context.Context, orgID string, limit, offset int) ([]domain.AuditLog, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_logs WHERE organization_id = $1::uuid`, orgID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := r.pool.Query(ctx, auditSelect+` FROM audit_logs
		WHERE organization_id = $1::uuid
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, orgID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs: %w", err)
	}
	defer rows.Close()

	logs, err := r.scanAuditLogs(rows)
	return logs, total, err
}

func (r *AuditRepository) ListByWorkspace(ctx context.Context, workspaceID string, limit, offset int) ([]domain.AuditLog, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_logs WHERE workspace_id = $1::uuid`, workspaceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := r.pool.Query(ctx, auditSelect+` FROM audit_logs
		WHERE workspace_id = $1::uuid
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, workspaceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs by workspace: %w", err)
	}
	defer rows.Close()

	logs, err := r.scanAuditLogs(rows)
	return logs, total, err
}

func (r *AuditRepository) ListByResource(ctx context.Context, resourceID string, limit, offset int) ([]domain.AuditLog, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_logs WHERE resource_id = $1::uuid`, resourceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := r.pool.Query(ctx, auditSelect+` FROM audit_logs
		WHERE resource_id = $1::uuid
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, resourceID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs by resource: %w", err)
	}
	defer rows.Close()

	logs, err := r.scanAuditLogs(rows)
	return logs, total, err
}

func (r *AuditRepository) ListByActor(ctx context.Context, userID string, limit, offset int) ([]domain.AuditLog, int, error) {
	var total int
	if err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM audit_logs WHERE actor_user_id = $1::uuid`, userID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count audit logs: %w", err)
	}

	rows, err := r.pool.Query(ctx, auditSelect+` FROM audit_logs
		WHERE actor_user_id = $1::uuid
		ORDER BY created_at DESC LIMIT $2 OFFSET $3`, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list audit logs by actor: %w", err)
	}
	defer rows.Close()

	logs, err := r.scanAuditLogs(rows)
	return logs, total, err
}

const auditSelect = `
	SELECT id::text, organization_id::text, workspace_id::text, actor_user_id::text,
	       actor_agent_id::text, action::text, resource_type, resource_id::text,
	       details, ip_address, user_agent, created_at`

func (r *AuditRepository) scanAuditLogs(rows pgx.Rows) ([]domain.AuditLog, error) {
	var logs []domain.AuditLog
	for rows.Next() {
		var entry domain.AuditLog
		var action string
		if err := rows.Scan(&entry.ID, &entry.OrganizationID, &entry.WorkspaceID,
			&entry.ActorUserID, &entry.ActorAgentID, &action, &entry.ResourceType,
			&entry.ResourceID, &entry.Details, &entry.IPAddress, &entry.UserAgent,
			&entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit log: %w", err)
		}
		entry.Action = domain.AuditAction(action)
		logs = append(logs, entry)
	}
	return logs, rows.Err()
}
