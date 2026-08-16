package metrics

import (
	"strings"
	"testing"
	"time"

	"monitoring-platform/packages/shared/domain"
)

func pingResult(success bool) *domain.ProbeResult {
	status := domain.StatusUp
	if !success {
		status = domain.StatusDown
	}
	metrics := map[string]any{"reachability": 0, "packet_loss_percent": 100.0}
	if success {
		metrics = map[string]any{"reachability": 1, "packet_loss_percent": 0.0, "rtt_ms": 42.0}
	}
	return &domain.ProbeResult{
		ID:             "r1",
		MonitorID:      "m1",
		Status:         status,
		Success:        success,
		DurationMillis: 40,
		Metrics:        metrics,
		Attributes:     map[string]any{"resource_id": "res-1", "workspace_id": "ws-1"},
		FinishedAt:     time.Now().UTC(),
	}
}

func TestBuildLinesEmitsMonitorPingStatus(t *testing.T) {
	lines := buildLines(pingResult(true), "ping", "loc-1")
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, `monitor_ping_status{monitor_id="m1",monitor_type="ping",probe_location="loc-1",resource_id="res-1",workspace_id="ws-1"} 1`) {
		t.Fatalf("expected up status line, got:\n%s", joined)
	}
	if !strings.Contains(joined, `monitor_ping_rtt_ms{`) {
		t.Fatalf("expected latency line on success, got:\n%s", joined)
	}
}

func TestBuildLinesDownWritesNoLatency(t *testing.T) {
	lines := buildLines(pingResult(false), "ping", "loc-1")
	joined := strings.Join(lines, "\n")

	if !strings.Contains(joined, `monitor_ping_status{monitor_id="m1",monitor_type="ping",probe_location="loc-1",resource_id="res-1",workspace_id="ws-1"} 0 `) {
		t.Fatalf("expected down status line with value 0, got:\n%s", joined)
	}
	if strings.Contains(joined, "monitor_ping_rtt_ms") {
		t.Fatalf("down result must not emit latency line, got:\n%s", joined)
	}
	if strings.Contains(joined, "monitor_ping_jitter_ms") {
		t.Fatalf("down result must not emit jitter line, got:\n%s", joined)
	}
}

func TestBuildLinesNonPingHasNoStatusGauge(t *testing.T) {
	result := &domain.ProbeResult{
		ID: "r1", MonitorID: "m1", Status: domain.StatusUp, Success: true,
		Metrics:    map[string]any{},
		Attributes: map[string]any{},
		FinishedAt: time.Now().UTC(),
	}
	joined := strings.Join(buildLines(result, "http", "loc-1"), "\n")
	if strings.Contains(joined, "monitor_ping_status") {
		t.Fatalf("http result must not emit ping status, got:\n%s", joined)
	}
}
