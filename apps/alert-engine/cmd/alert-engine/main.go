package main

import (
	"context"
	"os/signal"
	"syscall"

	"monitoring-platform/packages/shared/config"
	"monitoring-platform/packages/shared/logging"
)

func main() {
	cfg := config.Load()

	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "alert-engine")

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger.Info("alert-engine starting",
		"nats_url", cfg.NATSURL,
		"database_url_set", cfg.DatabaseURL != "",
	)

	<-ctx.Done()
	logger.Info("alert-engine stopped")
}
