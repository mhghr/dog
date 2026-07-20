package domain

import "time"

type User struct {
	ID             string
	GoogleID       string
	Email          string
	Phone          string
	Name           string
	AvatarURL      string
	OrganizationID string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastLoginAt    *time.Time
}

type OTPCode struct {
	ID        string
	Phone     string
	CodeHash  string
	ExpiresAt time.Time
	Attempts  int
	CreatedAt time.Time
}
