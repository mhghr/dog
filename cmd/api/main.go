package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"

	"monitoring-platform/internal/agents"
	"monitoring-platform/internal/alerting"
	"monitoring-platform/internal/api"
	"monitoring-platform/internal/auth"
	"monitoring-platform/internal/config"
	"monitoring-platform/internal/domain"
	"monitoring-platform/internal/events"
	"monitoring-platform/internal/health"
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

	if err := cfg.Require("DATABASE_URL", "REDIS_ADDRESS", "WORKER_TOKEN", "AGENT_SECRET_ENCRYPTION_KEY"); err != nil {
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

	var victoria *metrics.VictoriaClient
	if cfg.TelemetryPipeline.Mode != "nats" {
		victoria = metrics.NewVictoriaClient(cfg.VictoriaURL, logger)
		victoria.Start(ctx)
	}

	bus := events.NewBus()

	monitorRepo := postgres.NewMonitorRepository(pool)
	resultRepo := postgres.NewResultRepository(pool)
	locationRepo := postgres.NewLocationRepository(pool)
	userRepo := postgres.NewUserRepository(pool)
	refreshTokenRepo := postgres.NewRefreshTokenRepository(pool)
	otpRepo := postgres.NewOTPRepository(pool)
	orgRepo := postgres.NewOrganizationRepository(pool)

	alertRepo := postgres.NewAlertRepository(pool)
	channelRepo := postgres.NewChannelRepository(pool)
	alertEngine := alerting.NewEngine(alertRepo, logger)
	alertNotifier := alerting.NewNotifier(channelRepo, logger)

	healthRepo := postgres.NewHealthRepository(pool)
	healthEngine := health.NewEngine(healthRepo, logger)
	healthNotifier := health.NewNotificationEngine(healthRepo, logger)

	agentRepo := agents.NewRepository(pool)
	resourceRepo := postgres.NewResourceRepository(pool)

	var ca *agents.CertAuthority
	caCertPEM := os.Getenv("AGENT_CA_CERT")
	caKeyPEM := os.Getenv("AGENT_CA_KEY")
	if caCertPath := os.Getenv("AGENT_CA_CERT_FILE"); caCertPath != "" {
		data, rdErr := os.ReadFile(caCertPath)
		if rdErr != nil {
			logger.Error("failed to read AGENT_CA_CERT_FILE", "path", caCertPath, "error", rdErr)
			os.Exit(1)
		}
		caCertPEM = string(data)
	}
	if caKeyPath := os.Getenv("AGENT_CA_KEY_FILE"); caKeyPath != "" {
		data, rdErr := os.ReadFile(caKeyPath)
		if rdErr != nil {
			logger.Error("failed to read AGENT_CA_KEY_FILE", "path", caKeyPath, "error", rdErr)
			os.Exit(1)
		}
		caKeyPEM = string(data)
	}
	if caCertPEM != "" && caKeyPEM != "" {
		ca, err = agents.NewCertAuthority([]byte(caCertPEM), []byte(caKeyPEM))
		if err != nil {
			logger.Error("failed to load agent CA", "error", err)
			os.Exit(1)
		}
		logger.Info("agent CA loaded", "fingerprint", ca.Fingerprint())
	} else {
		var caCert, caKey []byte
		ca, caCert, caKey, err = agents.NewSelfSignedCA()
		if err != nil {
			logger.Error("failed to create self-signed CA", "error", err)
			os.Exit(1)
		}
		logger.Warn("no CA configured, generated self-signed CA", "fingerprint", ca.Fingerprint())
		_ = caCert
		_ = caKey
	}

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
		orgRepo,
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
		healthEngine,
		healthNotifier,
	)

	go watchQueueDepth(ctx, probeQueue, ingestionMetrics)
	go watchOfflineAgents(ctx, agentRepo, logger)
	if cfg.TelemetryPipeline.Mode != "nats" {
		go consumeGatewayResults(ctx, redisClient, ingestionService, logger)
	}

	router := api.NewRouter(api.Deps{
		Config:           cfg,
		Logger:           logger,
		Monitors:         monitorRepo,
		Results:          resultRepo,
		Locations:        locationRepo,
		StatusPages:      postgres.NewStatusPageRepository(pool),
		Orgs:             postgres.NewOrganizationRepository(pool),
		Projects:         postgres.NewProjectRepository(pool),
		AlertRepo:        alertRepo,
		ChannelRepo:      channelRepo,
		AlertEngine:      alertEngine,
		Notifier:         alertNotifier,
		HealthRepo:       healthRepo,
		HealthNotifier:   healthNotifier,
		Ingestion:        ingestionService,
		Auth:             authService,
		Issuer:           tokenIssuer,
		Bus:              *bus,
		Queue:            probeQueue,
		Pool:             pool,
		Redis:            redisClient,
		Victoria:         victoria,
		Prom:             metrics.Handler(registry),
		AgentRepo:        agentRepo,
		ResourceRepo:     resourceRepo,
		CA:               ca,
		MonitoringAgents: postgres.NewMonitoringAgentRepository(pool),
		BootstrapTokens:  postgres.NewBootstrapTokenRepository(pool),
		AgentConfigs:     postgres.NewAgentConfigRepository(pool),
	})

	server := httpserver.New(cfg.HTTPAddress, router)
	if err := httpserver.Run(ctx, server, logger); err != nil {
		logger.Error("http server failed", "error", err)
		os.Exit(1)
	}

	logger.Info("api stopped")
}

// consumeGatewayResults subscribes to the Redis probe_results channel and
// ingests results published by the agent gateway.
func consumeGatewayResults(ctx context.Context, rdb *redis.Client, svc *ingestion.Service, logger *slog.Logger) {
	pubsub := rdb.Subscribe(ctx, "probe_results")
	defer pubsub.Close()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			var result domain.ProbeResult
			if err := json.Unmarshal([]byte(msg.Payload), &result); err != nil {
				logger.Warn("gateway result: unmarshal failed", "error", err)
				continue
			}
			if _, err := svc.Ingest(ctx, &result); err != nil {
				logger.Error("gateway result: ingest failed", "error", err, "monitor_id", result.MonitorID)
			}
		}
	}
}
