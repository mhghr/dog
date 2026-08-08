package agents

import "errors"

var (
	ErrAgentNotFound     = errors.New("agent not found")
	ErrInvalidToken      = errors.New("enrollment token is invalid, expired, or already used")
	ErrLeaseNotFound     = errors.New("lease not found or expired")
	ErrInvalidTransition = errors.New("invalid agent state transition")
)
