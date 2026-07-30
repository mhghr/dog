package domain

import "time"

// MonitoringAgentStatus represents the lifecycle state of a monitoring agent.
type MonitoringAgentStatus string

const (
	// AgentStatusActive indicates the agent is connected and reporting.
	AgentStatusActive MonitoringAgentStatus = "active"
	// AgentStatusInactive indicates the agent has not been heard from recently.
	AgentStatusInactive MonitoringAgentStatus = "inactive"
	// AgentStatusDraining indicates the agent is shutting down gracefully.
	AgentStatusDraining MonitoringAgentStatus = "draining"
	// AgentStatusRemoved indicates the agent has been permanently decommissioned.
	AgentStatusRemoved MonitoringAgentStatus = "removed"
)

// MonitoringAgent represents an agent instance installed on a customer server.
type MonitoringAgent struct {
	// ID is the unique identifier for this agent record.
	ID string
	// TenantID is the tenant that owns this agent.
	TenantID string
	// ExternalID is a customer-provided identifier for the agent.
	ExternalID string
	// Hostname is the hostname of the server running the agent.
	Hostname string
	// OS is the operating system of the host (e.g. "linux", "windows").
	OS string
	// Arch is the CPU architecture of the host (e.g. "amd64", "arm64").
	Arch string
	// Version is the installed version of the agent software.
	Version string
	// AgentID is the public identifier the agent uses to authenticate.
	AgentID string
	// SecretHash is the bcrypt hash of the agent's shared secret.
	SecretHash string
	// Status is the current lifecycle state of the agent.
	Status MonitoringAgentStatus
	// LastSeenAt is the last time the agent reported a heartbeat.
	LastSeenAt *time.Time
	// RegisteredAt is when the agent first registered with the control plane.
	RegisteredAt time.Time
	// UpdatedAt is when the agent record was last modified.
	UpdatedAt time.Time
	// Labels are user-defined key/value pairs attached to the agent.
	Labels map[string]string
	// Capabilities lists the features this agent version supports.
	Capabilities []string
	// PrivateIPs lists the private IP addresses of the host.
	PrivateIPs []string
	// BootstrapTokenID is the token used during registration, if any.
	BootstrapTokenID *string
}
