package probe

import (
	"bytes"
	"context"
	"crypto/x509"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/security"
)

func testDeps() Deps {
	return Deps{
		Guard:  security.NewGuard(true),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

func testJob(monitorType domain.MonitorType, target string, config map[string]any) domain.ProbeJob {
	if config == nil {
		config = map[string]any{}
	}

	return domain.ProbeJob{
		ID:              "11111111-1111-1111-1111-111111111111",
		MonitorID:       "22222222-2222-2222-2222-222222222222",
		Type:            monitorType,
		Target:          target,
		TimeoutMillis:   5000,
		Retries:         0,
		Config:          config,
		ProbeLocationID: "33333333-3333-3333-3333-333333333333",
		ScheduledAt:     time.Now(),
	}
}

func execCtx(t *testing.T) context.Context {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestHTTPExecutorSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("service healthy"))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"expected_status_codes": []any{float64(200)},
		"body_contains":         "healthy",
	}))

	if !result.Success || result.Status != domain.StatusUp {
		t.Fatalf("expected success, got %+v", result)
	}
	if result.Attributes["status_code"] != 200 {
		t.Fatalf("expected status_code attribute 200, got %v", result.Attributes["status_code"])
	}
}

func TestHTTPExecutorURLConfigOverride(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())

	// The `url` config overrides the resource target.
	viaConfig := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, "https://wrong.example.com", map[string]any{
		"url": server.URL,
	}))
	if !viaConfig.Success {
		t.Fatalf("expected success when config.url overrides target, got %+v", viaConfig)
	}

	// A bare hostname is normalized to https:// (dial will fail at transport
	// level, but not at URL validation).
	bare := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, "https://wrong.example.com", map[string]any{
		"url": "203.0.113.10",
	}))
	if bare.ErrorCode == "invalid_target" {
		t.Fatalf("bare hostname must be normalized, not rejected as invalid target: %+v", bare)
	}
	if bare.Attributes["url"] != "https://203.0.113.10" {
		t.Fatalf("expected normalized url attribute, got %v", bare.Attributes["url"])
	}
}

func TestHTTPExecutorUnexpectedStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, nil))

	if result.Success {
		t.Fatal("expected failure")
	}
	if result.ErrorCode != "unexpected_status_code" {
		t.Fatalf("expected unexpected_status_code, got %s", result.ErrorCode)
	}
}

func TestHTTPExecutorBodyAssertionFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("something else"))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"body_contains": "healthy",
	}))

	if result.Success || result.ErrorCode != "body_assertion_failed" {
		t.Fatalf("expected body_assertion_failed, got %+v", result)
	}
}

func TestHTTPExecutorTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(ctx, testJob(domain.MonitorHTTP, server.URL, nil))

	if result.Success {
		t.Fatal("expected timeout failure")
	}
	if result.ErrorCode != "timeout" {
		t.Fatalf("expected timeout error code, got %s", result.ErrorCode)
	}
}

func TestHTTPExecutorBlockedTargetWhenGuardStrict(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	strictDeps := Deps{
		Guard:  security.NewGuard(false),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	executor := NewHTTPExecutor(strictDeps)
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, nil))

	if result.Success {
		t.Fatal("expected loopback target to be blocked")
	}
	if result.ErrorCode != "blocked_target" {
		t.Fatalf("expected blocked_target, got %s", result.ErrorCode)
	}
}

func TestHTTPExecutorPopulatesMetricsAndAttributes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, nil))

	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}

	if got, ok := result.Metrics["reachability"]; !ok || got != 1.0 {
		t.Fatalf("expected reachability metric 1, got %v", result.Metrics["reachability"])
	}
	if got, ok := result.Metrics["status_code"]; !ok || got != 200.0 {
		t.Fatalf("expected status_code metric 200, got %v", result.Metrics["status_code"])
	}
	// Timing metrics (dns/connect/ttfb) are only emitted when the httptrace
	// phase actually fires — never fabricated — so only assert the metrics
	// that are guaranteed on a successful loopback response.
	for _, key := range []string{"response_time_ms", "response_size_bytes", "content_assertion"} {
		if _, ok := result.Metrics[key]; !ok {
			t.Fatalf("expected metric %s to be present, got metrics %v", key, result.Metrics)
		}
	}
	if got, ok := result.Attributes["status_code"]; !ok || got != 200 {
		t.Fatalf("expected status_code attribute 200, got %v", result.Attributes["status_code"])
	}
	if got, ok := result.Attributes["url"]; !ok || got != server.URL {
		t.Fatalf("expected url attribute %s, got %v", server.URL, result.Attributes["url"])
	}
	if got, ok := result.Attributes["method"]; !ok || got != "GET" {
		t.Fatalf("expected method attribute GET, got %v", result.Attributes["method"])
	}
}

