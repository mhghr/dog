package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"monitoring-platform/internal/logging"
	"monitoring-platform/internal/monitoring/bootstrap"
	"monitoring-platform/internal/monitoring/collector"
	monitoringconfig "monitoring-platform/internal/monitoring/config"
	"monitoring-platform/internal/monitoring/credential"
	"monitoring-platform/internal/monitoring/health"
	"monitoring-platform/internal/monitoring/signer"
	"monitoring-platform/internal/monitoring/storage"
	"monitoring-platform/internal/monitoring/transport"
	"monitoring-platform/internal/monitoring/updater"
)

var version = "dev"

type agentConfig struct {
	ServerURL      string
	BootstrapToken string
	StateDir       string
	OTelBinary     string
	OTelEndpoint   string
	ProxyAddress   string
	CACertFile     string
	LogLevel       string
	LogFormat      string
	UpdateChannel  string
	UpdateBaseURL  string
}

func loadConfig() agentConfig {
	return agentConfig{
		ServerURL:      getEnv("MONITORING_SERVER_URL", "https://monitor.example.com"),
		BootstrapToken: getEnv("MONITORING_BOOTSTRAP_TOKEN", ""),
		StateDir:       getEnv("MONITORING_STATE_DIR", "/var/lib/monitoring-agent"),
		OTelBinary:     getEnv("MONITORING_OTEL_BINARY", "otelcol"),
		OTelEndpoint:   getEnv("MONITORING_OTEL_ENDPOINT", "https://monitor.example.com:4318"),
		ProxyAddress:   getEnv("MONITORING_PROXY_ADDRESS", "127.0.0.1:4319"),
		CACertFile:     getEnv("MONITORING_CA_CERT", ""),
		LogLevel:       getEnv("MONITORING_LOG_LEVEL", "info"),
		LogFormat:      getEnv("MONITORING_LOG_FORMAT", "json"),
		UpdateChannel:  getEnv("MONITORING_UPDATE_CHANNEL", "stable"),
		UpdateBaseURL:  getEnv("MONITORING_UPDATE_BASE_URL", "https://agent.example.com"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	cfg := loadConfig()
	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "monitoring-agent")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	credMgr, err := credential.NewManager(cfg.StateDir)
	if err != nil {
		logger.Error("failed to initialize credential manager", "error", err)
		os.Exit(1)
	}

	if !credMgr.HasCredentials() {
		if cfg.BootstrapToken == "" {
			logger.Error("no credentials found and no bootstrap token provided")
			os.Exit(1)
		}
		if err := bootstrapAgent(ctx, cfg, credMgr, logger); err != nil {
			logger.Error("bootstrap failed", "error", err)
			os.Exit(1)
		}
	}

	creds, err := credMgr.LoadCredentials()
	if err != nil {
		logger.Error("failed to load credentials", "error", err)
		os.Exit(1)
	}
	logger.Info("authenticated", "agent_id", creds.AgentID)

	// Start the local OTLP signing proxy. The embedded otelcol exporter points
	// at this local proxy, which signs each export with the agent credential
	// and forwards it over TLS to the otel-ingest gateway.
	proxyServer, signerProxy, err := startSigningProxy(ctx, cfg, credMgr, logger)
	if proxyServer != nil {
		defer proxyServer.Stop()
		defer signerProxy.Close()
	}

	// Resolve the otelcol binary path.
	binaryPath := cfg.OTelBinary
	if found, err := collector.FindBinary(); err == nil {
		binaryPath = found
	}

	configMgr := monitoringconfig.NewManager(cfg.ServerURL, creds.AgentID, credMgr.AuthHeader, logger)
	supervisor := collector.NewSupervisor(cfg.StateDir, binaryPath, logger)

	// Write initial collector config and start it.
	otelConfig := monitoringconfig.GenerateOTelConfig(configMgr.Get(), cfg.ProxyAddress, creds.AgentID)
	if err := supervisor.Reload(otelConfig); err != nil {
		logger.Warn("failed to write initial collector config", "error", err)
	}
	if err := supervisor.Start(ctx); err != nil {
		logger.Warn("failed to start collector, will retry", "error", err)
	}
	go supervisor.WatchRestarts(ctx)

	// Reload collector when the remote config advances.
	configMgr.OnChange(func(old, new *monitoringconfig.AgentConfig) {
		logger.Info("config changed, reloading collector",
			"old_version", old.Version,
			"new_version", new.Version,
		)
		yaml := monitoringconfig.GenerateOTelConfig(new, cfg.ProxyAddress, creds.AgentID)
		if err := supervisor.Reload(yaml); err != nil {
			logger.Error("failed to reload collector", "error", err)
		}
	})

	// Background services.
	configMgr.Start(ctx, 60*time.Second)

	healthMonitor := health.NewMonitor(logger)
	healthReporter := health.NewReporter(creds.HeartbeatURL, credMgr.AuthHeader, healthMonitor, logger)
	go healthReporter.Start(ctx, 30*time.Second)

	agentUpdater := updater.NewUpdater(version, cfg.UpdateChannel, cfg.UpdateBaseURL, binaryPath, logger)
	go agentUpdater.Start(ctx, 24*time.Hour)

	// Local spool (used for buffering; retained for future flush logic).
	spool, err := storage.NewQueue(cfg.StateDir+"/spool", 100)
	if err != nil {
		logger.Warn("failed to initialize spool", "error", err)
	} else {
		defer spool.Close()
	}

	logger.Info("monitoring-agent started",
		"agent_id", creds.AgentID,
		"version", version,
		"state_dir", cfg.StateDir,
		"collector_binary", binaryPath,
	)

	<-ctx.Done()
	logger.Info("shutting down")
	supervisor.Stop()
	logger.Info("monitoring-agent stopped")
}

// startSigningProxy dials the remote otel-ingest gateway and starts the local
// signing proxy listener. It returns nil values if the proxy could not be
// started (the agent keeps running, but metric exports will not be
// authenticated). The returned proxy and server share a lifetime: the caller
// must stop the server and close the proxy.
func startSigningProxy(ctx context.Context, cfg agentConfig, credMgr *credential.Manager, logger *slog.Logger) (*grpc.Server, *signer.Proxy, error) {
	conn, err := transport.NewGRPCConnection(ctx, transport.GRPCConfig{Endpoint: cfg.OTelEndpoint, CACertFile: cfg.CACertFile})
	if err != nil {
		logger.Warn("failed to dial otel-ingest for signing proxy, metrics will not be authenticated", "error", err)
		return nil, nil, nil
	}

	proxy, err := signer.New(conn, credMgr, logger)
	if err != nil {
		logger.Warn("failed to create signing proxy, metrics will not be authenticated", "error", err)
		conn.Close()
		return nil, nil, nil
	}

	server, _, err := signer.Serve(ctx, cfg.ProxyAddress, proxy, logger)
	if err != nil {
		logger.Warn("failed to start signing proxy, metrics will not be authenticated", "error", err)
		proxy.Close()
		return nil, nil, nil
	}
	logger.Info("signing proxy listening", "address", cfg.ProxyAddress)
	return server, proxy, nil
}

// bootstrapAgent performs the one-time bootstrap exchange and stores credentials.
func bootstrapAgent(ctx context.Context, cfg agentConfig, credMgr *credential.Manager, logger *slog.Logger) error {
	logger.Info("performing initial bootstrap")

	client := bootstrap.NewClient(bootstrap.ClientConfig{
		ServerURL:      cfg.ServerURL,
		BootstrapToken: cfg.BootstrapToken,
		Version:        version,
	})

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}

	result, err := client.Bootstrap(ctx, hostname, runtime.GOOS, runtime.GOARCH, []string{
		"host_metrics:cpu", "host_metrics:memory", "host_metrics:filesystem",
		"host_metrics:disk", "host_metrics:network", "host_metrics:load",
	})
	if err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}

	// Construct full URLs from the relative paths returned by Core.
	serverBase := strings.TrimRight(cfg.ServerURL, "/")
	fullConfigURL := serverBase + result.ConfigURL
	fullHeartbeatURL := serverBase + result.HeartbeatURL

	if err := credMgr.SaveCredentials(
		result.AgentID,
		result.AgentSecret,
		cfg.ServerURL,
		fullConfigURL,
		fullHeartbeatURL,
	); err != nil {
		return fmt.Errorf("save credentials: %w", err)
	}

	logger.Info("bootstrap complete",
		"agent_id", result.AgentID,
		"config_poll_seconds", result.ConfigPollSeconds,
	)
	return nil
}
