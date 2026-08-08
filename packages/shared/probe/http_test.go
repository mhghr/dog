package probe

import (
	"context"
	"io"
	"log/slog"
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
