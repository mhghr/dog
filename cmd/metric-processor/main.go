package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"monitoring-platform/internal/config"
	"monitoring-platform/internal/ingestion/processor"
	"monitoring-platform/internal/logging"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "metric-processor")

	if err := cfg.Require("NATS_URL", "VICTORIA_URL"); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	pool, err := processor.NewWorkerPool(processor.WorkerConfig{
		NATSURL:     cfg.NATSURL,
		VMURL:       cfg.VictoriaURL,
		Workers:     getEnvInt("WORKER_COUNT", 4),
		HTTPTimeout: 30 * time.Second,
	}, logger)
	if err != nil {
		logger.Error("failed to create worker pool", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("starting metric processor",
		"workers", getEnvInt("WORKER_COUNT", 4),
		"nats_url", cfg.NATSURL,
		"vm_url", cfg.VictoriaURL,
	)

	if err := pool.Start(ctx); err != nil {
		logger.Error("failed to start worker pool", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()
	logger.Info("shutting down metric processor")
	logger.Info("metric processor stopped")
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		var n int
		if _, err := fmt.Sscanf(v, "%d", &n); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}
