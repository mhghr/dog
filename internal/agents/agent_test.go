package agents

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCanTransition(t *testing.T) {
	tests := []struct {
		from     AgentStatus
		to       AgentStatus
		expected bool
	}{
		{AgentPending, AgentApproved, true},
		{AgentPending, AgentRejected, true},
		{AgentPending, AgentActive, false},
		{AgentPending, AgentOffline, false},
		{AgentPending, AgentDraining, false},

		{AgentApproved, AgentActive, true},
		{AgentApproved, AgentDisabled, true},
		{AgentApproved, AgentRevoked, true},
		{AgentApproved, AgentPending, false},
		{AgentApproved, AgentRejected, false},

		{AgentActive, AgentOffline, true},
		{AgentActive, AgentDisabled, true},
		{AgentActive, AgentRevoked, true},
		{AgentActive, AgentDraining, true},
		{AgentActive, AgentUpdating, true},
		{AgentActive, AgentPending, false},
		{AgentActive, AgentApproved, false},

		{AgentOffline, AgentActive, true},
		{AgentOffline, AgentDisabled, true},
		{AgentOffline, AgentRevoked, true},
		{AgentOffline, AgentDraining, false},
		{AgentOffline, AgentApproved, false},

		{AgentDisabled, AgentApproved, true},
		{AgentDisabled, AgentActive, false},
		{AgentDisabled, AgentPending, false},

		{AgentRejected, AgentActive, false},
		{AgentRejected, AgentApproved, false},
		{AgentRejected, AgentPending, false},

		{AgentRevoked, AgentActive, false},
		{AgentRevoked, AgentApproved, false},
		{AgentRevoked, AgentPending, false},

		{AgentDraining, AgentOffline, true},
		{AgentDraining, AgentActive, true},
		{AgentDraining, AgentPending, false},
		{AgentDraining, AgentApproved, false},

		{AgentUpdating, AgentActive, true},
		{AgentUpdating, AgentOffline, true},
		{AgentUpdating, AgentPending, false},
		{AgentUpdating, AgentDraining, false},
	}

	for _, tt := range tests {
		result := CanTransition(tt.from, tt.to)
		if result != tt.expected {
			t.Errorf("CanTransition(%s, %s) = %v, want %v", tt.from, tt.to, result, tt.expected)
		}
	}
}

func TestIsFinalStatus(t *testing.T) {
	tests := []struct {
		status   AgentStatus
		expected bool
	}{
		{AgentPending, false},
		{AgentApproved, false},
		{AgentActive, false},
		{AgentOffline, false},
		{AgentDisabled, false},
		{AgentRejected, true},
		{AgentRevoked, true},
		{AgentDraining, false},
		{AgentUpdating, false},
	}

	for _, tt := range tests {
		result := IsFinalStatus(tt.status)
		if result != tt.expected {
			t.Errorf("IsFinalStatus(%s) = %v, want %v", tt.status, result, tt.expected)
		}
	}
}

func TestIsOperational(t *testing.T) {
	tests := []struct {
		status   AgentStatus
		expected bool
	}{
		{AgentPending, false},
		{AgentApproved, false},
		{AgentActive, true},
		{AgentOffline, false},
		{AgentDisabled, false},
		{AgentRejected, false},
		{AgentRevoked, false},
		{AgentDraining, true},
		{AgentUpdating, true},
	}

	for _, tt := range tests {
		result := IsOperational(tt.status)
		if result != tt.expected {
			t.Errorf("IsOperational(%s) = %v, want %v", tt.status, result, tt.expected)
		}
	}
}

func TestGetCapacity(t *testing.T) {
	agent := ProbeAgent{
		MaxConcurrency: 10,
		RunningJobs:    4,
		SpoolBytes:     1024,
	}

	cap := agent.GetCapacity()
	if cap.MaxConcurrency != 10 {
		t.Errorf("MaxConcurrency = %d, want 10", cap.MaxConcurrency)
	}
	if cap.RunningJobs != 4 {
		t.Errorf("RunningJobs = %d, want 4", cap.RunningJobs)
	}
	if cap.AvailableSlots != 6 {
		t.Errorf("AvailableSlots = %d, want 6", cap.AvailableSlots)
	}
	if cap.SpoolBytes != 1024 {
		t.Errorf("SpoolBytes = %d, want 1024", cap.SpoolBytes)
	}
}

func TestGetCapacityOverProvisioned(t *testing.T) {
	agent := ProbeAgent{
		MaxConcurrency: 10,
		RunningJobs:    12,
	}

	cap := agent.GetCapacity()
	if cap.AvailableSlots != 0 {
		t.Errorf("AvailableSlots = %d, want 0 (floor at zero)", cap.AvailableSlots)
	}
}

