package config

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGenerateOTelConfig(t *testing.T) {
	c := DefaultConfig()
	out := GenerateOTelConfig(c, "localhost:4317", "ag_1")

	for _, want := range []string{
		"collection_interval: 60s",
		"send_batch_size: 500",
		"endpoint: localhost:4317",
		`x-agent-id: "ag_1"`,
		"cpu: {}",
		"memory: {}",
		"filesystem: {}",
		"network: {}",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("OTel config missing %q in:\n%s", want, out)
		}
	}
}

func TestGenerateOTelConfigDisablesCompression(t *testing.T) {
	c := DefaultConfig()
	c.Compress = false
	out := GenerateOTelConfig(c, "localhost:4317", "ag_1")
	if !strings.Contains(out, `compression: "none"`) {
		t.Errorf("expected compression none, got:\n%s", out)
	}
}

func TestSanitizeConfig(t *testing.T) {
	c := &AgentConfig{
		CollectionIntervalSeconds: 5,
		BatchSize:                 0,
		MaxMetricsPerBatch:        50,
		MaxLabelCount:             0,
	}
	SanitizeConfig(c)

	if c.CollectionIntervalSeconds != 10 {
		t.Errorf("CollectionIntervalSeconds = %d, want 10", c.CollectionIntervalSeconds)
	}
	if c.BatchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", c.BatchSize)
	}
	if c.MaxMetricsPerBatch != 100 {
		t.Errorf("MaxMetricsPerBatch = %d, want 100", c.MaxMetricsPerBatch)
	}
	if c.MaxLabelCount != 10 {
		t.Errorf("MaxLabelCount = %d, want 10", c.MaxLabelCount)
	}
	if len(c.EnabledReceivers) != 2 {
		t.Errorf("EnabledReceivers = %v, want default [cpu memory]", c.EnabledReceivers)
	}
	if c.FeatureFlags == nil {
		t.Error("FeatureFlags = nil, want empty map")
	}
	if c.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want info", c.LogLevel)
	}
}

func TestManagerFetchesNewerConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"version": 2,
			"config": map[string]any{
				"version":                     2,
				"collection_interval_seconds": 30,
				"batch_size":                  200,
				"export_interval_seconds":     30,
				"enabled_receivers":           []string{"cpu", "memory"},
				"max_metrics_per_batch":       2000,
				"max_label_count":             40,
				"max_label_length":            256,
				"feature_flags":               map[string]bool{},
				"compress":                    true,
				"log_level":                   "info",
			},
		})
	}))
	defer server.Close()

	changed := make(chan struct{})
	mgr := NewManager(server.URL, "ag_1", func() map[string]string { return map[string]string{"x-agent-id": "ag_1"} }, slog.New(slog.DiscardHandler))
	mgr.OnChange(func(old, new *AgentConfig) { close(changed) })

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mgr.Start(ctx, 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	select {
	case <-changed:
	case <-time.After(2 * time.Second):
		t.Fatal("OnChange not fired")
	}
	if got := mgr.Get().Version; got != 2 {
		t.Errorf("Version = %d, want 2", got)
	}
}

func TestManagerIgnoresOlderConfig(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"version": 0,
			"config":  DefaultConfig(),
		})
	}))
	defer server.Close()

	mgr := NewManager(server.URL, "ag_1", func() map[string]string { return map[string]string{"x-agent-id": "ag_1"} }, slog.New(slog.DiscardHandler))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mgr.Start(ctx, 50*time.Millisecond); err != nil {
		t.Fatal(err)
	}

	if got := mgr.Get().Version; got != 1 {
		t.Errorf("Version = %d, want default 1", got)
	}
}
