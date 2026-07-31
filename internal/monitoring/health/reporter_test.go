package health

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestReporterSendsHeartbeat(t *testing.T) {
	var mu sync.Mutex
	received := map[string]any{}
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		mu.Lock()
		gotAuth = r.Header.Get("x-agent-id")
		json.NewDecoder(r.Body).Decode(&received)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	monitor := NewMonitor(slog.New(slog.DiscardHandler))
	authHeaders := func() map[string]string {
		return map[string]string{"x-agent-id": "ag_test", "x-timestamp": time.Now().UTC().Format(time.RFC3339)}
	}
	reporter := NewReporter(server.URL, authHeaders, monitor, slog.New(slog.DiscardHandler))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	reporter.sendHeartbeat(ctx)

	mu.Lock()
	defer mu.Unlock()
	if gotAuth != "ag_test" {
		t.Errorf("x-agent-id = %q, want %q", gotAuth, "ag_test")
	}
	if _, ok := received["cpu_percent"]; !ok {
		t.Errorf("heartbeat body missing cpu_percent: %v", received)
	}
	if _, ok := received["uptime_seconds"]; !ok {
		t.Errorf("heartbeat body missing uptime_seconds: %v", received)
	}
}

func TestReporterToleratesNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer server.Close()

	monitor := NewMonitor(slog.New(slog.DiscardHandler))
	authHeaders := func() map[string]string {
		return map[string]string{"x-agent-id": "ag_test"}
	}
	reporter := NewReporter(server.URL, authHeaders, monitor, slog.New(slog.DiscardHandler))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		reporter.sendHeartbeat(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("sendHeartbeat did not return on non-2xx response")
	}
}

func TestReporterStartStopsOnCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	monitor := NewMonitor(slog.New(slog.DiscardHandler))
	authHeaders := func() map[string]string {
		return map[string]string{"x-agent-id": "ag_test"}
	}
	reporter := NewReporter(server.URL, authHeaders, monitor, slog.New(slog.DiscardHandler))

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		reporter.Start(ctx, 10*time.Millisecond)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after context cancel")
	}
}
