package bootstrap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// ClientConfig holds bootstrap connection settings.
type ClientConfig struct {
	ServerURL      string
	BootstrapToken string
	Version        string
	Timeout        time.Duration
}

// BootstrapRequest is the payload sent to the core bootstrap endpoint.
type BootstrapRequest struct {
	BootstrapToken string   `json:"bootstrap_token"`
	Hostname       string   `json:"hostname"`
	OS             string   `json:"os"`
	Architecture   string   `json:"architecture"`
	AgentVersion   string   `json:"agent_version"`
	Capabilities   []string `json:"capabilities"`
}

// BootstrapResponse is returned by the core on successful bootstrap.
type BootstrapResponse struct {
	AgentID           string `json:"agent_id"`
	AgentSecret       string `json:"agent_secret"`
	ConfigURL         string `json:"config_url"`
	HeartbeatURL      string `json:"heartbeat_url"`
	ConfigPollSeconds int    `json:"config_poll_seconds"`
}

// Client performs the one-time bootstrap exchange with the core.
type Client struct {
	cfg    ClientConfig
	client *http.Client
}

// NewClient creates a bootstrap client.
func NewClient(cfg ClientConfig) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}
	return &Client{
		cfg: cfg,
		client: &http.Client{Timeout: cfg.Timeout},
	}
}

// Bootstrap exchanges the one-time token for agent credentials.
func (c *Client) Bootstrap(ctx context.Context, hostname, os, arch string, capabilities []string) (*BootstrapResponse, error) {
	req := BootstrapRequest{
		BootstrapToken: c.cfg.BootstrapToken,
		Hostname:       hostname,
		OS:             os,
		Architecture:   arch,
		AgentVersion:   c.cfg.Version,
		Capabilities:   capabilities,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.cfg.ServerURL+"/api/v1/monitoring/bootstrap", bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("bootstrap request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("bootstrap failed: status %d", resp.StatusCode)
	}

	var result BootstrapResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if result.AgentID == "" || result.AgentSecret == "" {
		return nil, fmt.Errorf("bootstrap response missing agent credentials")
	}

	return &result, nil
}
