package ingest_test

import (
	"errors"
	"testing"
	"time"

	"monitoring-platform/internal/telemetry/ingest"
)

var errVMUnreachable = errors.New("vm unreachable")

func TestCircuitBreaker_ClosedStaysClosedOnSuccess(t *testing.T) {
	cb := ingest.NewCircuitBreaker(ingest.CircuitBreakerConfig{
		FailureThreshold: 3,
		OpenDuration:     100 * time.Millisecond,
		HalfOpenMaxReqs:  1,
	})
	if cb.State() != ingest.CBClosed {
		t.Fatal("expected closed state initially")
	}
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cb.State() != ingest.CBClosed {
		t.Fatal("expected closed after success")
	}
}

func TestCircuitBreaker_OpensAfterConsecutiveFailures(t *testing.T) {
	cb := ingest.NewCircuitBreaker(ingest.CircuitBreakerConfig{
		FailureThreshold: 3,
		OpenDuration:     100 * time.Millisecond,
		HalfOpenMaxReqs:  1,
	})
	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error { return errVMUnreachable })
	}
	if cb.State() != ingest.CBOpen {
		t.Fatalf("expected open after 3 failures, got %s", cb.State())
	}
}

func TestCircuitBreaker_ReturnsErrOnOpen(t *testing.T) {
	cb := ingest.NewCircuitBreaker(ingest.CircuitBreakerConfig{
		FailureThreshold: 1,
		OpenDuration:     500 * time.Millisecond,
		HalfOpenMaxReqs:  1,
	})
	_ = cb.Execute(func() error { return errVMUnreachable })
	if cb.State() != ingest.CBOpen {
		t.Fatal("expected open state")
	}
	err := cb.Execute(func() error { return nil })
	if err == nil {
		t.Fatal("expected error when circuit is open")
	}
	if !errors.Is(err, ingest.ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got: %v", err)
	}
}

func TestCircuitBreaker_HalfOpenToClosedOnSuccess(t *testing.T) {
	cb := ingest.NewCircuitBreaker(ingest.CircuitBreakerConfig{
		FailureThreshold: 1,
		OpenDuration:     10 * time.Millisecond,
		HalfOpenMaxReqs:  1,
	})
	_ = cb.Execute(func() error { return errVMUnreachable })
	time.Sleep(20 * time.Millisecond)

	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Fatalf("unexpected error in half-open: %v", err)
	}
	if cb.State() != ingest.CBClosed {
		t.Fatal("expected closed after half-open success")
	}
}

func TestCircuitBreaker_HalfOpenToOpenOnFailure(t *testing.T) {
	cb := ingest.NewCircuitBreaker(ingest.CircuitBreakerConfig{
		FailureThreshold: 1,
		OpenDuration:     10 * time.Millisecond,
		HalfOpenMaxReqs:  1,
	})
	_ = cb.Execute(func() error { return errVMUnreachable })
	time.Sleep(20 * time.Millisecond)

	_ = cb.Execute(func() error { return errVMUnreachable })
	if cb.State() != ingest.CBOpen {
		t.Fatal("expected back to open after half-open failure")
	}
}
