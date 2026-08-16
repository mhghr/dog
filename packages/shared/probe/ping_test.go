package probe

import (
	"context"
	"errors"
	"testing"
	"time"

	"monitoring-platform/packages/shared/domain"
)

func newPingResult() domain.ProbeResult {
	return newBaseResult(testJob(domain.MonitorPing, "127.0.0.1", nil))
}

func TestShapePingResultSuccess(t *testing.T) {
	result := shapePingResult(newPingResult(), "127.0.0.1", pingStats{
		packetsSent:     4,
		packetsReceived: 4,
		packetLoss:      0,
		avgRTT:          42 * time.Millisecond,
		minRTT:          38 * time.Millisecond,
		maxRTT:          47 * time.Millisecond,
		stdDevRTT:       5 * time.Millisecond,
	})

	if result.Status != domain.StatusUp || !result.Success {
		t.Fatalf("expected up/success, got %+v", result)
	}
	if result.Metrics["reachability"] != 1 {
		t.Fatalf("expected reachability=1, got %v", result.Metrics["reachability"])
	}
	if result.Metrics["packet_loss_percent"] != 0.0 {
		t.Fatalf("expected packet_loss_percent=0, got %v", result.Metrics["packet_loss_percent"])
	}
	if result.Metrics["rtt_ms"] != 42.0 {
		t.Fatalf("expected rtt_ms=42, got %v", result.Metrics["rtt_ms"])
	}
	if result.Attributes["packets_sent"] != 4 || result.Attributes["packets_received"] != 4 {
		t.Fatalf("expected packet counters in attributes, got %+v", result.Attributes)
	}
}

func TestShapePingResultTotalFailureKeepsLatencyNull(t *testing.T) {
	result := shapePingResult(newPingResult(), "127.0.0.1", pingStats{
		packetsSent:     4,
		packetsReceived: 0,
		packetLoss:      100,
	})

	if result.Status != domain.StatusDown || result.Success {
		t.Fatalf("expected down/failure, got %+v", result)
	}
	if result.Metrics["reachability"] != 0 {
		t.Fatalf("expected reachability=0, got %v", result.Metrics["reachability"])
	}
	if result.Metrics["packet_loss_percent"] != 100.0 {
		t.Fatalf("expected packet_loss_percent=100, got %v", result.Metrics["packet_loss_percent"])
	}
	for _, key := range []string{"rtt_ms", "min_rtt_ms", "max_rtt_ms", "jitter_ms"} {
		if _, present := result.Metrics[key]; present {
			t.Fatalf("expected %s to be ABSENT (NULL), got %v", key, result.Metrics[key])
		}
	}
	if result.ErrorCode != "timeout" {
		t.Fatalf("expected error_code=timeout, got %s", result.ErrorCode)
	}
}

func TestShapePingResultPartialLoss(t *testing.T) {
	result := shapePingResult(newPingResult(), "127.0.0.1", pingStats{
		packetsSent:     4,
		packetsReceived: 3,
		packetLoss:      25,
		avgRTT:          40 * time.Millisecond,
	})

	if result.Status != domain.StatusUp || !result.Success {
		t.Fatalf("expected up (partial loss is still reachable), got %+v", result)
	}
	if result.Metrics["packet_loss_percent"] != 25.0 {
		t.Fatalf("expected packet_loss_percent=25, got %v", result.Metrics["packet_loss_percent"])
	}
}

func TestShapePingResultSetsResolvedIP(t *testing.T) {
	result := shapePingResult(newPingResult(), "10.0.0.1", pingStats{
		packetsSent:     4,
		packetsReceived: 4,
		avgRTT:          20 * time.Millisecond,
	})
	if result.Attributes["resolved_ip"] != "10.0.0.1" {
		t.Fatalf("expected resolved_ip attribute, got %+v", result.Attributes)
	}
}

func TestMapErrorCodePermissionDenied(t *testing.T) {
	if code := mapErrorCode("ping_failed", errors.New("listen icmp: operation not permitted")); code != "permission_denied" {
		t.Fatalf("expected permission_denied, got %s", code)
	}
	if code := mapErrorCode("ping_failed", errors.New("socket: permission denied")); code != "permission_denied" {
		t.Fatalf("expected permission_denied, got %s", code)
	}
}

func TestMapErrorCodeTimeout(t *testing.T) {
	if code := mapErrorCode("ping_failed", context.DeadlineExceeded); code != "timeout" {
		t.Fatalf("expected timeout, got %s", code)
	}
}
