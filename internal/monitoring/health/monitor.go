package health

import (
	"context"
	"log/slog"
	"time"
)

// SystemHealth is a snapshot of local host resource usage.
type SystemHealth struct {
	CPUPercent    float64
	MemoryPercent float64
	DiskPercent   float64
	UptimeSeconds int64
	LoadAvg1      float64
	LoadAvg5      float64
	LoadAvg15     float64
}

// Monitor samples host resource usage.
type Monitor struct {
	logger        *slog.Logger
	startTime     time.Time
	lastCPUSample int64
	lastCPUTime   int64
}

// NewMonitor creates a health monitor.
func NewMonitor(logger *slog.Logger) *Monitor {
	return &Monitor{
		logger:    logger,
		startTime: time.Now(),
	}
}

// Collect samples current host health.
func (m *Monitor) Collect(ctx context.Context) SystemHealth {
	h := SystemHealth{
		UptimeSeconds: int64(time.Since(m.startTime).Seconds()),
	}
	m.collectPlatform(&h)
	return h
}
