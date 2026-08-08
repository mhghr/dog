package domain

import (
	"encoding/json"
	"net"
	"time"
)

// AuditAction categorises every auditable operation in the platform.
type AuditAction string

const (
	AuditResourceCreated      AuditAction = "resource.created"
	AuditResourceUpdated      AuditAction = "resource.updated"
	AuditResourceDeleted      AuditAction = "resource.deleted"
	AuditResourceStatusChange AuditAction = "resource.status_changed"
	AuditMonitorCreated       AuditAction = "monitor.created"
	AuditMonitorUpdated       AuditAction = "monitor.updated"
	AuditMonitorDeleted       AuditAction = "monitor.deleted"
	AuditMonitorEnabled       AuditAction = "monitor.enabled"
	AuditMonitorDisabled      AuditAction = "monitor.disabled"
	AuditMonitorStatusChange  AuditAction = "monitor.status_changed"
	AuditWorkspaceCreated     AuditAction = "workspace.created"
	AuditWorkspaceUpdated     AuditAction = "workspace.updated"
	AuditWorkspaceDeleted     AuditAction = "workspace.deleted"
	AuditMemberInvited        AuditAction = "member.invited"
	AuditMemberRemoved        AuditAction = "member.removed"
	AuditMemberRoleChanged    AuditAction = "member.role_changed"
	AuditAlertCreated         AuditAction = "alert.created"
	AuditAlertAcknowledged    AuditAction = "alert.acknowledged"
	AuditAlertResolved        AuditAction = "alert.resolved"
	AuditAgentEnrolled        AuditAction = "agent.enrolled"
	AuditAgentApproved        AuditAction = "agent.approved"
	AuditAgentRevoked         AuditAction = "agent.revoked"
	AuditAgentStatusChanged   AuditAction = "agent.status_changed"
	AuditProbeEnrolled        AuditAction = "probe.enrolled"
	AuditProbeUpdated         AuditAction = "probe.updated"
	AuditProbeStatusChanged   AuditAction = "probe.status_changed"
	AuditPluginEnabled        AuditAction = "plugin.enabled"
	AuditPluginDisabled       AuditAction = "plugin.disabled"
	AuditCredentialCreated    AuditAction = "credential.created"
	AuditCredentialUpdated    AuditAction = "credential.updated"
	AuditCredentialDeleted    AuditAction = "credential.deleted"
	AuditSNMPDeviceAdded      AuditAction = "snmp.device_added"
	AuditSNMPDeviceRemoved    AuditAction = "snmp.device_removed"
	AuditConfigChanged        AuditAction = "config.changed"
	AuditSettingsUpdated      AuditAction = "settings.updated"
	AuditUserLogin            AuditAction = "user.login"
	AuditUserLogout           AuditAction = "user.logout"
	AuditUserPasswordChanged  AuditAction = "user.password_changed"
	AuditTokenCreated         AuditAction = "token.created"
	AuditTokenRevoked         AuditAction = "token.revoked"
)

type AuditLog struct {
	ID             string          `json:"id"`
	OrganizationID string          `json:"organization_id"`
	WorkspaceID    *string         `json:"workspace_id,omitempty"`
	ActorUserID    *string         `json:"actor_user_id,omitempty"`
	ActorAgentID   *string         `json:"actor_agent_id,omitempty"`
	Action         AuditAction     `json:"action"`
	ResourceType   string          `json:"resource_type"`
	ResourceID     *string         `json:"resource_id,omitempty"`
	Details        json.RawMessage `json:"details"`
	IPAddress      net.IP          `json:"ip_address,omitempty"`
	UserAgent      string          `json:"user_agent,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
}
