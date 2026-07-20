package probe

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"

	"monitoring-platform/internal/domain"
)

func startTestDNSServer(t *testing.T) string {
	t.Helper()

	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp failed: %v", err)
	}

	mux := dns.NewServeMux()
	mux.HandleFunc("example.org.", func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetReply(r)

		if len(r.Question) > 0 && r.Question[0].Qtype == dns.TypeA {
			record, _ := dns.NewRR("example.org. 300 IN A 203.0.113.10")
			response.Answer = append(response.Answer, record)
		}

		_ = w.WriteMsg(response)
	})
	mux.HandleFunc("missing.org.", func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetRcode(r, dns.RcodeNameError)
		_ = w.WriteMsg(response)
	})

	server := &dns.Server{PacketConn: packetConn, Handler: mux}

	go func() {
		_ = server.ActivateAndServe()
	}()

	t.Cleanup(func() {
		_ = server.Shutdown()
	})

	time.Sleep(50 * time.Millisecond)
	return packetConn.LocalAddr().String()
}

func TestDNSExecutorSuccessWithExpectedValues(t *testing.T) {
	serverAddress := startTestDNSServer(t)

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorDNS, "example.org", map[string]any{
		"server":          serverAddress,
		"record_type":     "A",
		"expected_values": []any{"203.0.113.10"},
	}))

	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if result.Attributes["rcode"] != "NOERROR" {
		t.Fatalf("expected NOERROR rcode, got %v", result.Attributes["rcode"])
	}
}

func TestDNSExecutorAssertionFailure(t *testing.T) {
	serverAddress := startTestDNSServer(t)

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorDNS, "example.org", map[string]any{
		"server":          serverAddress,
		"record_type":     "A",
		"expected_values": []any{"198.51.100.99"},
	}))

	if result.Success || result.ErrorCode != "dns_assertion_failed" {
		t.Fatalf("expected dns_assertion_failed, got %+v", result)
	}
}

func TestDNSExecutorRcodeFailure(t *testing.T) {
	serverAddress := startTestDNSServer(t)

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorDNS, "missing.org", map[string]any{
		"server": serverAddress,
	}))

	if result.Success || result.ErrorCode != "dns_rcode_failed" {
		t.Fatalf("expected dns_rcode_failed, got %+v", result)
	}
}

func TestDNSExecutorInvalidRecordType(t *testing.T) {
	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorDNS, "example.org", map[string]any{
		"record_type": "BOGUS",
	}))

	if result.Success || result.ErrorCode != "invalid_record_type" {
		t.Fatalf("expected invalid_record_type, got %+v", result)
	}
}
