package engines

import (
	"context"
	"encoding/json"
	"testing"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/ingestion/messagebus"
	"monitoring-platform/packages/shared/telemetry/ingest"
)

func TestProcessResultMessageDecodesEnvelope(t *testing.T) {
	result := &domain.ProbeResult{MonitorID: "m1", JobID: "j1"}
	envelope, err := ingest.NewProbeResultEnvelope("evt-1", "test", "m1", "", "", result)
	if err != nil {
		t.Fatalf("envelope: %v", err)
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got *domain.ProbeResult
	err = processResultMessage(context.Background(), messagebus.Message{Subject: HealthSubject, Data: data}, func(ctx context.Context, r *domain.ProbeResult) error {
		got = r
		return nil
	}, testLogger())
	if err != nil {
		t.Fatalf("processResultMessage: %v", err)
	}
	if got == nil || got.MonitorID != "m1" || got.JobID != "j1" {
		t.Fatalf("unexpected decoded result: %+v", got)
	}
}

func TestProcessResultMessageHandlerErrorPropagates(t *testing.T) {
	result := &domain.ProbeResult{MonitorID: "m1"}
	envelope, _ := ingest.NewProbeResultEnvelope("evt-1", "test", "m1", "", "", result)
	data, _ := json.Marshal(envelope)

	err := processResultMessage(context.Background(), messagebus.Message{Subject: AlertSubject, Data: data}, func(ctx context.Context, r *domain.ProbeResult) error {
		return context.DeadlineExceeded
	}, testLogger())
	if err == nil {
		t.Fatal("expected handler error to propagate for redelivery")
	}
}

func TestProcessResultMessageMalformedIsAcked(t *testing.T) {
	handled := false
	err := processResultMessage(context.Background(), messagebus.Message{Subject: HealthSubject, Data: []byte("not-json")}, func(ctx context.Context, r *domain.ProbeResult) error {
		handled = true
		return nil
	}, testLogger())
	if err != nil {
		t.Fatalf("malformed payload should be acked (nil error), got %v", err)
	}
	if handled {
		t.Fatal("handler must not run for a malformed payload")
	}
}

func TestProcessResultMessageInvalidEnvelopeIsAcked(t *testing.T) {
	handled := false
	data, _ := json.Marshal(map[string]any{"schema_version": 99, "event_id": "", "type": "nope"})
	err := processResultMessage(context.Background(), messagebus.Message{Subject: HealthSubject, Data: data}, func(ctx context.Context, r *domain.ProbeResult) error {
		handled = true
		return nil
	}, testLogger())
	if err != nil {
		t.Fatalf("invalid envelope should be acked (nil error), got %v", err)
	}
	if handled {
		t.Fatal("handler must not run for an invalid envelope")
	}
}
