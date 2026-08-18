package domain

import (
	"encoding/json"
	"time"
)

// SNMPSecurityLevel maps to the SNMP v3 security model.
type SNMPSecurityLevel string

const (
	SNMPNoAuthNoPriv SNMPSecurityLevel = "noAuthNoPriv"
	SNMPAuthNoPriv   SNMPSecurityLevel = "authNoPriv"
	SNMPAuthPriv     SNMPSecurityLevel = "authPriv"
)

type SNMPCredential struct {
	ID                       string               `json:"id"`
	WorkspaceID              string               `json:"workspace_id"`
	Name                     string               `json:"name"`
	Description              string               `json:"description"`
	Version                  string               `json:"version"`
	Community                string               `json:"community,omitempty"`
	Username                 string               `json:"username,omitempty"`
	AuthenticationProtocol   string               `json:"authentication_protocol,omitempty"`
	AuthenticationPassphrase string               `json:"-"`
	PrivacyProtocol          string               `json:"privacy_protocol,omitempty"`
	PrivacyPassphrase        string               `json:"-"`
	SecurityLevel            SNMPSecurityLevel    `json:"security_level"`
	ContextName              string               `json:"context_name,omitempty"`
	EncryptedConfig          json.RawMessage      `json:"encrypted_config"`
	CreatedAt                time.Time            `json:"created_at"`
	UpdatedAt                time.Time            `json:"updated_at"`
}

type SNMPDevice struct {
	ID              string          `json:"id"`
	ResourceID      string          `json:"resource_id"`
	CredentialID    string          `json:"credential_id"`
	Transport       string          `json:"transport"`
	Port            int             `json:"port"`
	MaxRepetitions  int             `json:"max_repetitions"`
	TimeoutSeconds  int             `json:"timeout_seconds"`
	Retries         int             `json:"retries"`
	OIDs            json.RawMessage `json:"oids"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}
