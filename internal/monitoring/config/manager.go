package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Manager polls Core for the latest agent configuration and applies it.
type Manager struct {
	serverURL   string
	agentID     string
	authHeaders func() map[string]string
	logger      *slog.Logger
	client      *http.Client

	mu       sync.RWMutex
	current  *AgentConfig
	onChange func(old, new *AgentConfig)
}

// NewManager creates a config manager.
func NewManager(serverURL, agentID string, authHeaders func() map[string]string, logger *slog.Logger) *Manager {
	return &Manager{
		serverURL:   strings.TrimRight(serverURL, "/"),
		agentID:     agentID,
		authHeaders: authHeaders,
		logger:      logger,
		client:      &http.Client{Timeout: 30 * time.Second},
		current:     DefaultConfig(),
	}
}

// Start begins polling for config updates every pollInterval.
func (m *Manager) Start(ctx context.Context, pollInterval time.Duration) error {
	if err := m.fetchAndApply(ctx); err != nil {
		m.logger.Warn("initial config fetch failed, using defaults", "error", err)
	}

	go func() {
		ticker := time.NewTicker(pollInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := m.fetchAndApply(ctx); err != nil {
					m.logger.Debug("config poll failed", "error", err)
				}
			}
		}
	}()

	return nil
}

// OnChange registers a callback invoked when the config version advances.
func (m *Manager) OnChange(fn func(old, new *AgentConfig)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = fn
}

// Get returns the current config.
func (m *Manager) Get() *AgentConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Manager) fetchAndApply(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/v1/monitoring/agents/%s/config", m.serverURL, m.agentID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("create config request: %w", err)
	}
	for k, v := range m.authHeaders() {
		req.Header.Set(k, v)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch config: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("config fetch returned %d", resp.StatusCode)
	}

	var result struct {
		Version int         `json:"version"`
		Config  AgentConfig `json:"config"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode config: %w", err)
	}

	SanitizeConfig(&result.Config)

	old := m.Get()
	if result.Config.Version <= old.Version {
		return nil
	}

	m.mu.Lock()
	m.current = &result.Config
	m.mu.Unlock()

	m.logger.Info("config updated", "version", result.Config.Version)

	if m.onChange != nil {
		m.onChange(old, &result.Config)
	}

	return nil
}
