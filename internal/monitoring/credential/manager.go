package credential

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"time"
)

// Manager owns the agent's credentials, encrypted storage, and request signing.
type Manager struct {
	store *Store
	creds *AgentCredentials
	key   []byte
}

// NewManager creates a manager for the given state directory.
func NewManager(stateDir string) (*Manager, error) {
	store, err := NewStore(stateDir)
	if err != nil {
		return nil, err
	}

	hostname, _ := os.Hostname()
	machineID := fmt.Sprintf("monitoring-agent-%s", hostname)
	key := deriveKey([]byte(machineID))

	return &Manager{store: store, key: key}, nil
}

// SaveCredentials persists the credentials returned by bootstrap.
func (m *Manager) SaveCredentials(agentID, secret, serverURL, configURL, heartbeatURL string) error {
	m.creds = &AgentCredentials{
		AgentID:      agentID,
		Secret:       secret,
		ServerURL:    serverURL,
		ConfigURL:    configURL,
		HeartbeatURL: heartbeatURL,
	}
	return m.store.Save(m.creds, m.key)
}

// LoadCredentials loads and decrypts the stored credentials.
func (m *Manager) LoadCredentials() (*AgentCredentials, error) {
	creds, err := m.store.Load(m.key)
	if err != nil {
		return nil, err
	}
	m.creds = creds
	return creds, nil
}

// HasCredentials reports whether credentials exist on disk.
func (m *Manager) HasCredentials() bool {
	return m.store.Exists()
}

// AgentID returns the current agent ID, or empty if not loaded.
func (m *Manager) AgentID() string {
	if m.creds == nil {
		return ""
	}
	return m.creds.AgentID
}

// Secret returns the current raw secret, or empty if not loaded.
func (m *Manager) Secret() string {
	if m.creds == nil {
		return ""
	}
	return m.creds.Secret
}

// SignMessage computes base64url(HMAC-SHA256(secret, parts joined by ":")).
// Parts are joined WITHOUT a trailing colon to match the server-side
// computeSignature scheme ("agent_id:timestamp").
func (m *Manager) SignMessage(parts ...string) string {
	if m.creds == nil {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(m.creds.Secret))
	for i, p := range parts {
		if i > 0 {
			mac.Write([]byte(":"))
		}
		mac.Write([]byte(p))
	}
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// AuthHeader builds the authenticated headers for API requests.
// Matches the server-side HMAC scheme: signature over "agent_id:timestamp".
func (m *Manager) AuthHeader() map[string]string {
	if m.creds == nil {
		return nil
	}
	timestamp := time.Now().UTC().Format(time.RFC3339)
	return map[string]string{
		"x-agent-id":  m.creds.AgentID,
		"x-signature": m.SignMessage(m.creds.AgentID, timestamp),
		"x-timestamp": timestamp,
	}
}
