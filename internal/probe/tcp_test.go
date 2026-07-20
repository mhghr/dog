package probe

import (
	"net"
	"strconv"
	"testing"

	"monitoring-platform/internal/domain"
)

func TestTCPExecutorSuccess(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
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
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, listener.Addr().String(), nil))

	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if result.Attributes["remote_address"] == nil {
		t.Fatal("expected remote_address attribute")
	}
}

func TestTCPExecutorPortFromConfig(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
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

	_, portRaw, _ := net.SplitHostPort(listener.Addr().String())
	port, _ := strconv.Atoi(portRaw)

	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "127.0.0.1", map[string]any{
		"port": float64(port),
	}))

	if !result.Success {
		t.Fatalf("expected success with config port, got %+v", result)
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
	if result.ErrorCode != "tcp_connect_failed" {
		t.Fatalf("expected tcp_connect_failed, got %s", result.ErrorCode)
	}
}

func TestTCPExecutorMissingPort(t *testing.T) {
	executor := NewTCPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorTCP, "127.0.0.1", nil))

	if result.Success || result.ErrorCode != "invalid_target" {
		t.Fatalf("expected invalid_target, got %+v", result)
	}
}
