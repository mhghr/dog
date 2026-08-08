package domain

import "time"

// AgentHeartbeat represents a periodic health report from an agent.
type AgentHeartbeat struct {
	// ID is the auto-incrementing primary key.
	ID int64
	// AgentID identifies the agent that sent the heartbeat.
	AgentID string
	// TenantID is the tenant that owns the agent.
	TenantID string
	// CPUPercent is the host CPU utilization at the time of the heartbeat.
	CPUPercent float64
	// MemoryPercent is the host memory utilization at the time of the heartbeat.
	MemoryPercent float64
	// DiskPercent is the host disk utilization at the time of the heartbeat.
	DiskPercent float64
	// UptimeSeconds is the total uptime of the host.
	UptimeSeconds int64
	// MetricsSent is the number of metrics exported since the agent started.
	MetricsSent int64
	// MetricsQueued is the number of metrics waiting in the export queue.
	MetricsQueued int64
	// CollectorUptimeSeconds is the uptime of the collector process.
	CollectorUptimeSeconds int64
	// PublicIP is the public IP address of the host.
	PublicIP string
	// RecordedAt is when the heartbeat was received by the control plane.
	RecordedAt time.Time
}
