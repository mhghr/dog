package credential

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SaveCredentials("ag_1", "topsecret", "https://core.example.com", "/config", "/heartbeat"); err != nil {
		t.Fatal(err)
	}
	creds, err := m.LoadCredentials()
	if err != nil {
		t.Fatal(err)
	}
	if creds.AgentID != "ag_1" || creds.Secret != "topsecret" {
		t.Errorf("round trip mismatch: %+v", creds)
	}
	if creds.ServerURL != "https://core.example.com" || creds.ConfigURL != "/config" || creds.HeartbeatURL != "/heartbeat" {
		t.Errorf("endpoint mismatch: %+v", creds)
	}
}

func TestHasCredentials(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if m.HasCredentials() {
		t.Fatal("HasCredentials() = true before save")
	}
	if err := m.SaveCredentials("ag_1", "topsecret", "https://core.example.com", "/config", "/heartbeat"); err != nil {
		t.Fatal(err)
	}
	if !m.HasCredentials() {
		t.Fatal("HasCredentials() = false after save")
	}
}

func TestAuthHeader(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if hdr := m.AuthHeader(); hdr != nil {
		t.Fatalf("AuthHeader before save = %v, want nil", hdr)
	}
	if err := m.SaveCredentials("ag_1", "topsecret", "https://core.example.com", "/config", "/heartbeat"); err != nil {
		t.Fatal(err)
	}
	hdr := m.AuthHeader()
	for _, key := range []string{"x-agent-id", "x-signature", "x-timestamp"} {
		if hdr[key] == "" {
			t.Errorf("AuthHeader missing %q", key)
		}
	}
	if hdr["x-agent-id"] != "ag_1" {
		t.Errorf("x-agent-id = %q, want ag_1", hdr["x-agent-id"])
	}
}

func TestSignMessageMatchesServerScheme(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := m.SignMessage("ag_1"); got != "" {
		t.Fatalf("SignMessage before save = %q, want empty", got)
	}
	if err := m.SaveCredentials("ag_1", "topsecret", "https://core.example.com", "/config", "/heartbeat"); err != nil {
		t.Fatal(err)
	}

	timestamp := "2026-07-31T12:00:00Z"
	got := m.SignMessage(m.AgentID(), timestamp)

	mac := hmac.New(sha256.New, []byte("topsecret"))
	mac.Write([]byte("ag_1"))
	mac.Write([]byte(":"))
	mac.Write([]byte(timestamp))
	want := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	if got != want {
		t.Errorf("SignMessage = %q, want server-compatible %q", got, want)
	}
}

func TestSecretEncryptedAtRest(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SaveCredentials("ag_1", "topsecret", "https://core.example.com", "/config", "/heartbeat"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "credentials.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "topsecret") {
		t.Error("raw secret found in credentials.json")
	}
}

func TestKeyPersistsAcrossManagersOnSameDir(t *testing.T) {
	dir := t.TempDir()
	m1, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m1.SaveCredentials("ag_1", "topsecret", "https://core.example.com", "/config", "/heartbeat"); err != nil {
		t.Fatal(err)
	}

	m2, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	creds, err := m2.LoadCredentials()
	if err != nil {
		t.Fatalf("load with a fresh manager on the same state dir failed: %v", err)
	}
	if creds.AgentID != "ag_1" || creds.Secret != "topsecret" {
		t.Errorf("round trip via fresh manager mismatch: %+v", creds)
	}
}

func TestEnsureKeyCreatesRandomPersistedKey(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	key, err := s.EnsureKey()
	if err != nil {
		t.Fatal(err)
	}
	if len(key) != 32 {
		t.Fatalf("EnsureKey returned %d bytes, want 32", len(key))
	}
	keyPath := filepath.Join(dir, "encryption.key")
	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatalf("key file not created: %v", err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0600 {
		t.Errorf("key file mode = %v, want 0600", info.Mode().Perm())
	}
	data, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(key) {
		t.Error("key file contents differ from returned key")
	}

	key2, err := s.EnsureKey()
	if err != nil {
		t.Fatal(err)
	}
	if string(key2) != string(key) {
		t.Error("EnsureKey returned a different key on second call")
	}
}
