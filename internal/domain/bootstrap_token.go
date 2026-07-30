package domain

import "time"

type BootstrapToken struct {
	ID          string
	TenantID    string
	TokenHash   string
	Description string
	ExpiresAt   time.Time
	UsedAt      *time.Time
	RevokedAt   *time.Time
	CreatedBy   *string
	CreatedAt   time.Time
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
