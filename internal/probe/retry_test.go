package probe

import (
	"context"
	"errors"
	"testing"
	"time"

	"monitoring-platform/internal/domain"
)

type flakyExecutor struct {
	failuresBeforeSuccess int
	calls                 int
}

func (e *flakyExecutor) Type() domain.MonitorType {
	return domain.MonitorHTTP
}

func (e *flakyExecutor) Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult {
	e.calls++

	result := newBaseResult(job)
	if e.calls <= e.failuresBeforeSuccess {
		return finishFailure(result, "http_request_failed", errors.New("boom"))
	}

	return finishSuccess(result)
}

func TestExecuteWithRetrySucceedsAfterFailure(t *testing.T) {
	executor := &flakyExecutor{failuresBeforeSuccess: 1}

	job := domain.ProbeJob{
		ID:            "job-1",
		MonitorID:     "monitor-1",
		Type:          domain.MonitorHTTP,
		TimeoutMillis: 1000,
		Retries:       2,
	}

	result := ExecuteWithRetry(context.Background(), executor, job)

	if !result.Success {
		t.Fatalf("expected success after retry, got %+v", result)
	}
	if executor.calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", executor.calls)
	}
	if result.Attributes["attempt"] != 2 {
		t.Fatalf("expected attempt attribute 2, got %v", result.Attributes["attempt"])
	}
	if result.Attributes["max_attempts"] != 3 {
		t.Fatalf("expected max_attempts 3, got %v", result.Attributes["max_attempts"])
	}
}

func TestExecuteWithRetryExhaustsAttempts(t *testing.T) {
	executor := &flakyExecutor{failuresBeforeSuccess: 10}

	job := domain.ProbeJob{
		ID:            "job-2",
		MonitorID:     "monitor-2",
		Type:          domain.MonitorHTTP,
		TimeoutMillis: 500,
		Retries:       1,
	}

	start := time.Now()
	result := ExecuteWithRetry(context.Background(), executor, job)

	if result.Success {
		t.Fatal("expected failure")
	}
	if executor.calls != 2 {
		t.Fatalf("expected 2 attempts, got %d", executor.calls)
	}
	if elapsed := time.Since(start); elapsed < retryBackoff {
		t.Fatalf("expected backoff between attempts, elapsed %s", elapsed)
	}
	if result.ErrorCode != "http_request_failed" {
		t.Fatalf("unexpected error code %s", result.ErrorCode)
	}
}

func TestExecuteWithRetryZeroRetriesRunsOnce(t *testing.T) {
	executor := &flakyExecutor{failuresBeforeSuccess: 10}

	job := domain.ProbeJob{
		ID:            "job-3",
		MonitorID:     "monitor-3",
		Type:          domain.MonitorHTTP,
		TimeoutMillis: 500,
		Retries:       0,
	}

	ExecuteWithRetry(context.Background(), executor, job)

	if executor.calls != 1 {
		t.Fatalf("expected exactly 1 attempt, got %d", executor.calls)
	}
}
