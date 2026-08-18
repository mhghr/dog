package ingest

import (
	"encoding/json"
	"fmt"
)

type EnvelopeType string

const (
	TypeProbeResult      EnvelopeType = "probe_result"
	TypeMetricBatch      EnvelopeType = "metric_batch"
	TypeAlertEvent       EnvelopeType = "alert_event"
	TypeLogBatch         EnvelopeType = "log_batch"
	TypeTraceSpan        EnvelopeType = "trace_span"
	TypeSyntheticResult  EnvelopeType = "synthetic_result"

	CurrentSchemaVersion = 1
)

var validTypes = map[EnvelopeType]bool{
	TypeProbeResult:      true,
	TypeMetricBatch:      true,
	TypeAlertEvent:       true,
	TypeLogBatch:         true,
	TypeTraceSpan:        true,
	TypeSyntheticResult:  true,
}

type Envelope struct {
	SchemaVersion int             `json:"schema_version"`
	Type          EnvelopeType    `json:"type"`
	EventID       string          `json:"event_id"`
	MonitorID     string          `json:"monitor_id,omitempty"`
	ResourceID    string          `json:"resource_id,omitempty"`
	WorkspaceID   string          `json:"workspace_id,omitempty"`
	TenantID      string          `json:"tenant_id,omitempty"`
	AgentID       string          `json:"agent_id,omitempty"`
	Timestamp     string          `json:"timestamp,omitempty"`
	Source        string          `json:"source,omitempty"`
	Value         json.RawMessage `json:"value,omitempty"`
}

func (e *Envelope) Validate() error {
	if e.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unknown schema_version: %d (supported: %d)", e.SchemaVersion, CurrentSchemaVersion)
	}
	if e.EventID == "" {
		return fmt.Errorf("event_id is required")
	}
	if e.Type == "" {
		return fmt.Errorf("type is required")
	}
	if !validTypes[e.Type] {
		return fmt.Errorf("unknown type: %q", e.Type)
	}
	return nil
}

func (e *Envelope) UnmarshalValue(v interface{}) error {
	return json.Unmarshal(e.Value, v)
}

func NewEnvelope(typ EnvelopeType, eventID string, value interface{}) (*Envelope, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("marshal value: %w", err)
	}
	return &Envelope{
		SchemaVersion: CurrentSchemaVersion,
		Type:          typ,
		EventID:       eventID,
		Value:         data,
	}, nil
}

func NewProbeResultEnvelope(eventID string, source string, monitorID, resourceID, workspaceID string, value interface{}) (*Envelope, error) {
	env, err := NewEnvelope(TypeProbeResult, eventID, value)
	if err != nil {
		return nil, err
	}
	env.MonitorID = monitorID
	env.ResourceID = resourceID
	env.WorkspaceID = workspaceID
	env.Source = source
	return env, nil
}
