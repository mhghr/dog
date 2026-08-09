package domain

import "context"

type WorkspacePermission string

const (
	PermWorkspaceManage WorkspacePermission = "workspace.manage"
	PermBillingManage   WorkspacePermission = "workspace.billing.manage"
	PermResourceCreate  WorkspacePermission = "resource.create"
	PermResourceUpdate  WorkspacePermission = "resource.update"
	PermResourceDelete  WorkspacePermission = "resource.delete"
	PermMonitorCreate   WorkspacePermission = "monitor.create"
	PermMonitorUpdate   WorkspacePermission = "monitor.update"
	PermMonitorDelete   WorkspacePermission = "monitor.delete"
	PermAlertManage     WorkspacePermission = "alert.manage"
	PermTeamManage      WorkspacePermission = "team.manage"
)

const WorkspaceIDContextKey OrgContextKey = "workspace.id"

func WorkspaceIDFromContext(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(WorkspaceIDContextKey).(string)
	return id, ok && id != ""
}
