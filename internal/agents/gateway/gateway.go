package gateway

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/internal/agents"
)

type GatewayConfig struct {
	ListenAddress string
	HealthAddress string
	TLSCertFile   string
	TLSKeyFile    string
	CACertFile    string
	DatabaseURL   string
}

type GatewayServer struct {
	db        *pgxpool.Pool
	agentRepo *agents.Repository
	config    GatewayConfig
	logger    *slog.Logger
	server    *http.Server
}

func New(cfg GatewayConfig, logger *slog.Logger) (*GatewayServer, error) {
	pool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	return &GatewayServer{
		db:        pool,
		agentRepo: agents.NewRepository(pool),
		config:    cfg,
		logger:    logger,
	}, nil
}

func (g *GatewayServer) ListenAndServe(ctx context.Context) error {
	tlsConfig, err := g.buildTLSConfig()
	if err != nil {
		return fmt.Errorf("build TLS config: %w", err)
	}

	gatewayMux := g.buildGatewayMux()
	healthMux := g.buildHealthMux()

	g.server = &http.Server{
		Addr:              g.config.ListenAddress,
		Handler:           gatewayMux,
		TLSConfig:         tlsConfig,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	healthServer := &http.Server{
		Addr:              g.config.HealthAddress,
		Handler:           healthMux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errChan := make(chan error, 2)

	go func() {
		g.logger.Info("gateway listening", "address", g.config.ListenAddress)
		if err := g.server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	go func() {
		g.logger.Info("health server listening", "address", g.config.HealthAddress)
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	g.logger.Info("shutting down gateway")
	g.server.Shutdown(shutdownCtx)
	healthServer.Shutdown(shutdownCtx)

	return nil
}

func (g *GatewayServer) Close() {
	if g.db != nil {
		g.db.Close()
	}
}

func (g *GatewayServer) buildTLSConfig() (*tls.Config, error) {
	caCert, err := os.ReadFile(g.config.CACertFile)
	if err != nil {
		return nil, fmt.Errorf("read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	return &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caCertPool,
		MinVersion: tls.VersionTLS12,
	}, nil
}

func (g *GatewayServer) buildGatewayMux() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/agent/v1/hello", g.handleAgentHello)
	mux.HandleFunc("/agent/v1/heartbeat", g.handleAgentHeartbeat)
	mux.HandleFunc("/agent/v1/connect", g.handleAgentConnect)

	return mux
}

func (g *GatewayServer) buildHealthMux() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if err := g.db.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			json.NewEncoder(w).Encode(map[string]string{
				"status": "unavailable",
				"error":  fmt.Sprintf("database: %v", err),
			})
			return
		}

		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"role":   "agent-gateway",
		})
	})

	return mux
}

func (g *GatewayServer) handleAgentHello(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	agentID, err := g.verifyAgent(r)
	if err != nil {
		g.logger.Warn("agent verification failed", "error", err)
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"status": "unauthenticated", "error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":        "authenticated",
		"agent_id":      agentID.String(),
		"gateway_id":    "agent-gateway-01",
		"server_time":   time.Now().UTC().Format(time.RFC3339),
		"protocol_min":  "1",
		"protocol_max":  "1",
	})
}

func (g *GatewayServer) handleAgentHeartbeat(w http.ResponseWriter, r *http.Request) {
	agentID, err := g.verifyAgent(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	publicIP, _, _ := net.SplitHostPort(r.RemoteAddr)

	if err := g.agentRepo.AgentHeartbeat(ctx, agentID, publicIP); err != nil {
		g.logger.Error("agent heartbeat update failed", "agent_id", agentID, "error", err)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "ok",
		"agent_id": agentID.String(),
	})
}

func (g *GatewayServer) handleAgentConnect(w http.ResponseWriter, r *http.Request) {
	agentID, err := g.verifyAgent(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	agent, err := g.agentRepo.GetAgent(ctx, agentID)
	if err != nil {
		g.logger.Error("agent lookup failed", "agent_id", agentID, "error", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !agents.IsOperational(agent.Status) && agent.Status != agents.AgentActive {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{
			"status":       "forbidden",
			"agent_status": string(agent.Status),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "connected",
		"agent_id":        agent.ID.String(),
		"location_id":     agent.LocationID.String(),
		"capabilities":    agent.Capabilities,
		"max_concurrency": agent.MaxConcurrency,
	})
}

func (g *GatewayServer) verifyAgent(r *http.Request) (uuid.UUID, error) {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return uuid.Nil, fmt.Errorf("no client certificate")
	}

	clientCert := r.TLS.PeerCertificates[0]
	cn := clientCert.Subject.CommonName

	if !strings.HasPrefix(cn, "probe-agent-") {
		return uuid.Nil, fmt.Errorf("invalid certificate CN: %s", cn)
	}

	agentIDStr := strings.TrimPrefix(cn, "probe-agent-")
	agentID, err := uuid.Parse(agentIDStr)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid agent ID in certificate: %w", err)
	}

	return agentID, nil
}
