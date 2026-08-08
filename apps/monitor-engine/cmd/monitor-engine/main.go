package main

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"

	"monitoring-platform/packages/shared/config"
	"monitoring-platform/packages/shared/logging"
)

func main() {
	cfg := config.Load()

	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "monitor-engine")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("monitor-engine starting",
		"nats_url", cfg.NATSURL,
		"database_url_set", cfg.DatabaseURL != "",
	)

	<-ctx.Done()
	logger.Info("monitor-engine stopped")
	_ = slog.Default()
}
