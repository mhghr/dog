package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monitoring-platform/internal/config"
	"monitoring-platform/internal/httpserver"
	"monitoring-platform/internal/logging"
)

func main() {
	cfg := config.Load()

	httpserver.RunHealthcheckCommand(cfg.HTTPAddress)

	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "agent-gateway")

	if err := cfg.Require("DATABASE_URL", "REDIS_ADDRESS"); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	gatewayMux := http.NewServeMux()

	gatewayMux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	gatewayMux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
			"role":   "agent-gateway",
		})
	})

	gatewayMux.HandleFunc("/api/v1/agents/hello", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":        "authenticated",
			"gateway_id":    "agent-gateway-01",
			"server_time":   time.Now().UTC().Format(time.RFC3339),
			"protocol_min":  "1",
			"protocol_max":  "1",
		})
	})

	healthServer := httpserver.New(cfg.HealthAddress, gatewayMux)

	go func() {
		if err := httpserver.Run(ctx, healthServer, logger); err != nil {
			logger.Error("gateway server failed", "error", err)
		}
	}()

	logger.Info("agent-gateway started", "health_address", cfg.HealthAddress)

	<-ctx.Done()
	logger.Info("agent-gateway stopped")
}
