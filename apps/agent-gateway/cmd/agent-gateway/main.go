package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"monitoring-platform/packages/shared/agents/gateway"
	"monitoring-platform/packages/shared/httpserver"
	"monitoring-platform/packages/shared/logging"
)

func main() {
	httpserver.RunHealthcheckCommand(os.Getenv("GATEWAY_HEALTH_ADDRESS"))

	logger := logging.New(
		getEnv("LOG_LEVEL", "info"),
		getEnv("LOG_FORMAT", "json"),
		"agent-gateway",
	)

	gwCfg := gateway.GatewayConfig{
		ListenAddress: getEnv("GATEWAY_ADDRESS", ":8443"),
		HealthAddress: getEnv("GATEWAY_HEALTH_ADDRESS", ":8081"),
		TLSCertFile:   getEnv("GATEWAY_TLS_CERT", "/etc/agent-gateway/server.crt"),
		TLSKeyFile:    getEnv("GATEWAY_TLS_KEY", "/etc/agent-gateway/server.key"),
		CACertFile:    getEnv("GATEWAY_CA_CERT", "/etc/agent-gateway/ca.crt"),
		DatabaseURL:   os.Getenv("DATABASE_URL"),
	}

	if gwCfg.DatabaseURL == "" {
		logger.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	gw, err := gateway.New(gwCfg, logger)
	if err != nil {
		logger.Error("failed to create gateway", "error", err)
		os.Exit(1)
	}
	defer gw.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := gw.ListenAndServe(ctx); err != nil {
		logger.Error("gateway failed", "error", err)
		os.Exit(1)
	}

	logger.Info("agent-gateway stopped")
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
