package health

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"math"
	"net/http"
	"time"
)

// Reporter periodically sends heartbeats to Core.
type Reporter struct {
	heartbeatURL string
	authHeaders  func() map[string]string
	monitor      *Monitor
	logger       *slog.Logger
	client       *http.Client
}

// NewReporter creates a heartbeat reporter.
//
// heartbeatURL must be a full URL (scheme + host + path) to the core
// heartbeat endpoint, e.g. "https://core.example.com/api/v1/monitoring/agents/ag_123/heartbeat".
// The caller is responsible for joining the relative heartbeat path returned
// by bootstrap with the core server URL. authHeaders returns the HMAC
// authenticated headers (e.g. from the credential manager's AuthHeader).
func NewReporter(heartbeatURL string, authHeaders func() map[string]string, monitor *Monitor, logger *slog.Logger) *Reporter {
	return &Reporter{
		heartbeatURL: heartbeatURL,
		authHeaders:  authHeaders,
		monitor:      monitor,
		logger:       logger,
		client:       &http.Client{Timeout: 30 * time.Second},
	}
}

// Start reports heartbeats every interval until the context is cancelled.
func (r *Reporter) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.sendHeartbeat(ctx)
		}
	}
}

func (r *Reporter) sendHeartbeat(ctx context.Context) {
	h := r.monitor.Collect(ctx)

	body := map[string]any{
		"cpu_percent":    round2(h.CPUPercent),
		"memory_percent": round2(h.MemoryPercent),
		"disk_percent":   round2(h.DiskPercent),
		"uptime_seconds": h.UptimeSeconds,
		"public_ip":      "",
	}

	data, err := json.Marshal(body)
	if err != nil {
		r.logger.Error("marshal heartbeat", "error", err)
		return
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.heartbeatURL, bytes.NewReader(data))
	if err != nil {
		r.logger.Error("create heartbeat request", "error", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range r.authHeaders() {
		req.Header.Set(k, v)
	}

	resp, err := r.client.Do(req)
	if err != nil {
		r.logger.Debug("heartbeat request failed", "error", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		r.logger.Warn("heartbeat rejected", "status", resp.StatusCode)
	}
}

func round2(v float64) float64 {
	return math.Round(v*100) / 100
}
