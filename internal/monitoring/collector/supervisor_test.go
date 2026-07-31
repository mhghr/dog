package collector

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"
)

func TestNewSupervisorConfigPath(t *testing.T) {
	dir := t.TempDir()
	s := NewSupervisor(dir, "otelcol", slog.New(slog.DiscardHandler))

	if got, want := s.configPath, filepath.Join(dir, "otel-config.yaml"); got != want {
		t.Errorf("configPath = %q, want %q", got, want)
	}
	if s.binaryPath != "otelcol" {
		t.Errorf("binaryPath = %q, want otelcol", s.binaryPath)
	}
}

func TestReloadWritesConfig(t *testing.T) {
	dir := t.TempDir()
	s := NewSupervisor(dir, "otelcol", slog.New(slog.DiscardHandler))

	const want = "receivers:\n  hostmetrics:\n"
	if err := s.Reload(want); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "otel-config.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if string(got) != want {
		t.Errorf("config content = %q, want %q", string(got), want)
	}
}

func TestFindBinaryMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())

	if _, err := FindBinary(); err == nil {
		t.Fatal("FindBinary succeeded, want error when otelcol is not on PATH")
	}
}
