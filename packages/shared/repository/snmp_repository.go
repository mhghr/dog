package repository

import (
	"context"

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
}
