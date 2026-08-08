package domain

import "time"

type ProbeJob struct {
	ID              string         `json:"id"`
	MonitorID       string         `json:"monitor_id"`
	Type            MonitorType    `json:"type"`
	Target          string         `json:"target"`
	TimeoutMillis   int            `json:"timeout_millis"`
	Retries         int            `json:"retries"`
	Config          map[string]any `json:"config"`
	ProbeLocationID string         `json:"probe_location_id"`
	ScheduledAt     time.Time      `json:"scheduled_at"`
	Deadline        time.Time      `json:"deadline"`
	LeaseID         string         `json:"lease_id,omitempty"`
	LeaseExpiresAt  time.Time      `json:"lease_expires_at,omitempty"`
	Attempt         int            `json:"attempt,omitempty"`
	ConfigVersion   string         `json:"config_version,omitempty"`
}
