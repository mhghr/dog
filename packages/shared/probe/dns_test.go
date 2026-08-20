package probe

import (
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"

	"monitoring-platform/packages/shared/domain"
)

// startDNSServer serves DNS over UDP on an ephemeral loopback port. The
// handler is fully configurable so tests can produce every RCODE / answer
// shape the executor must classify.
func startDNSServer(t *testing.T, handler dns.HandlerFunc) string {
	t.Helper()

	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen udp failed: %v", err)
	}

	server := &dns.Server{PacketConn: packetConn, Handler: handler}
	go func() {
		_ = server.ActivateAndServe()
	}()

	t.Cleanup(func() {
		_ = server.Shutdown()
	})

	time.Sleep(20 * time.Millisecond)
	return packetConn.LocalAddr().String()
}

// answerHandler responds to any query with a fixed RR (or just an empty
// NOERROR reply when rr is "").
func answerHandler(rr string) dns.HandlerFunc {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetReply(r)
		if rr != "" {
			if record, err := dns.NewRR(rr); err == nil {
				response.Answer = append(response.Answer, record)
			}
		}
		_ = w.WriteMsg(response)
	}
}

// typeAwareHandler responds based on the query name and record type.
func typeAwareHandler(records map[string]string) dns.HandlerFunc {
	return func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetReply(r)

		if len(r.Question) == 0 {
			_ = w.WriteMsg(response)
			return
		}
		question := r.Question[0]
		if rr, ok := records[question.Name+"|"+dns.TypeToString[question.Qtype]]; ok {
			if record, err := dns.NewRR(rr); err == nil {
				response.Answer = append(response.Answer, record)
			}
		}
		_ = w.WriteMsg(response)
	}
}

func dnsJob(name string, config map[string]any) domain.ProbeJob {
	return testJob(domain.MonitorDNS, name, config)
}

func TestDNSExecutorSuccessARecord(t *testing.T) {
	server := startDNSServer(t, answerHandler("example.org. 300 IN A 203.0.113.10"))

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":    server,
		"record_type": "A",
	}))

	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if result.Metrics["reachability"] != 1.0 {
		t.Fatalf("expected reachability=1, got %v", result.Metrics["reachability"])
	}
	if result.Metrics["answer_count"] != 1.0 {
		t.Fatalf("expected answer_count=1, got %v", result.Metrics["answer_count"])
	}
	if result.Metrics["ttl_seconds"] != 300.0 {
		t.Fatalf("expected ttl_seconds=300, got %v", result.Metrics["ttl_seconds"])
	}
	if _, ok := result.Metrics["response_time_ms"]; !ok {
		t.Fatal("expected response_time_ms metric")
	}
	if result.Attributes["record_type"] != "A" {
		t.Fatalf("expected record_type A, got %v", result.Attributes["record_type"])
	}
	if result.Attributes["resolver"] == nil {
		t.Fatal("expected resolver attribute")
	}
}

func TestDNSExecutorRecordTypes(t *testing.T) {
	cases := []struct {
		recordType string
		answer     string
	}{
		{"A", "example.org. 300 IN A 203.0.113.10"},
		{"AAAA", "example.org. 300 IN AAAA 2001:db8::10"},
		{"CNAME", "example.org. 300 IN CNAME target.example.net."},
		{"MX", "example.org. 300 IN MX 10 mail.example.org."},
		{"TXT", `example.org. 300 IN TXT "hello" "world"`},
		{"NS", "example.org. 300 IN NS ns1.example.org."},
	}

	for _, tc := range cases {
		t.Run(tc.recordType, func(t *testing.T) {
			server := startDNSServer(t, answerHandler(tc.answer))

			executor := NewDNSExecutor(testDeps())
			result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
				"resolver":    server,
				"record_type": tc.recordType,
			}))

			if !result.Success {
				t.Fatalf("expected success for %s, got %+v", tc.recordType, result)
			}
			if result.Metrics["answer_count"] == 0.0 {
				t.Fatalf("expected at least one answer for %s", tc.recordType)
			}
		})
	}
}

func TestDNSExecutorExpectedAnswerMatch(t *testing.T) {
	server := startDNSServer(t, answerHandler("example.org. 300 IN A 203.0.113.10"))

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":         server,
		"record_type":      "A",
		"expected_values":  []any{"203.0.113.10"},
		"expected_answers": []any{"203.0.113.10"},
	}))

	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if result.Metrics["expected_record_match"] != 1.0 {
		t.Fatalf("expected expected_record_match=1, got %v", result.Metrics["expected_record_match"])
	}
}

