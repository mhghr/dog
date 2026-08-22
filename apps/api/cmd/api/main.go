package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"

	"monitoring-platform/packages/shared/agents"
	"monitoring-platform/packages/shared/alerting"
	"monitoring-platform/packages/shared/interfaces/http"
	"monitoring-platform/packages/shared/auth"
	"monitoring-platform/packages/shared/config"
	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/events"
	"monitoring-platform/packages/shared/health"
	"monitoring-platform/packages/shared/httpserver"
	"monitoring-platform/packages/shared/ingestion"
	"monitoring-platform/packages/shared/logging"
	"monitoring-platform/packages/shared/metrics"
	"monitoring-platform/packages/shared/infrastructure/postgres"
	"monitoring-platform/packages/shared/queue"
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

	// Distributed SSE: when LIVE_EVENTS_NATS=1, live events are fanned out to a
	// shared NATS bus so every API replica serves the same stream to its own
	// browsers. The local bus is always the delivery target; the publisher
	// chooses local-only vs local+NATS. The relay on each replica forwards
	// events from other instances into this instance's local bus.
	var livePublisher events.Publisher = bus
	if os.Getenv("LIVE_EVENTS_NATS") == "1" && cfg.NATSURL != "" {
		nc, err := nats.Connect(cfg.NATSURL,
			nats.Name("dog-api-live-events"),
			nats.Timeout(10*time.Second),
			nats.ReconnectWait(2*time.Second),
			nats.MaxReconnects(-1),
		)
		if err != nil {
			logger.Warn("distributed live events disabled: NATS connect failed", "error", err)
		} else {
			origin := events.CleanNATSURL(cfg.NATSURL) + "-" + hostnameSuffix()
			livePublisher = events.NewDistributedPublisher(bus, nc, origin, logger)

			relay := events.NewNATSRelay(bus, nc, origin, logger)
			go func() {
				if err := relay.Start(ctx); err != nil {
					logger.Warn("live event relay stopped", "error", err)
				}
			}()
			logger.Info("distributed live events enabled", "origin", origin, "nats_url", cfg.NATSURL)
		}
	}

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
	monitorTypeParams := postgres.NewMonitorTypeParameterRepository(pool)
	snmpRepo := postgres.NewSNMPRepository(pool)
	metricSeriesRepo := postgres.NewMetricSeriesRepository(pool)

	// On-demand SNMP operations (test connection / discovery) run on the SNMP
	// collector — via the worker in NATS mode, inline otherwise.
	snmpRunner, snmpRunnerIsNATS := api.SelectSNMPTaskRunner(cfg, snmpRepo, logger)
	if snmpRunnerIsNATS {
		go func() {
			if err := snmpRunner.Start(ctx); err != nil {
				logger.Error("snmp task result consumer failed", "error", err)
			}
		}()
	}

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
		livePublisher,
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
		Results:          resultRepo,
		MetricQuery:      postgres.NewMetricQueryService(resultRepo, metricSeriesRepo),
		Locations:        locationRepo,
		StatusPages:      postgres.NewStatusPageRepository(pool),
		Orgs:             postgres.NewOrganizationRepository(pool),
		AlertRepo:        alertRepo,
		ChannelRepo:      channelRepo,
		AlertEngine:      alertEngine,
		Notifier:         alertNotifier,
		HealthRepo:       healthRepo,
		HealthNotifier:   healthNotifier,
		Ingestion:        ingestionService,
		Auth:             authService,
		Issuer:           tokenIssuer,
		Bus:              bus,
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
		MonitorTypeParams: monitorTypeParams,
		MonitorRepo:      monitorRepo,
		SNMP:             snmpRepo,
		SNMPRunner:       snmpRunner,
	})

	// SNMP trap receiver (UDP 162). Enabled via SNMP_TRAP_ENABLED=true and
	// SNMP_TRAP_ADDRESS (default ":162"). Traps are normalized and persisted
	// as events; polling remains the primary metric source.
	if os.Getenv("SNMP_TRAP_ENABLED") == "true" {
		trapAddr := os.Getenv("SNMP_TRAP_ADDRESS")
		if trapAddr == "" {
			trapAddr = ":162"
		}
		trapReceiver := api.NewSNMPTrapReceiver(api.Deps{
			Config:     cfg,
			Logger:     logger,
			MonitorRepo: monitorRepo,
			SNMP:       snmpRepo,
		}, trapAddr)
		if err := trapReceiver.Start(); err != nil {
			logger.Error("failed to start snmp trap receiver", "error", err)
			os.Exit(1)
		}
		logger.Info("snmp trap receiver started", "address", trapAddr)
		defer trapReceiver.Close()
	}

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

// hostnameSuffix returns a short, stable per-process suffix used to build the
// distributed live-event origin. It keeps replicas distinct even when several
// run on the same host.
func hostnameSuffix() string {
	host, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	host = strings.TrimSuffix(host, ".local")
	if len(host) > 24 {
		host = host[:24]
	}
	return host
}
