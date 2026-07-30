package domain

import "time"

type MonitoringAgentStatus string

const (
	AgentStatusActive   MonitoringAgentStatus = "active"
	AgentStatusInactive MonitoringAgentStatus = "inactive"
	AgentStatusDraining MonitoringAgentStatus = "draining"
	AgentStatusRemoved  MonitoringAgentStatus = "removed"
)

type MonitoringAgent struct {
	ID               string
	TenantID         string
	ExternalID       string
	Hostname         string
	OS               string
	Arch             string
	Version          string
	AgentID          string
	SecretHash       string
	Status           MonitoringAgentStatus
	LastSeenAt       *time.Time
	RegisteredAt     time.Time
	UpdatedAt        time.Time
	Labels           map[string]string
	Capabilities     []string
	PrivateIPs       []string
	BootstrapTokenID *string
}
