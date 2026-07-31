package updater

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	agentupdater "monitoring-platform/internal/agent/updater"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestPlatformKey(t *testing.T) {
	key := platformKey()
	if key == "" {
		t.Fatal("platformKey() returned an empty string")
	}
	if !strings.Contains(key, "/") {
		t.Fatalf("platformKey() = %q, want a string containing '/'", key)
	}
}

func TestNewUpdater(t *testing.T) {
	u := NewUpdater("1.0.0", "stable", "http://localhost:8080", t.TempDir()+"/agent", discardLogger())
	if u == nil {
		t.Fatal("NewUpdater returned nil")
	}
	if u.currentVersion != "1.0.0" {
		t.Fatalf("currentVersion = %q, want %q", u.currentVersion, "1.0.0")
	}
	if u.channel != "stable" {
		t.Fatalf("channel = %q, want %q", u.channel, "stable")
	}
	if u.checkNow == nil {
		t.Fatal("checkNow not defaulted to real checker")
	}
	if u.apply == nil {
		t.Fatal("apply not defaulted to real ApplyUpdate")
	}
}

func TestUpdaterAppliesUpdate(t *testing.T) {
	artifactData := []byte("fake-binary-content")
	sum := sha256.Sum256(artifactData)
	expectedSHA := hex.EncodeToString(sum[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(artifactData)
	}))
	defer server.Close()

	var applied []byte
	var appliedPath string
	u := &Updater{
		currentVersion: "1.0.0",
		channel:        "stable",
		binaryPath:     t.TempDir() + "/agent",
		logger:         discardLogger(),
		checkNow: func(currentVersion, channel, osArch string) (*agentupdater.ReleaseManifest, error) {
			return &agentupdater.ReleaseManifest{
				Version: "2.0.0",
				Channel: "stable",
				Artifacts: map[string]agentupdater.ArtifactRef{
					osArch: {URL: server.URL, SHA256: expectedSHA},
				},
			}, nil
		},
		apply: func(data []byte, path string) error {
			applied = data
			appliedPath = path
			return nil
		},
	}

	u.check(t.Context())

	if applied == nil {
		t.Fatal("apply was not called")
	}
	if string(applied) != string(artifactData) {
		t.Fatalf("applied data = %q, want %q", applied, artifactData)
	}
	if appliedPath != u.binaryPath {
		t.Fatalf("applied path = %q, want %q", appliedPath, u.binaryPath)
	}
}

func TestCheckNoManifestSkipsApply(t *testing.T) {
	u := &Updater{
		currentVersion: "1.0.0",
		channel:        "stable",
		binaryPath:     t.TempDir() + "/agent",
		logger:         discardLogger(),
		checkNow: func(currentVersion, channel, osArch string) (*agentupdater.ReleaseManifest, error) {
			return nil, nil
		},
		apply: func(data []byte, path string) error {
			t.Fatal("apply should not be called when no update is available")
			return nil
		},
	}

	u.check(t.Context())
}

func TestChecksumMismatchSkipsApply(t *testing.T) {
	artifactData := []byte("fake-binary-content")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(artifactData)
	}))
	defer server.Close()

	u := &Updater{
		currentVersion: "1.0.0",
		channel:        "stable",
		binaryPath:     t.TempDir() + "/agent",
		logger:         discardLogger(),
		checkNow: func(currentVersion, channel, osArch string) (*agentupdater.ReleaseManifest, error) {
			return &agentupdater.ReleaseManifest{
				Version: "2.0.0",
				Channel: "stable",
				Artifacts: map[string]agentupdater.ArtifactRef{
					osArch: {URL: server.URL, SHA256: "0badbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadbadb"},
				},
			}, nil
		},
		apply: func(data []byte, path string) error {
			t.Fatal("apply should not be called when the checksum mismatches")
			return nil
		},
	}

	u.check(t.Context())
}
