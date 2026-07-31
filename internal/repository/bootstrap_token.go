package repository

import (
	"context"
	"monitoring-platform/internal/domain"
)

// BootstrapTokenRepository persists bootstrap tokens for agent registration.
type BootstrapTokenRepository interface {
	// Create persists a new bootstrap token.
	Create(ctx context.Context, token *domain.BootstrapToken) error
	// GetByTokenHash retrieves a token by its SHA-256 hex hash.
	GetByTokenHash(ctx context.Context, hash string) (domain.BootstrapToken, error)
	// MarkUsed marks a token as consumed during registration.
	MarkUsed(ctx context.Context, tokenID string) error
	// MarkRevoked marks a token as manually revoked.
	MarkRevoked(ctx context.Context, tokenID string) error
	// ListByTenant returns a paginated list of tokens for a tenant along with the total count.
	ListByTenant(ctx context.Context, tenantID string, limit, offset int) ([]domain.BootstrapToken, int, error)
	// DeleteExpired removes all expired tokens and returns the number deleted.
	DeleteExpired(ctx context.Context) (int64, error)
}
