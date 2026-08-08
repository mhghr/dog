package agent

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// GRPCAgent maintains the connection between a probe agent and the Dog
// control plane gateway. It periodically reports identity/liveness over
// the configured gateway address.
type GRPCAgent struct {
	cfg     AgentConfig
	agentID string
	logger  *slog.Logger
	client  *http.Client
}

// NewGRPCAgent creates a probe agent connection. The transport is kept
// protocol-agnostic; when the gRPC gateway contract is generated the
// underlying dial will switch to grpc.Dial.
func NewGRPCAgent(cfg AgentConfig, agentID string, logger *slog.Logger) (*GRPCAgent, error) {
	if agentID == "" {
		return nil, fmt.Errorf("agent id is required")
	}
	return &GRPCAgent{
		cfg:     cfg,
		agentID: agentID,
		logger:  logger,
		client:  &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (g *GRPCAgent) Close() {
	g.client.CloseIdleConnections()
}

// Run blocks until the context is cancelled, reporting liveness to the
// gateway at a fixed interval.
func (g *GRPCAgent) Run(ctx context.Context) error {
	g.logger.Info("probe agent connected", "agent_id", g.agentID, "gateway", g.cfg.Gateway)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			g.heartbeat(ctx)
		}
	}
}

func (g *GRPCAgent) heartbeat(ctx context.Context) {
	// Best-effort liveness report. The gateway endpoint is resolved from
	// the gateway address; failures are logged but do not stop the agent.
	host := g.cfg.Gateway
	if _, _, err := net.SplitHostPort(host); err != nil {
		host = net.JoinHostPort(host, "8443")
	}

	url := fmt.Sprintf("https://%s/healthz", host)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return
	}
	req.Header.Set("X-Agent-ID", g.agentID)

	resp, err := g.client.Do(req)
	if err != nil {
		g.logger.Debug("gateway heartbeat failed", "error", err)
		return
	}
	resp.Body.Close()
}
