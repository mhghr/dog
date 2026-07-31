package bootstrap_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"monitoring-platform/internal/monitoring/bootstrap"
)

func TestBootstrapSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/monitoring/bootstrap" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bootstrap.BootstrapResponse{
			AgentID:           "ag_test",
			AgentSecret:       "secret123",
			ConfigURL:         "/api/v1/monitoring/agents/ag_test/config",
			HeartbeatURL:      "/api/v1/monitoring/agents/ag_test/heartbeat",
			ConfigPollSeconds: 60,
		})
	}))
	defer server.Close()

	client := bootstrap.NewClient(bootstrap.ClientConfig{
		ServerURL:      server.URL,
		BootstrapToken: "bt_test",
		Version:        "1.0.0",
	})

	result, err := client.Bootstrap(context.Background(), "web-01", "linux", "amd64", []string{"cpu"})
	if err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if result.AgentID != "ag_test" {
		t.Errorf("AgentID = %q, want %q", result.AgentID, "ag_test")
	}
	if result.AgentSecret != "secret123" {
		t.Errorf("AgentSecret = %q, want %q", result.AgentSecret, "secret123")
	}
	if result.ConfigURL != "/api/v1/monitoring/agents/ag_test/config" {
		t.Errorf("ConfigURL = %q, want %q", result.ConfigURL, "/api/v1/monitoring/agents/ag_test/config")
	}
	if result.HeartbeatURL != "/api/v1/monitoring/agents/ag_test/heartbeat" {
		t.Errorf("HeartbeatURL = %q, want %q", result.HeartbeatURL, "/api/v1/monitoring/agents/ag_test/heartbeat")
	}
	if result.ConfigPollSeconds != 60 {
		t.Errorf("ConfigPollSeconds = %d, want %d", result.ConfigPollSeconds, 60)
	}
}

func TestBootstrapUnauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := bootstrap.NewClient(bootstrap.ClientConfig{
		ServerURL:      server.URL,
		BootstrapToken: "bt_bad",
		Version:        "1.0.0",
	})

	result, err := client.Bootstrap(context.Background(), "web-01", "linux", "amd64", []string{"cpu"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Errorf("error = %q, want it to mention status 401", err.Error())
	}
}

func TestBootstrapMissingCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bootstrap.BootstrapResponse{
			AgentID: "",
		})
	}))
	defer server.Close()

	client := bootstrap.NewClient(bootstrap.ClientConfig{
		ServerURL:      server.URL,
		BootstrapToken: "bt_test",
		Version:        "1.0.0",
	})

	result, err := client.Bootstrap(context.Background(), "web-01", "linux", "amd64", []string{"cpu"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if !strings.Contains(err.Error(), "missing agent credentials") {
		t.Errorf("error = %q, want it to mention missing agent credentials", err.Error())
	}
}

func TestBootstrapHitsCorrectPath(t *testing.T) {
	hit := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/monitoring/bootstrap" {
			hit = true
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(bootstrap.BootstrapResponse{
			AgentID:     "ag_test",
			AgentSecret: "secret123",
		})
	}))
	defer server.Close()

	client := bootstrap.NewClient(bootstrap.ClientConfig{
		ServerURL:      server.URL,
		BootstrapToken: "bt_test",
		Version:        "1.0.0",
	})

	if _, err := client.Bootstrap(context.Background(), "web-01", "linux", "amd64", []string{"cpu"}); err != nil {
		t.Fatalf("Bootstrap() error = %v", err)
	}
	if !hit {
		t.Error("client did not hit /api/v1/monitoring/bootstrap")
	}
}
