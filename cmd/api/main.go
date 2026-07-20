package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"

	"monitoring-platform/internal/api"
	"monitoring-platform/internal/auth"
	"monitoring-platform/internal/config"
	"monitoring-platform/internal/events"
	"monitoring-platform/internal/httpserver"
	"monitoring-platform/internal/ingestion"
	"monitoring-platform/internal/logging"
	"monitoring-platform/internal/metrics"
	"monitoring-platform/internal/postgres"
	"monitoring-platform/internal/queue"
)

func main() {
	cfg := config.Load()

	httpserver.RunHealthcheckCommand(cfg.HTTPAddress)

	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "api")

	if err := cfg.Require("DATABASE_URL", "REDIS_ADDRESS", "WORKER_TOKEN"); err != nil {
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

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddress,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	defer redisClient.Close()

	probeQueue := queue.NewRedisQueue(redisClient, queue.StreamConfig{
		Stream:     cfg.QueueStream,
		Group:      cfg.QueueGroup,
		DeadLetter: cfg.QueueDeadLetter,
		MaxLen:     cfg.QueueMaxLen,
	}, logger)

	registry := metrics.NewRegistry()
	ingestionMetrics := metrics.NewIngestionMetrics(registry)

	victoria := metrics.NewVictoriaClient(cfg.VictoriaURL, logger)
	victoria.Start(ctx)

	bus := events.NewBus()

	monitorRepo := postgres.NewMonitorRepository(pool)
	resultRepo := postgres.NewResultRepository(pool)
	locationRepo := postgres.NewLocationRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(pool)
	otpRepo := postgres.NewOTPRepository(pool)

	if cfg.AuthJWTSecret == "dev-insecure-jwt-secret-change-me" && !cfg.IsDevelopment() {
		logger.Error("AUTH_JWT_SECRET must be set outside development")
		os.Exit(1)
	}

	tokenIssuer := auth.NewTokenIssuer(cfg.AuthJWTSecret, cfg.AccessTokenTTL)
	googleVerifier := auth.NewGoogleVerifier(cfg.GoogleClientID, cfg.GoogleClientSecret)
	if !googleVerifier.Enabled() {
		logger.Warn("google login disabled: GOOGLE_CLIENT_ID/GOOGLE_CLIENT_SECRET not set")
	}

	authService := auth.NewService(
		userRepo,
		refreshTokenRepo,
		otpRepo,
		tokenIssuer,
		googleVerifier,
		&auth.LogSender{Logger: logger},
		auth.Config{
			RefreshTTL:     cfg.RefreshTokenTTL,
			GoogleRedirect: cfg.GoogleRedirectURI,
			OTPDevMode:     cfg.OTPDevMode,
		},
		cfg.AuthJWTSecret,
		logger,
	)

	ingestionService := ingestion.NewService(
		resultRepo,
		monitorRepo,
		locationRepo,
		victoria,
		bus,
		logger,
		ingestionMetrics,
	)

	go watchQueueDepth(ctx, probeQueue, ingestionMetrics)

	router := api.NewRouter(api.Deps{
		Config:      cfg,
		Logger:      logger,
		Monitors:    monitorRepo,
		Results:     resultRepo,
		Locations:   locationRepo,
		StatusPages: postgres.NewStatusPageRepository(pool),
		Ingestion:   ingestionService,
		Auth:        authService,
		Issuer:      tokenIssuer,
		Bus:         bus,
		Queue:       probeQueue,
		Pool:        pool,
		Redis:       redisClient,
		Victoria:    victoria,
		Prom:        metrics.Handler(registry),
	})

	server := httpserver.New(cfg.HTTPAddress, router)
	if err := httpserver.Run(ctx, server, logger); err != nil {
		logger.Error("http server failed", "error", err)
		os.Exit(1)
	}

	logger.Info("api stopped")
}
