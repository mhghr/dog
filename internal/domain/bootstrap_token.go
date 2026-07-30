package domain

import "time"

// BootstrapToken represents a one-time registration token for agent onboarding.
type BootstrapToken struct {
	// ID is the unique identifier for this token.
	ID string
	// TenantID is the tenant that owns this token.
	TenantID string
	// TokenHash is the bcrypt hash of the raw token value.
	TokenHash string
	// Description is a human-readable label for this token.
	Description string
	// ExpiresAt is when this token ceases to be valid.
	ExpiresAt time.Time
	// UsedAt is when this token was consumed during registration, if at all.
	UsedAt *time.Time
	// RevokedAt is when this token was manually revoked, if at all.
	RevokedAt *time.Time
	// CreatedBy is the user or system that created the token.
	CreatedBy *string
	// CreatedAt is when the token was created.
	CreatedAt time.Time
}

func (t *BootstrapToken) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

func (t *BootstrapToken) IsUsed() bool {
	return t.UsedAt != nil
}

func (t *BootstrapToken) IsRevoked() bool {
	return t.RevokedAt != nil
}

func (t *BootstrapToken) IsValid() bool {
	return !t.IsExpired() && !t.IsUsed() && !t.IsRevoked()
}
