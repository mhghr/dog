package ingest_test

import (
	"encoding/json"
	"testing"

	"monitoring-platform/internal/telemetry/ingest"
)

func TestEnvelopeValidation_ValidProbeResult(t *testing.T) {
	env := ingest.Envelope{
		SchemaVersion: 1,
		Type:          ingest.TypeProbeResult,
		EventID:       "550e8400-e29b-41d4-a716-446655440000",
		Timestamp:     "2026-08-04T01:32:05Z",
	}
	if err := env.Validate(); err != nil {
		t.Fatalf("expected valid envelope, got: %v", err)
	}
}

func TestEnvelopeValidation_InvalidType(t *testing.T) {
	env := ingest.Envelope{
		SchemaVersion: 1,
		Type:          "invalid_type",
		EventID:       "550e8400-e29b-41d4-a716-446655440000",
	}
	err := env.Validate()
	if err == nil {
		t.Fatal("expected validation error for invalid type")
	}
}

func TestEnvelopeValidation_MissingEventID(t *testing.T) {
	env := ingest.Envelope{
		SchemaVersion: 1,
		Type:          ingest.TypeProbeResult,
	}
	err := env.Validate()
	if err == nil {
		t.Fatal("expected validation error for missing event_id")
	}
}

func TestEnvelopeValidation_UnknownSchemaVersion(t *testing.T) {
	env := ingest.Envelope{
		SchemaVersion: 99,
		Type:          ingest.TypeProbeResult,
		EventID:       "550e8400-e29b-41d4-a716-446655440000",
	}
	err := env.Validate()
	if err == nil {
		t.Fatal("expected validation error for unknown schema version")
	}
}

func TestEnvelopeSerialization_RoundTrip(t *testing.T) {
	original := ingest.Envelope{
		SchemaVersion: 1,
		Type:          ingest.TypeMetricBatch,
		EventID:       "550e8400-e29b-41d4-a716-446655440000",
		MonitorID:     "mon-123",
		ResourceID:    "res-456",
		WorkspaceID:   "ws-789",
		TenantID:      "tenant-1",
		AgentID:       "agent-1",
		Timestamp:     "2026-08-04T01:32:05Z",
		Source:        "otel-ingest",
	}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var roundtripped ingest.Envelope
	if err := json.Unmarshal(data, &roundtripped); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if roundtripped.EventID != original.EventID {
		t.Fatalf("event_id mismatch: %s != %s", roundtripped.EventID, original.EventID)
	}
	if roundtripped.Type != original.Type {
		t.Fatal("type mismatch")
	}
}
