package repository

import (
	"context"
	"time"

	"monitoring-platform/internal/domain"
)

type UserRepository interface {
	GetByID(ctx context.Context, id string) (domain.User, error)
	// UpsertGoogleUser links by google_id first, then by verified email,
	// otherwise creates the user (signup-less first login).
	UpsertGoogleUser(ctx context.Context, googleID, email, name, avatarURL string) (domain.User, error)
	// UpsertPhoneUser finds or creates a user by normalized phone number.
	UpsertPhoneUser(ctx context.Context, phone string) (domain.User, error)
	// SetOrganizationID updates the user's organization membership.
	SetOrganizationID(ctx context.Context, userID, orgID string) error
}

type RefreshTokenRepository interface {
	Create(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error
	// Rotate atomically revokes the old token and stores the new one.
	// Returns domain.ErrTokenReused when a revoked token is replayed (all
	// user sessions are revoked in that case) and domain.ErrTokenExpired
	// for expired tokens.
	Rotate(ctx context.Context, oldHash, newHash string, expiresAt time.Time) (string, error)
	Revoke(ctx context.Context, tokenHash string) error
}

type OTPRepository interface {
	Create(ctx context.Context, phone, codeHash string, expiresAt time.Time) error
	CountSince(ctx context.Context, phone string, since time.Time) (int, error)
	LatestActive(ctx context.Context, phone string) (domain.OTPCode, error)
	RegisterFailure(ctx context.Context, id string) error
	Consume(ctx context.Context, id string) error
}
