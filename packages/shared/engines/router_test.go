package engines

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/ingestion/messagebus"
	"monitoring-platform/packages/shared/telemetry/ingest"
)

type fakePublisher struct {
	published []messagebus.PublishOptions
}

func (f *fakePublisher) Publish(ctx context.Context, opts messagebus.PublishOptions) error {
	f.published = append(f.published, opts)
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRouterNATSModePublishesToBothSubjects(t *testing.T) {
	pub := &fakePublisher{}
	r := NewRouter(ModeNATS, ModeNATS, pub, nil, nil, nil, nil, testLogger())

	result := &domain.ProbeResult{MonitorID: "m1", JobID: "j1"}
	r.RouteResult(context.Background(), result)

	if len(pub.published) != 2 {
		t.Fatalf("expected 2 publishes, got %d", len(pub.published))
	}

	subjects := map[string]bool{}
	for _, p := range pub.published {
		subjects[p.Subject] = true
		var envelope ingest.Envelope
		if err := json.Unmarshal(p.Data, &envelope); err != nil {
			t.Fatalf("publish payload is not an envelope: %v", err)
		}
		if envelope.MonitorID != "m1" {
			t.Fatalf("envelope monitor_id = %q, want m1", envelope.MonitorID)
		}
		var payload domain.ProbeResult
		if err := envelope.UnmarshalValue(&payload); err != nil {
			t.Fatalf("envelope value is not a probe result: %v", err)
		}
		if payload.JobID != "j1" {
			t.Fatalf("envelope job_id = %q, want j1", payload.JobID)
		}
	}
	if !subjects[HealthSubject] {
		t.Errorf("expected publish to %s", HealthSubject)
	}
	if !subjects[AlertSubject] {
		t.Errorf("expected publish to %s", AlertSubject)
	}
}

func TestRouterMixedModesPublishOnlyNATSHealth(t *testing.T) {
	pub := &fakePublisher{}
	r := NewRouter(ModeNATS, ModeInline, pub, nil, nil, nil, nil, testLogger())

	r.RouteResult(context.Background(), &domain.ProbeResult{MonitorID: "m1"})

	if len(pub.published) != 1 || pub.published[0].Subject != HealthSubject {
		t.Fatalf("expected a single publish to %s, got %+v", HealthSubject, pub.published)
	}
}

func TestRouterInlineModeDoesNotPublish(t *testing.T) {
	pub := &fakePublisher{}
	r := NewRouter(ModeInline, ModeInline, pub, nil, nil, nil, nil, testLogger())

	r.RouteResult(context.Background(), &domain.ProbeResult{MonitorID: "m1"})

	if len(pub.published) != 0 {
		t.Fatalf("inline mode must not publish, got %d", len(pub.published))
	}
}

func TestRouterNilEnginesDoesNotPanic(t *testing.T) {
	r := NewRouter("", "", nil, nil, nil, nil, nil, testLogger())
	r.RouteResult(context.Background(), &domain.ProbeResult{MonitorID: "m1"})
}

func TestRouterNATSModeWithoutBusDoesNotPanic(t *testing.T) {
	r := NewRouter(ModeNATS, ModeNATS, nil, nil, nil, nil, nil, testLogger())
	r.RouteResult(context.Background(), &domain.ProbeResult{MonitorID: "m1"})
}
