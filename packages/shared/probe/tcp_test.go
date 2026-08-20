package probe

import (
	"context"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"monitoring-platform/packages/shared/domain"
)

// startTCPListener listens on 127.0.0.1 and accepts (and closes) connections
// until the listener is closed.
func startTCPListener(t *testing.T) net.Listener {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	t.Cleanup(func() { _ = listener.Close() })
	return listener
}

func TestTCPExecutorSuccess(t *testing.T) {
	listener := startTCPListener(t)

	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, listener.Addr().String(), nil))

	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if result.Metrics["reachability"] != 1.0 {
		t.Fatalf("expected reachability=1, got %v", result.Metrics["reachability"])
	}
	if _, ok := result.Metrics["connect_time_ms"]; !ok {
		t.Fatal("expected connect_time_ms metric")
	}
	if result.Attributes["protocol"] != "tcp" {
		t.Fatalf("expected protocol=tcp, got %v", result.Attributes["protocol"])
	}
	if result.Attributes["port"] == nil {
		t.Fatal("expected port attribute")
	}
	if result.Attributes["ip_family"] == nil || result.Attributes["ip_family"] != "ipv4" {
		t.Fatalf("expected ip_family=ipv4, got %v", result.Attributes["ip_family"])
	}
}

func TestTCPExecutorPortFromConfig(t *testing.T) {
	listener := startTCPListener(t)

	_, portRaw, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portRaw)

	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "127.0.0.1", map[string]any{
		"port": float64(port),
	}))

	if !result.Success {
		t.Fatalf("expected success with config port, got %+v", result)
	}
	if result.Attributes["port"] != port {
		t.Fatalf("expected port attribute %d, got %v", port, result.Attributes["port"])
	}
}

func TestTCPExecutorConnectionRefused(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	address := listener.Addr().String()
	listener.Close()

	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, address, nil))

	if result.Success {
		t.Fatal("expected failure for closed port")
	}
	if result.ErrorCode != "connection_refused" {
		t.Fatalf("expected connection_refused, got %s", result.ErrorCode)
	}
	if result.Attributes["error_type"] != "connection_refused" {
		t.Fatalf("expected error_type connection_refused, got %v", result.Attributes["error_type"])
	}
	if result.Metrics["reachability"] != 0.0 {
		t.Fatalf("expected reachability=0, got %v", result.Metrics["reachability"])
	}
	if _, ok := result.Metrics["connect_time_ms"]; ok {
		t.Fatal("connect_time_ms must not be emitted for a failed connection")
	}
}

func TestTCPExecutorTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)

	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(ctx, testJob(domain.MonitorTCP, "127.0.0.1:9", nil))

	if result.Success {
		t.Fatal("expected failure on expired deadline")
	}
	if result.ErrorCode != "timeout" {
		t.Fatalf("expected timeout, got %s", result.ErrorCode)
	}
	if result.Attributes["error_type"] != "timeout" {
		t.Fatalf("expected error_type timeout, got %v", result.Attributes["error_type"])
	}
}

func TestTCPExecutorDNSResolutionFailure(t *testing.T) {
	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "does-not-exist.invalid:80", nil))

	if result.Success {
		t.Fatal("expected failure for unresolvable hostname")
	}
	if result.ErrorCode != "dns_resolution_failed" {
		t.Fatalf("expected dns_resolution_failed, got %s", result.ErrorCode)
	}
	if result.Attributes["error_type"] != "dns_resolution_failed" {
		t.Fatalf("expected error_type dns_resolution_failed, got %v", result.Attributes["error_type"])
	}
}

func TestTCPExecutorInvalidPort(t *testing.T) {
	executor := NewTCPExecutor(testDeps())

	for _, cfg := range []map[string]any{
		{"port": float64(0)},
		{"port": float64(70000)},
	} {
		result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "example.com", cfg))
		if result.Success || result.ErrorCode != "invalid_target" {
			t.Fatalf("expected invalid_target for %v, got %+v", cfg, result)
		}
	}
}

func TestTCPExecutorMissingPort(t *testing.T) {
	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "127.0.0.1", nil))

	if result.Success || result.ErrorCode != "invalid_target" {
		t.Fatalf("expected invalid_target, got %+v", result)
	}
}

func TestTCPExecutorEmptyTarget(t *testing.T) {
	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "", map[string]any{"port": float64(80)}))

	if result.Success || result.ErrorCode != "invalid_target" {
		t.Fatalf("expected invalid_target, got %+v", result)
	}
}