func TestDNSExecutorExpectedValueSingular(t *testing.T) {
	server := startDNSServer(t, answerHandler("example.org. 300 IN A 203.0.113.10"))

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":        server,
		"record_type":     "A",
		"expected_value":  "203.0.113.10",
	}))

	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if result.Metrics["expected_record_match"] != 1.0 {
		t.Fatalf("expected expected_record_match=1, got %v", result.Metrics["expected_record_match"])
	}
}

func TestDNSExecutorAnswerMismatch(t *testing.T) {
	server := startDNSServer(t, answerHandler("example.org. 300 IN A 203.0.113.10"))

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":        server,
		"record_type":     "A",
		"expected_values": []any{"198.51.100.99"},
	}))

	if result.Success || result.ErrorCode != "answer_mismatch" {
		t.Fatalf("expected answer_mismatch, got %+v", result)
	}
	if result.Metrics["expected_record_match"] != 0.0 {
		t.Fatalf("expected expected_record_match=0, got %v", result.Metrics["expected_record_match"])
	}
	if result.Metrics["reachability"] != 0.0 {
		t.Fatalf("expected reachability=0, got %v", result.Metrics["reachability"])
	}
}

func TestDNSExecutorNXDOMAIN(t *testing.T) {
	server := startDNSServer(t, func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetRcode(r, dns.RcodeNameError)
		_ = w.WriteMsg(response)
	})

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("missing.example.org", map[string]any{
		"resolver":    server,
		"record_type": "A",
	}))

	if result.Success || result.ErrorCode != "nxdomain" {
		t.Fatalf("expected nxdomain, got %+v", result)
	}
	if result.Attributes["rcode"] != "NXDOMAIN" {
		t.Fatalf("expected rcode NXDOMAIN, got %v", result.Attributes["rcode"])
	}
}

func TestDNSExecutorServerFailure(t *testing.T) {
	server := startDNSServer(t, func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetRcode(r, dns.RcodeServerFailure)
		_ = w.WriteMsg(response)
	})

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":    server,
		"record_type": "A",
	}))

	if result.Success || result.ErrorCode != "server_failure" {
		t.Fatalf("expected server_failure, got %+v", result)
	}
}

func TestDNSExecutorRefused(t *testing.T) {
	server := startDNSServer(t, func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetRcode(r, dns.RcodeRefused)
		_ = w.WriteMsg(response)
	})

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":    server,
		"record_type": "A",
	}))

	if result.Success || result.ErrorCode != "refused" {
		t.Fatalf("expected refused, got %+v", result)
	}
}

func TestDNSExecutorNoAnswer(t *testing.T) {
	server := startDNSServer(t, answerHandler(""))

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":    server,
		"record_type": "A",
	}))

	if result.Success || result.ErrorCode != "no_answer" {
		t.Fatalf("expected no_answer, got %+v", result)
	}
}

func TestDNSExecutorExpectedRecordNotFound(t *testing.T) {
	server := startDNSServer(t, answerHandler(""))

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":        server,
		"record_type":     "A",
		"expected_values": []any{"203.0.113.10"},
	}))

	if result.Success || result.ErrorCode != "expected_record_not_found" {
		t.Fatalf("expected expected_record_not_found, got %+v", result)
	}
}

func TestDNSExecutorTimeout(t *testing.T) {
	// A UDP socket that silently drops queries produces a deterministic
	// timeout once the short per-check timeout expires.
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer packetConn.Close()

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":    packetConn.LocalAddr().String(),
		"record_type": "A",
		"timeout_ms":  float64(150),
	}))

	if result.Success || result.ErrorCode != "timeout" {
		t.Fatalf("expected timeout, got %+v", result)
	}
}

func TestDNSExecutorMalformedResponse(t *testing.T) {
	packetConn, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen failed: %v", err)
	}
	defer packetConn.Close()

	go func() {
		buffer := make([]byte, 512)
		_, addr, err := packetConn.ReadFrom(buffer)
		if err != nil {
			return
		}
		// Respond with garbage that cannot be unpacked as a DNS message.
		_, _ = packetConn.WriteTo([]byte{0xde, 0xad, 0xbe, 0xef, 0x00, 0x01}, addr)
	}()

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":    packetConn.LocalAddr().String(),
		"record_type": "A",
	}))

	if result.Success || result.ErrorCode != "malformed_response" {
		t.Fatalf("expected malformed_response, got %+v (error: %s)", result, result.ErrorMessage)
	}
}

func TestDNSExecutorResolverUnreachable(t *testing.T) {
	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":    "does-not-exist.invalid:53",
		"record_type": "A",
	}))

	if result.Success || result.ErrorCode != "resolver_unreachable" {
		t.Fatalf("expected resolver_unreachable, got %+v", result)
	}
}

