package repository

import (
	"context"
	"monitoring-platform/internal/domain"
)

type BootstrapTokenRepository interface {
	Create(ctx context.Context, token *domain.BootstrapToken) error
	GetByTokenHash(ctx context.Context, hash string) (*domain.BootstrapToken, error)
	MarkUsed(ctx context.Context, tokenID string) error
	MarkRevoked(ctx context.Context, tokenID string) error
	DeleteExpired(ctx context.Context) (int64, error)
}
