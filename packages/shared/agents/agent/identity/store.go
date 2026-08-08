package identity

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Identity struct {
	AgentID string `json:"agent_id"`
}

func SaveIdentity(stateDir, agentID, certPEM, keyPEM string) error {
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}

	identity := Identity{AgentID: agentID}
	data, err := json.MarshalIndent(identity, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal identity: %w", err)
	}

	identityPath := filepath.Join(stateDir, "identity.json")
	if err := os.WriteFile(identityPath, data, 0600); err != nil {
		return fmt.Errorf("write identity: %w", err)
	}

	if certPEM != "" {
		certPath := filepath.Join(stateDir, "agent.crt")
		if err := os.WriteFile(certPath, []byte(certPEM), 0600); err != nil {
			return fmt.Errorf("write certificate: %w", err)
		}
	}

	if keyPEM != "" {
		keyPath := filepath.Join(stateDir, "agent.key")
		if err := os.WriteFile(keyPath, []byte(keyPEM), 0600); err != nil {
			return fmt.Errorf("write private key: %w", err)
		}
	}

	return nil
}

func LoadIdentity(stateDir string) (agentID, certPEM, keyPEM string, err error) {
	identityPath := filepath.Join(stateDir, "identity.json")
	data, err := os.ReadFile(identityPath)
	if err != nil {
		return "", "", "", fmt.Errorf("read identity: %w", err)
	}

	var identity Identity
	if err := json.Unmarshal(data, &identity); err != nil {
		return "", "", "", fmt.Errorf("unmarshal identity: %w", err)
	}

	certPath := filepath.Join(stateDir, "agent.crt")
	certData, err := os.ReadFile(certPath)
	if err != nil {
		return identity.AgentID, "", "", fmt.Errorf("read certificate: %w", err)
	}

	keyPath := filepath.Join(stateDir, "agent.key")
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return identity.AgentID, string(certData), "", fmt.Errorf("read private key: %w", err)
	}

	return identity.AgentID, string(certData), string(keyData), nil
}

func HasIdentity(stateDir string) bool {
	identityPath := filepath.Join(stateDir, "identity.json")
	_, err := os.Stat(identityPath)
	return err == nil
}

func ClearEnrollmentToken(cfg interface{}) error {
	type configWithToken interface {
		GetEnrollmentToken() string
	}
	if c, ok := cfg.(configWithToken); ok {
		_ = c.GetEnrollmentToken()
	}
	return nil
}