func TestHTTPExecutorFollowRedirects(t *testing.T) {
	var redirectHits int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		redirectHits++
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/end", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("landed"))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	job := testJob(domain.MonitorHTTP, server.URL+"/start", nil)
	result := executor.Execute(execCtx(t), job)

	if !result.Success {
		t.Fatalf("expected redirect to succeed, got %+v", result)
	}
	if redirectHits < 2 {
		t.Fatalf("expected redirect to be followed, server hits=%d", redirectHits)
	}
	if got, ok := result.Attributes["final_url"]; !ok || got != server.URL+"/end" {
		t.Fatalf("expected final_url %s, got %v", server.URL+"/end", result.Attributes["final_url"])
	}
}

func TestHTTPExecutorNoFollowRedirects(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/end", http.StatusFound)
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"follow_redirects": false,
		"expected_status_codes": []any{float64(302)},
	}))

	if !result.Success {
		t.Fatalf("expected 302 to be accepted without following, got %+v", result)
	}
	if got, ok := result.Attributes["status_code"]; !ok || got != 302 {
		t.Fatalf("expected status_code 302, got %v", result.Attributes["status_code"])
	}
}

func TestHTTPExecutorMethodHeadersAndBody(t *testing.T) {
	var gotMethod, gotHeader, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotHeader = r.Header.Get("X-Test")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"method": "POST",
		"headers": map[string]any{"X-Test": "abc"},
		"request_body": `{"ping":1}`,
	}))

	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if gotMethod != "POST" {
		t.Fatalf("expected POST method, got %s", gotMethod)
	}
	if gotHeader != "abc" {
		t.Fatalf("expected header X-Test=abc, got %s", gotHeader)
	}
	if gotBody != `{"ping":1}` {
		t.Fatalf("expected request body, got %q", gotBody)
	}
}

func TestHTTPExecutorVerifySSLDisabled(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("secure"))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())

	// Default verify_tls=true against a self-signed TLS server must fail.
	failing := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, nil))
	if failing.Success {
		t.Fatal("expected self-signed TLS to fail by default")
	}

	// With verify_tls=false the request should succeed.
	passing := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"verify_tls": false,
	}))
	if !passing.Success {
		t.Fatalf("expected success with verify_tls=false, got %+v", passing)
	}
}

func TestHTTPExecutorExpectedStatusAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"expected_status": 201,
	}))

	if !result.Success {
		t.Fatalf("expected 201 to pass with expected_status alias, got %+v", result)
	}
}

func TestHTTPExecutorContentAssertionMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("service healthy"))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())

	matching := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"body_contains": "healthy",
	}))
	if !matching.Success {
		t.Fatalf("expected body match to succeed, got %+v", matching)
	}
	if got := matching.Metrics["content_assertion"]; got != 1.0 {
		t.Fatalf("expected content_assertion=1 on match, got %v", got)
	}

	failing := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"body_contains": "missing text",
	}))
	if failing.Success || failing.ErrorCode != "body_assertion_failed" {
		t.Fatalf("expected body_assertion_failed, got %+v", failing)
	}
	if got := failing.Metrics["content_assertion"]; got != 0.0 {
		t.Fatalf("expected content_assertion=0 on mismatch, got %v", got)
	}
}

func TestHTTPExecutorFailureLeavesTimingAbsent(t *testing.T) {
	// A target that immediately refuses the connection (closed listener port).
	// The executor must report reachability=0 and NOT fabricate latency.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := listener.Addr().String()
	_ = listener.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, "http://"+addr, nil))

	if result.Success {
		t.Fatal("expected connection failure")
	}
	if got := result.Metrics["reachability"]; got != 0.0 {
		t.Fatalf("expected reachability=0 on failure, got %v", got)
	}
	if _, ok := result.Metrics["response_time_ms"]; ok {
		t.Fatalf("must not emit response_time_ms for a failed request, got %v", result.Metrics)
	}
	if _, ok := result.Metrics["ttfb_ms"]; ok {
		t.Fatalf("must not emit ttfb_ms for a failed request, got %v", result.Metrics)
	}
	if got := result.Attributes["error_type"]; got != "connection_failed" {
		t.Fatalf("expected error_type connection_failed, got %v", got)
	}
}

