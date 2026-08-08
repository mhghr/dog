package domain

import "errors"

var (
	ErrNotFound  = errors.New("resource not found")
	ErrDuplicate = errors.New("resource already exists")

	// Auth token lifecycle errors.
	ErrTokenExpired = errors.New("token expired")
	ErrTokenReused  = errors.New("token reuse detected")
)
