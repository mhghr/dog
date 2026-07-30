package domain

import "time"

type AgentHeartbeat struct {
	ID                     int64
	AgentID                string
	TenantID               string
	CPUPercent             float64
	MemoryPercent          float64
	DiskPercent            float64
	UptimeSeconds          int64
	MetricsSent            int64
	MetricsQueued          int64
	CollectorUptimeSeconds int64
	PublicIP               string
	RecordedAt             time.Time
}