func TestEnrollmentTokenExpiry(t *testing.T) {
	now := time.Now().UTC()

	futureToken := EnrollmentToken{
		ID:        uuid.New(),
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if futureToken.ExpiresAt.Before(now) {
		t.Error("future token should not be expired")
	}

	pastToken := EnrollmentToken{
		ID:        uuid.New(),
		ExpiresAt: now.Add(-24 * time.Hour),
	}
	if !pastToken.ExpiresAt.Before(now) {
		t.Error("past token should be expired")
	}
}

func TestEnrollmentTokenUsed(t *testing.T) {
	now := time.Now().UTC()

	unused := EnrollmentToken{
		ID:        uuid.New(),
		ExpiresAt: now.Add(24 * time.Hour),
	}
	if unused.UsedAt != nil {
		t.Error("new token should not be marked as used")
	}

	used := EnrollmentToken{
		ID:        uuid.New(),
		ExpiresAt: now.Add(24 * time.Hour),
		UsedAt:    &now,
	}
	if used.UsedAt == nil {
		t.Error("used token should have a used_at timestamp")
	}
}

func TestAgentStatusConstants(t *testing.T) {
	if string(AgentPending) != "pending" {
		t.Errorf("AgentPending = %q, want %q", AgentPending, "pending")
	}
	if string(AgentApproved) != "approved" {
		t.Errorf("AgentApproved = %q, want %q", AgentApproved, "approved")
	}
	if string(AgentActive) != "active" {
		t.Errorf("AgentActive = %q, want %q", AgentActive, "active")
	}
	if string(AgentOffline) != "offline" {
		t.Errorf("AgentOffline = %q, want %q", AgentOffline, "offline")
	}
	if string(AgentDisabled) != "disabled" {
		t.Errorf("AgentDisabled = %q, want %q", AgentDisabled, "disabled")
	}
	if string(AgentRejected) != "rejected" {
		t.Errorf("AgentRejected = %q, want %q", AgentRejected, "rejected")
	}
	if string(AgentRevoked) != "revoked" {
		t.Errorf("AgentRevoked = %q, want %q", AgentRevoked, "revoked")
	}
	if string(AgentDraining) != "draining" {
		t.Errorf("AgentDraining = %q, want %q", AgentDraining, "draining")
	}
	if string(AgentUpdating) != "updating" {
		t.Errorf("AgentUpdating = %q, want %q", AgentUpdating, "updating")
	}
}

func TestInvalidTransitionRejectedToActive(t *testing.T) {
	if CanTransition(AgentRejected, AgentActive) {
		t.Error("should not be able to transition from rejected to active")
	}
}

func TestInvalidTransitionRejectedToApproved(t *testing.T) {
	if CanTransition(AgentRejected, AgentApproved) {
		t.Error("should not be able to transition from rejected to approved")
	}
}

func TestInvalidTransitionRevokedToActive(t *testing.T) {
	if CanTransition(AgentRevoked, AgentActive) {
		t.Error("should not be able to transition from revoked to active")
	}
}

func TestValidTransitionActiveToDraining(t *testing.T) {
	if !CanTransition(AgentActive, AgentDraining) {
		t.Error("should be able to transition from active to draining")
	}
}

func TestValidTransitionDrainingToOffline(t *testing.T) {
	if !CanTransition(AgentDraining, AgentOffline) {
		t.Error("should be able to transition from draining to offline")
	}
}

func TestValidTransitionDrainingToActive(t *testing.T) {
	if !CanTransition(AgentDraining, AgentActive) {
		t.Error("should be able to transition from draining back to active")
	}
}

func TestLeaseCreation(t *testing.T) {
	id := uuid.New()
	if id == uuid.Nil {
		t.Error("lease ID should not be nil")
	}
}

func TestCapacityTracking(t *testing.T) {
	tests := []struct {
		maxConcurrency int32
		runningJobs    int32
		wantSlots      int32
	}{
		{50, 0, 50},
		{50, 10, 40},
		{50, 50, 0},
		{50, 60, 0},
		{1, 0, 1},
		{1, 1, 0},
	}

	for _, tt := range tests {
		agent := ProbeAgent{
			MaxConcurrency: tt.maxConcurrency,
			RunningJobs:    tt.runningJobs,
		}
		cap := agent.GetCapacity()
		if cap.AvailableSlots != tt.wantSlots {
			t.Errorf("MaxConcurrency=%d RunningJobs=%d => AvailableSlots=%d, want %d",
				tt.maxConcurrency, tt.runningJobs, cap.AvailableSlots, tt.wantSlots)
		}
	}
}
