package repository

import (
	"context"
	"encoding/json"
	"time"

	"monitoring-platform/packages/shared/domain"
)

type SNMPRepository interface {
	CreateCredential(ctx context.Context, cred *domain.SNMPCredential) error
	GetCredential(ctx context.Context, id string) (domain.SNMPCredential, error)
	ListCredentials(ctx context.Context, workspaceID string) ([]domain.SNMPCredential, error)
	UpdateCredential(ctx context.Context, cred *domain.SNMPCredential) error
	DeleteCredential(ctx context.Context, id string) error

	CreateDevice(ctx context.Context, dev *domain.SNMPDevice) error
	GetDevice(ctx context.Context, id string) (domain.SNMPDevice, error)
	ListDevicesByResource(ctx context.Context, resourceID string) ([]domain.SNMPDevice, error)
	UpdateDevice(ctx context.Context, dev *domain.SNMPDevice) error
	DeleteDevice(ctx context.Context, id string) error

	// Discovery cache.
	UpsertDiscovery(ctx context.Context, monitorID string, discovery *domain.SNMPDiscoveryResult) error
	GetDiscovery(ctx context.Context, monitorID string) (domain.SNMPDiscoveryResult, error)

	// Per-interface monitoring policy.
	UpsertInterface(ctx context.Context, row *domain.SNMPInterfaceRow) error
	ListInterfaces(ctx context.Context, monitorID string) ([]domain.SNMPInterfaceRow, error)
	BulkUpsertInterfaces(ctx context.Context, monitorID string, rows []domain.SNMPInterfaceRow) error

	// Event stream.
	InsertEvent(ctx context.Context, event *domain.SNMPEvent) error
	ListEvents(ctx context.Context, filter SNMPEventFilter) ([]domain.SNMPEvent, error)

	// On-demand tasks (test connection / discovery).
	CreateTask(ctx context.Context, task *domain.SNMPTask) error
	GetTask(ctx context.Context, taskID string) (domain.SNMPTask, error)
	SetTaskRunning(ctx context.Context, taskID string) error
	FinishTask(ctx context.Context, taskID string, status domain.SNMPTaskStatus, result json.RawMessage, errorMsg string) error
}

// SNMPEventFilter bounds event queries.
type SNMPEventFilter struct {
	ResourceID string
	MonitorID  string
	Kind       string
	Limit      int
	Since      time.Time
}