func TestTCPExecutorIPv4Selection(t *testing.T) {
	listener := startTCPListener(t)

	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "127.0.0.1", map[string]any{
		"port":       float64(portOf(listener)),
		"ip_version": "ipv4",
	}))

	if !result.Success {
		t.Fatalf("expected ipv4 success, got %+v", result)
	}
	if result.Attributes["ip_version"] != "ipv4" {
		t.Fatalf("expected ip_version ipv4, got %v", result.Attributes["ip_version"])
	}
}

func TestTCPExecutorIPv6Selection(t *testing.T) {
	listener, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "::1", map[string]any{
		"port":       float64(portOf(listener)),
		"ip_version": "ipv6",
	}))

	if !result.Success {
		t.Fatalf("expected ipv6 success, got %+v", result)
	}
	if result.Attributes["ip_family"] != "ipv6" {
		t.Fatalf("expected ip_family ipv6, got %v", result.Attributes["ip_family"])
	}
}

func TestTCPExecutorIPv4ForcesFailureOnIPv6OnlyTarget(t *testing.T) {
	listener, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}
	defer listener.Close()

	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "::1", map[string]any{
		"port":       float64(portOf(listener)),
		"ip_version": "ipv4",
	}))

	if result.Success {
		t.Fatal("expected failure when family filter excludes all addresses")
	}
	if result.ErrorCode != "blocked_target" && result.ErrorCode != "connection_failed" {
		t.Fatalf("unexpected error code %s", result.ErrorCode)
	}
}

func portOf(listener net.Listener) int {
	_, portRaw, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portRaw)
	return port
}

// ── Security: private / reserved destinations must be blocked ─────────────

func TestTCPExecutorBlocksPrivateTargets(t *testing.T) {
	executor := NewTCPExecutor(restrictiveDeps())

	cases := []struct {
		name   string
		target string
	}{
		{"loopback", "127.0.0.1:80"},
		{"private ipv4", "10.0.0.1:80"},
		{"private ipv4 c", "192.168.1.1:80"},
		{"private ipv4 b", "172.16.0.1:80"},
		{"link-local ipv4", "169.254.169.254:80"},
		{"metadata range", "100.100.100.200:80"},
		{"cgnat", "100.64.0.1:80"},
		{"multicast", "224.0.0.1:80"},
		{"reserved", "240.0.0.1:80"},
		{"ipv6 loopback", "[::1]:80"},
		{"ipv6 unique local", "[fc00::1]:80"},
		{"ipv6 link-local", "[fe80::1]:80"},
		{"ipv6 multicast", "[ff02::1]:80"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, tc.target, nil))
			if result.Success {
				t.Fatalf("expected blocked failure, got success: %+v", result)
			}
			if result.ErrorCode != "blocked_target" {
				t.Fatalf("expected blocked_target, got %s (error: %s)", result.ErrorCode, result.ErrorMessage)
			}
			if result.Attributes["error_type"] != "blocked_target" {
				t.Fatalf("expected error_type blocked_target, got %v", result.Attributes["error_type"])
			}
		})
	}
}

func TestTCPExecutorBlocksPrivateHostnameTarget(t *testing.T) {
	// A hostname resolving to private space must be rejected at dial time
	// (post-resolution validation), not only when the target is a literal IP.
	executor := NewTCPExecutor(restrictiveDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "localhost:80", nil))

	if result.Success {
		t.Fatal("expected blocked failure for localhost")
	}
	if result.ErrorCode != "blocked_target" {
		t.Fatalf("expected blocked_target, got %s (error: %s)", result.ErrorCode, result.ErrorMessage)
	}
}

func TestTCPExecutorAllowsPublicTarget(t *testing.T) {
	executor := NewTCPExecutor(restrictiveDeps())
	// TEST-NET-3 is public but unreachable; dial should not be policy-blocked,
	// so the failure must come from the transport layer, never from the Guard.
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "203.0.113.1:9", map[string]any{
		"timeout_ms": float64(200),
	}))

	if result.Success {
		t.Fatal("expected failure for unreachable public address")
	}
	if result.ErrorCode == "blocked_target" {
		t.Fatalf("public target must not be policy-blocked: %s", result.ErrorMessage)
	}
	if strings.Contains(result.ErrorMessage, "blocked") {
		t.Fatalf("unexpected block message: %s", result.ErrorMessage)
	}
}

func TestTCPExecutorHostConfigOverride(t *testing.T) {
	listener := startTCPListener(t)

	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "", map[string]any{
		"host": "127.0.0.1",
		"port": float64(portOf(listener)),
	}))

	if !result.Success {
		t.Fatalf("expected success with host config override, got %+v", result)
	}
	if result.Attributes["target"] != "127.0.0.1" {
		t.Fatalf("expected target 127.0.0.1, got %v", result.Attributes["target"])
	}
}
