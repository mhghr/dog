package probe

import (
	"context"
	"errors"
	"testing"
	"time"

	"monitoring-platform/packages/shared/domain"
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
	if result.Attributes["attempts"] != 2 {
		t.Fatalf("expected attempts attribute 2, got %v", result.Attributes["attempts"])
	}
	if result.Attributes["first_attempt_failed"] != true {
		t.Fatalf("expected first_attempt_failed=true, got %v", result.Attributes["first_attempt_failed"])
	}
	if result.Attributes["execution_id"] != "job-1" {
		t.Fatalf("expected execution_id=job-1, got %v", result.Attributes["execution_id"])
	}
}

func TestExecuteWithRetryCleanSuccessBookkeeping(t *testing.T) {
	executor := &flakyExecutor{failuresBeforeSuccess: 0}

	job := domain.ProbeJob{
		ID:            "job-4",
		MonitorID:     "monitor-4",
		Type:          domain.MonitorHTTP,
		TimeoutMillis: 1000,
		Retries:       2,
	}

	result := ExecuteWithRetry(context.Background(), executor, job)

	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
	if result.Attributes["attempts"] != 1 {
		t.Fatalf("expected attempts=1, got %v", result.Attributes["attempts"])
	}
	if result.Attributes["first_attempt_failed"] != false {
		t.Fatalf("expected first_attempt_failed=false, got %v", result.Attributes["first_attempt_failed"])
	}
	if result.Attributes["execution_id"] != "job-4" {
		t.Fatalf("expected execution_id=job-4, got %v", result.Attributes["execution_id"])
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
	if result.Attributes["attempts"] != 2 {
		t.Fatalf("expected attempts=2 on exhausted retries, got %v", result.Attributes["attempts"])
	}
	if result.Attributes["first_attempt_failed"] != true {
		t.Fatalf("expected first_attempt_failed=true, got %v", result.Attributes["first_attempt_failed"])
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