func TestDNSExecutorInvalidRecordType(t *testing.T) {
	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"record_type": "BOGUS",
	}))

	if result.Success || result.ErrorCode != "invalid_record_type" {
		t.Fatalf("expected invalid_record_type, got %+v", result)
	}
}

func TestDNSExecutorEmptyTarget(t *testing.T) {
	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("", map[string]any{}))

	if result.Success || result.ErrorCode != "invalid_target" {
		t.Fatalf("expected invalid_target, got %+v", result)
	}
}

func TestDNSExecutorTruncationFallsBackToTCP(t *testing.T) {
	// A handler that sets the TC bit for UDP triggers the TCP fallback. The
	// resolver must answer over both UDP and TCP on the same port, as a real
	// DNS server does.
	tcpListener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("tcp listen failed: %v", err)
	}
	defer tcpListener.Close()

	_, portRaw, _ := net.SplitHostPort(tcpListener.Addr().String())
	packetConn, err := net.ListenPacket("udp", net.JoinHostPort("127.0.0.1", portRaw))
	if err != nil {
		t.Skipf("cannot bind udp on tcp port %s: %v", portRaw, err)
	}
	defer packetConn.Close()

	truncated := dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetReply(r)
		response.Truncated = true
		_ = w.WriteMsg(response)
	})

	udpServer := &dns.Server{PacketConn: packetConn, Handler: truncated}
	go func() { _ = udpServer.ActivateAndServe() }()
	t.Cleanup(func() { _ = udpServer.Shutdown() })

	tcpServer := &dns.Server{Listener: tcpListener, Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
		response := new(dns.Msg)
		response.SetReply(r)
		record, _ := dns.NewRR("example.org. 300 IN A 203.0.113.10")
		response.Answer = append(response.Answer, record)
		_ = w.WriteMsg(response)
	})}
	go func() { _ = tcpServer.ActivateAndServe() }()
	t.Cleanup(func() { _ = tcpServer.Shutdown() })

	time.Sleep(20 * time.Millisecond)

	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":    packetConn.LocalAddr().String(),
		"record_type": "A",
	}))

	if !result.Success {
		t.Fatalf("expected success via TCP fallback, got %+v", result)
	}
	if result.Metrics["answer_count"] != 1.0 {
		t.Fatalf("expected answer_count=1, got %v", result.Metrics["answer_count"])
	}
}

// ── Security: the resolver itself is validated by the Guard ───────────────

func TestDNSExecutorBlocksPrivateResolver(t *testing.T) {
	server := startDNSServer(t, answerHandler("example.org. 300 IN A 203.0.113.10"))

	executor := NewDNSExecutor(restrictiveDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":    server, // 127.0.0.1 — private loopback
		"record_type": "A",
	}))

	if result.Success {
		t.Fatal("expected blocked failure for private resolver")
	}
	if result.ErrorCode != "blocked_target" {
		t.Fatalf("expected blocked_target, got %s (error: %s)", result.ErrorCode, result.ErrorMessage)
	}
	if result.Attributes["error_type"] != "blocked_target" {
		t.Fatalf("expected error_type blocked_target, got %v", result.Attributes["error_type"])
	}
}

func TestDNSExecutorAllowsPublicResolver(t *testing.T) {
	// TEST-NET resolver IPs are public: validation must not policy-block them,
	// even though no real resolver answers there.
	executor := NewDNSExecutor(restrictiveDeps())
	result := executor.Execute(execCtx(t), dnsJob("example.org", map[string]any{
		"resolver":    "203.0.113.53:53",
		"record_type": "A",
		"timeout_ms":  float64(150),
	}))

	if result.ErrorCode == "blocked_target" {
		t.Fatalf("public resolver must not be policy-blocked: %s", result.ErrorMessage)
	}
	if result.Attributes["error_type"] == "blocked_target" {
		t.Fatalf("public resolver must not be policy-blocked: %v", result.Attributes["error_type"])
	}
}

func TestDNSExecutorSystemResolverNXDOMAIN(t *testing.T) {
	executor := NewDNSExecutor(testDeps())
	result := executor.Execute(execCtx(t), dnsJob("does-not-exist.invalid", map[string]any{
		"record_type": "A",
	}))

	if result.Success {
		t.Fatal("expected failure for .invalid via system resolver")
	}
	if result.ErrorCode != "nxdomain" {
		t.Fatalf("expected nxdomain, got %s", result.ErrorCode)
	}
	if result.Attributes["resolver"] != "system" {
		t.Fatalf("expected resolver=system, got %v", result.Attributes["resolver"])
	}
}
