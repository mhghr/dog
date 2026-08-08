package domain

import (
	"context"
	"time"
)

// NotificationChannel represents a destination for alert notifications.
type NotificationChannel struct {
	ID             string
	OrganizationID string
	WorkspaceID    *string
	Name           string
	Type           string
	Config         map[string]string
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// AlertPolicy defines rules for when alerts should trigger.
type AlertPolicy struct {
	ID                 string
	OrganizationID     string
	WorkspaceID        *string
	Name               string
	Scope              AlertPolicyScope
	Conditions         AlertConditions
	Severity           string
	OpeningFailures    int
	ResolvingSuccesses int
	CooldownSeconds    int
	RenotifySeconds    int
	Enabled            bool
	ChannelIDs         []string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Alert represents a stateful alert event.
type Alert struct {
	ID                   string
	OrganizationID       string
	WorkspaceID          *string
	ResourceID           *string
	PolicyID             string
	MonitorID            string
	MonitorName          string
	State                string
	Severity             string
	Title                string
	Description          string
	DedupKey             string
	ConsecutiveFailures  int
	ConsecutiveSuccesses int
	OpenedAt             *time.Time
	ResolvedAt           *time.Time
	AcknowledgedAt       *time.Time
	AcknowledgedBy       *string
	LastNotifiedAt       *time.Time
	NotificationCount    int
	CreatedAt            time.Time
}

// EvaluateResult is the outcome of evaluating a probe result against policies.
type EvaluateResult struct {
	AlertID     string
	OldState    string
	NewState    string
	MonitorID   string
	MonitorName string
	Severity    string
	Title       string
	ChannelIDs  []string
}

// Notification carries the payload sent to notification channels.
type Notification struct {
	AlertID   string    `json:"alert_id"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Severity  string    `json:"severity"`
	Monitor   string    `json:"monitor"`
	MonitorID string    `json:"monitor_id"`
	FiredAt   time.Time `json:"fired_at"`
}

// AlertRepository is the persistence port for alert state.
type AlertRepository interface {
	ListActivePolicies(ctx context.Context, monitorID string) ([]AlertPolicy, error)
	FindByDedup(ctx context.Context, dedupKey string) (Alert, error)
	UpsertAlert(ctx context.Context, alert *Alert) error
	ListFiring(ctx context.Context) ([]Alert, error)
	RecordNotification(ctx context.Context, alertID string) error
}

// ChannelRepository is the persistence port for notification channels.
type ChannelRepository interface {
	ListByIDs(ctx context.Context, ids []string) ([]NotificationChannel, error)
}
