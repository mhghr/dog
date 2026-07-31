package credential

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AgentCredentials are the agent's identity and endpoints.
type AgentCredentials struct {
	AgentID      string `json:"agent_id"`
	Secret       string `json:"secret_encrypted"`
	ServerURL    string `json:"server_url"`
	ConfigURL    string `json:"config_url"`
	HeartbeatURL string `json:"heartbeat_url"`
}

// Store persists AgentCredentials to disk, encrypting the secret.
type Store struct {
	stateDir string
}

// NewStore creates a store rooted at stateDir.
func NewStore(stateDir string) (*Store, error) {
	if err := os.MkdirAll(stateDir, 0700); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}
	return &Store{stateDir: stateDir}, nil
}

// Save writes credentials with the secret encrypted using the given key.
func (s *Store) Save(creds *AgentCredentials, encryptionKey []byte) error {
	credsCopy := *creds
	encrypted, err := encrypt(credsCopy.Secret, encryptionKey)
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}
	credsCopy.Secret = encrypted

	data, err := json.MarshalIndent(credsCopy, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}

	path := filepath.Join(s.stateDir, "credentials.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

// Load reads and decrypts credentials.
func (s *Store) Load(encryptionKey []byte) (*AgentCredentials, error) {
	path := filepath.Join(s.stateDir, "credentials.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	var creds AgentCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("unmarshal credentials: %w", err)
	}

	decrypted, err := decrypt(creds.Secret, encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("decrypt secret: %w", err)
	}
	creds.Secret = decrypted
	return &creds, nil
}

// Exists reports whether credentials have been persisted.
func (s *Store) Exists() bool {
	path := filepath.Join(s.stateDir, "credentials.json")
	_, err := os.Stat(path)
	return err == nil
}

func deriveKey(material []byte) []byte {
	sum := sha256.Sum256(material)
	return sum[:]
}

func encrypt(plaintext string, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(encoded string, key []byte) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