func TestHTTPExecutorRedirectToPrivateBlocked(t *testing.T) {
	// A redirect to a private/metadata address must never be followed.
	// http://127.0.0.1:1 is unresolvable as a public IP, so the redirect
	// target is rejected by the guarded dialer.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://169.254.169.254/latest/meta-data", http.StatusFound)
	}))
	defer server.Close()

	strictDeps := Deps{
		Guard:  security.NewGuard(false),
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	}

	executor := NewHTTPExecutor(strictDeps)
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"follow_redirects": true,
	}))

	if result.Success {
		t.Fatal("expected redirect to private metadata endpoint to be blocked")
	}
}

func TestHTTPExecutorIPFamilyIPv4(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"ip_version": "ipv4",
	}))

	if !result.Success {
		t.Fatalf("expected ipv4-family request to succeed against loopback v4, got %+v", result)
	}
	if got := result.Attributes["ip_version"]; got != "ipv4" {
		t.Fatalf("expected ip_version attribute ipv4, got %v", got)
	}
}

func TestHTTPExecutorIPFamilyIPv6(t *testing.T) {
	// Loopback test server only binds IPv4; forcing IPv6 must fail cleanly
	// with reachability=0 and no fabricated latency.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"ip_version": "ipv6",
	}))

	if result.Success {
		t.Fatal("expected ipv6-only request against an ipv4-only target to fail")
	}
	if got := result.Metrics["reachability"]; got != 0.0 {
		t.Fatalf("expected reachability=0, got %v", got)
	}
	if _, ok := result.Metrics["response_time_ms"]; ok {
		t.Fatalf("must not emit response_time_ms for a failed request, got %v", result.Metrics)
	}
}


func TestHTTPExecutorResponseTooLarge(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 4096))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"max_response_size_bytes": 1024,
	}))

	if result.Success {
		t.Fatal("expected oversized response to fail")
	}
	if result.ErrorCode != "response_too_large" {
		t.Fatalf("expected response_too_large, got %s", result.ErrorCode)
	}
	if got := result.Metrics["reachability"]; got != 1.0 {
		t.Fatalf("expected reachability=1 (response arrived) on oversized body, got %v", got)
	}
}

func TestHTTPExecutorResponseSizeLimitDefault(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 64*1024))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, nil))

	if !result.Success {
		t.Fatalf("expected 64KB body within default limit to succeed, got %+v", result)
	}
	if got, ok := result.Metrics["response_size_bytes"]; !ok || got != float64(64*1024) {
		t.Fatalf("expected response_size_bytes=65536, got %v", result.Metrics["response_size_bytes"])
	}
}

func TestHTTPExecutorDoesNotExposeCertificateAttributes(t *testing.T) {
	// HTTP monitoring covers availability and behavior only; certificate / TLS
	// layer inspection belongs to the dedicated SSL monitoring type.
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("secure"))
	}))
	defer server.Close()

	executor := NewHTTPExecutor(testDeps())
	result := executor.Execute(execCtx(t), testJob(domain.MonitorHTTP, server.URL, map[string]any{
		"verify_tls": false,
	}))

	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	for _, key := range []string{"tls_issuer", "tls_days_remaining", "tls_expires_at", "certificate_subject", "certificate_issuer"} {
		if _, ok := result.Attributes[key]; ok {
			t.Fatalf("HTTP result must not expose %q (SSL separation), got %v", key, result.Attributes)
		}
	}
}

func TestClassifyTLSFailure(t *testing.T) {
	// x509 errors map to precise categories.
	expired := x509.CertificateInvalidError{Reason: x509.Expired}
	code, errorType := classifyTLSFailure(&expired)
	if code != "tls_certificate_expired" || errorType != "tls_certificate_expired" {
		t.Fatalf("expired cert: got %s/%s", code, errorType)
	}

	hostname := x509.HostnameError{Certificate: new(x509.Certificate)}
	code, errorType = classifyTLSFailure(&hostname)
	if code != "tls_hostname_mismatch" || errorType != "tls_hostname_mismatch" {
		t.Fatalf("hostname mismatch: got %s/%s", code, errorType)
	}

	unknownCA := x509.UnknownAuthorityError{}
	code, errorType = classifyTLSFailure(&unknownCA)
	if code != "tls_unknown_ca" || errorType != "tls_unknown_ca" {
		t.Fatalf("unknown ca: got %s/%s", code, errorType)
	}

	code, errorType = classifyTLSFailure(errors.New("remote error: tls: handshake failure"))
	if code != "tls_handshake_failed" || errorType != "tls_handshake_failed" {
		t.Fatalf("generic handshake: got %s/%s", code, errorType)
	}
}


