package agents

import (
	"time"

	"github.com/google/uuid"
)

type AgentStatus string

const (
	AgentPending   AgentStatus = "pending"
	AgentApproved  AgentStatus = "approved"
	AgentActive    AgentStatus = "active"
	AgentOffline   AgentStatus = "offline"
	AgentDisabled  AgentStatus = "disabled"
	AgentRejected  AgentStatus = "rejected"
	AgentRevoked   AgentStatus = "revoked"
	AgentDraining  AgentStatus = "draining"
	AgentUpdating  AgentStatus = "updating"
)

type ProbeAgent struct {
	ID                 uuid.UUID
	LocationID         uuid.UUID
	Name               string
	Hostname           string
	MachineFingerprint string
	PublicKey          string
	CertificateSerial  string
	GatewayCert        string
	AgentSecret        string
	Version            string
	OperatingSystem    string
	Architecture       string
	PublicIP           string
	PrivateIPs         []string
	Capabilities       []string
	MaxConcurrency     int32
	RunningJobs        int32
	SpoolBytes         int64
	Latitude           *float64
	Longitude          *float64
	City               string
	Country            string
	Status             AgentStatus
	ApprovedBy         *uuid.UUID
	ApprovedAt         *time.Time
	LastSeenAt         *time.Time
	RevokedAt          *time.Time
	EnrollmentTokenID  *uuid.UUID
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type AgentCapacity struct {
	MaxConcurrency int32
	RunningJobs    int32
	AvailableSlots int32
	SpoolBytes     int64
}

func (a *ProbeAgent) GetCapacity() AgentCapacity {
	available := a.MaxConcurrency - a.RunningJobs
	if available < 0 {
		available = 0
	}
	return AgentCapacity{
		MaxConcurrency: a.MaxConcurrency,
		RunningJobs:    a.RunningJobs,
		AvailableSlots: available,
		SpoolBytes:     a.SpoolBytes,
	}
}

type EnrollmentToken struct {
	ID                  uuid.UUID
	TokenHash           []byte
	RequestedLocationID uuid.UUID
	ExpiresAt           time.Time
	UsedAt              *time.Time
	CreatedBy           uuid.UUID
	CreatedAt           time.Time
}

type AuditEntry struct {
	ID            uuid.UUID
	AgentID       uuid.UUID
	ActorUserID   *uuid.UUID
	Action        string
	PreviousState []byte
	NextState     []byte
	RemoteIP      string
	CreatedAt     time.Time
}

type EnrollRequest struct {
	EnrollmentToken    string   `json:"enrollment_token"`
	Hostname           string   `json:"hostname"`
	MachineFingerprint string   `json:"machine_fingerprint"`
	PublicKey          string   `json:"public_key"`
	Version            string   `json:"version"`
	OperatingSystem    string   `json:"operating_system"`
	Architecture       string   `json:"architecture"`
	PrivateIPs         []string `json:"private_ips"`
	Capabilities       []string `json:"capabilities"`
	MaxConcurrency     int32    `json:"max_concurrency"`
	RequestedLocation  string   `json:"requested_location,omitempty"`
}

type EnrollResponse struct {
	RequestID uuid.UUID
	Status    AgentStatus
	Message   string
}

var validTransitions = map[AgentStatus][]AgentStatus{
	AgentPending:  {AgentApproved, AgentRejected},
	AgentApproved: {AgentActive, AgentDisabled, AgentRevoked},
	AgentActive:   {AgentOffline, AgentDisabled, AgentRevoked, AgentDraining, AgentUpdating},
	AgentOffline:  {AgentActive, AgentDisabled, AgentRevoked},
	AgentDisabled: {AgentApproved},
	AgentRejected: {},
	AgentRevoked:  {},
	AgentDraining: {AgentOffline, AgentActive},
	AgentUpdating: {AgentActive, AgentOffline},
}

func CanTransition(from, to AgentStatus) bool {
	targets, ok := validTransitions[from]
	if !ok {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

func IsFinalStatus(s AgentStatus) bool {
	return s == AgentRejected || s == AgentRevoked
}

func IsOperational(s AgentStatus) bool {
	return s == AgentActive || s == AgentDraining || s == AgentUpdating
}
