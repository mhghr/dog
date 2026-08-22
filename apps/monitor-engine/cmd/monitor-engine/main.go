package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"monitoring-platform/packages/shared/config"
	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/engines"
	"monitoring-platform/packages/shared/health"
	"monitoring-platform/packages/shared/httpserver"
	"monitoring-platform/packages/shared/ingestion/messagebus"
	"monitoring-platform/packages/shared/infrastructure/postgres"
	"monitoring-platform/packages/shared/logging"
)

func main() {
	cfg := config.Load()

	httpserver.RunHealthcheckCommand(cfg.HealthAddress)

	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "monitor-engine")

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

	healthRepo := postgres.NewHealthRepository(pool)
	healthEngine := health.NewEngine(healthRepo, logger)
	healthNotifier := health.NewNotificationEngine(healthRepo, logger)

	if err := engines.StartResultConsumer(ctx, bus, engines.ConsumerOptions{
		Subject: engines.HealthSubject,
		Queue:   "monitor-engines",
		Durable: "monitor-engine",
		Stream:  "ENGINE_EVENTS",
		Workers: 4,
	}, func(ctx context.Context, result *domain.ProbeResult) error {
		outcomes, err := healthEngine.EvaluateResult(ctx, result)
		if err != nil {
			return err
		}
		for i := range outcomes {
			healthNotifier.Evaluate(ctx, outcomes[i])
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

	logger.Info("monitor-engine starting",
		"nats_url", cfg.NATSURL,
		"subject", engines.HealthSubject,
		"workers", 4,
	)

	<-ctx.Done()
	logger.Info("monitor-engine stopped")
}
