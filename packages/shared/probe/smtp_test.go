package probe

import (
	"bufio"
	"net"
	"strings"
	"testing"

	"monitoring-platform/packages/shared/domain"
)

// startFakeSMTPServer speaks a minimal ESMTP dialogue for handshake tests.
func startFakeSMTPServer(t *testing.T) string {
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

			go func(conn net.Conn) {
				defer conn.Close()

				writer := bufio.NewWriter(conn)
				reader := bufio.NewReader(conn)

				writeLine := func(line string) {
					_, _ = writer.WriteString(line + "\r\n")
					_ = writer.Flush()
				}

				writeLine("220 test.local ESMTP ready")

				for {
					line, err := reader.ReadString('\n')
					if err != nil {
						return
					}

					command := strings.ToUpper(strings.TrimSpace(line))
					switch {
					case strings.HasPrefix(command, "EHLO"):
						writeLine("250-test.local greets you")
						writeLine("250-SIZE 10240000")
						writeLine("250-8BITMIME")
						writeLine("250 HELP")
					case strings.HasPrefix(command, "NOOP"):
						writeLine("250 OK")
					case strings.HasPrefix(command, "QUIT"):
						writeLine("221 bye")
						return
					default:
						writeLine("502 command not implemented")
					}
				}
			}(conn)
		}
	}()

	t.Cleanup(func() { _ = listener.Close() })

	return listener.Addr().String()
}

func smtpJobFor(address string, config map[string]any) domain.ProbeJob {
	host, portRaw, _ := net.SplitHostPort(address)
	if config == nil {
		config = map[string]any{}
	}
	config["port"] = portRaw
	if _, exists := config["mode"]; !exists {
		config["mode"] = "plain"
	}

	return testJob(domain.MonitorSMTP, host, config)
}

func TestSMTPExecutorPlainHandshake(t *testing.T) {
	address := startFakeSMTPServer(t)

	executor := NewSMTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), smtpJobFor(address, nil))

	if !result.Success {
		t.Fatalf("expected success, got %+v (error: %s)", result, result.ErrorMessage)
	}

	capabilities, ok := result.Attributes["capabilities"].([]string)
	if !ok {
		t.Fatalf("expected capabilities attribute, got %T", result.Attributes["capabilities"])
	}

	found := false
	for _, capability := range capabilities {
		if capability == "SIZE" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected SIZE capability, got %v", capabilities)
	}
}

func TestSMTPExecutorExpectedCapabilityMissing(t *testing.T) {
	address := startFakeSMTPServer(t)

	executor := NewSMTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), smtpJobFor(address, map[string]any{
		"expected_capabilities": []any{"DSN"},
	}))

	if result.Success || result.ErrorCode != "smtp_capability_missing" {
		t.Fatalf("expected smtp_capability_missing, got %+v", result)
	}
}

func TestSMTPExecutorStartTLSUnavailable(t *testing.T) {
	address := startFakeSMTPServer(t)

	executor := NewSMTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), smtpJobFor(address, map[string]any{
		"mode":             "starttls",
		"require_starttls": true,
	}))

	if result.Success || result.ErrorCode != "smtp_starttls_unavailable" {
		t.Fatalf("expected smtp_starttls_unavailable, got %+v", result)
	}
}

func TestSMTPExecutorBannerAssertion(t *testing.T) {
	address := startFakeSMTPServer(t)

	executor := NewSMTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), smtpJobFor(address, map[string]any{
		"expected_banner_contains": "postfix",
	}))

	if result.Success || result.ErrorCode != "smtp_invalid_banner" {
		t.Fatalf("expected smtp_invalid_banner, got %+v", result)
	}
}

func TestSMTPExecutorConnectFailure(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	address := listener.Addr().String()
	listener.Close()

	executor := NewSMTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), smtpJobFor(address, nil))

	if result.Success || result.ErrorCode != "smtp_connect_failed" {
		t.Fatalf("expected smtp_connect_failed, got %+v", result)
	}
}
