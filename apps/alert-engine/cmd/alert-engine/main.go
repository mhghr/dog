package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"monitoring-platform/packages/shared/alerting"
	"monitoring-platform/packages/shared/config"
	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/engines"
	"monitoring-platform/packages/shared/httpserver"
	"monitoring-platform/packages/shared/ingestion/messagebus"
	"monitoring-platform/packages/shared/infrastructure/postgres"
	"monitoring-platform/packages/shared/logging"
)

func main() {
	cfg := config.Load()

	httpserver.RunHealthcheckCommand(cfg.HealthAddress)

	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "alert-engine")

	if err := cfg.Require("DATABASE_URL", "NATS_URL"); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("connect to postgres failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	bus, err := messagebus.NewNATSBus(messagebus.NATSConfig{
		URL:       cfg.NATSURL,
		Reconnect: true,
		MaxReconn: 10,
	}, logger)
	if err != nil {
		logger.Error("connect to NATS failed", "error", err)
		os.Exit(1)
	}
	defer bus.Close()

	alertRepo := postgres.NewAlertRepository(pool)
	channelRepo := postgres.NewChannelRepository(pool)
	alertEngine := alerting.NewEngine(alertRepo, logger)
	alertNotifier := alerting.NewNotifier(channelRepo, logger)

	if err := engines.StartResultConsumer(ctx, bus, engines.ConsumerOptions{
		Subject: engines.AlertSubject,
		Queue:   "alert-engines",
		Durable: "alert-engine",
		Stream:  "ENGINE_EVENTS",
		Workers: 4,
	}, func(ctx context.Context, result *domain.ProbeResult) error {
		events := alertEngine.Evaluate(ctx, *result)
		for _, evt := range events {
			alertNotifier.Dispatch(ctx, evt, evt.ChannelIDs)
		}
		return nil
	}, logger); err != nil {
		logger.Error("engine consumer setup failed", "error", err)
		os.Exit(1)
	}

	healthServer := httpserver.New(cfg.HealthAddress, engines.HealthMux(pool))
	go func() {
		if err := httpserver.Run(ctx, healthServer, logger); err != nil {
			logger.Error("health server failed", "error", err)
		}
	}()

	logger.Info("alert-engine starting",
		"nats_url", cfg.NATSURL,
		"subject", engines.AlertSubject,
		"workers", 4,
	)

	<-ctx.Done()
	logger.Info("alert-engine stopped")
}
