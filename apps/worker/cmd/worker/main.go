package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"monitoring-platform/packages/shared/agents/agent/spool"
	"monitoring-platform/packages/shared/config"
	"monitoring-platform/packages/shared/heartbeat"
	"monitoring-platform/packages/shared/httpserver"
	"monitoring-platform/packages/shared/ingestion/messagebus"
	"monitoring-platform/packages/shared/logging"
	"monitoring-platform/packages/shared/metrics"
	"monitoring-platform/packages/shared/probe"
	"monitoring-platform/packages/shared/queue"
	"monitoring-platform/packages/shared/security"
	"monitoring-platform/packages/shared/worker"
)

func main() {
	cfg := config.Load()

	httpserver.RunHealthcheckCommand(cfg.HealthAddress)

	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "worker")

	if err := cfg.Require("REDIS_ADDRESS", "API_BASE_URL", "WORKER_TOKEN"); err != nil {
		logger.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

	if err := probeQueue.EnsureGroup(ctx); err != nil {
		logger.Error("ensure consumer group failed", "error", err)
		os.Exit(1)
	}

	guard := security.NewGuard(cfg.SSRFAllowPrivate)
	if cfg.SSRFAllowPrivate {
		logger.Warn("SSRF protection allows private targets; do not use this in production")
	}

	registry := probe.DefaultRegistry(probe.Deps{
		Guard:          guard,
		Logger:         logger,
		PingPrivileged: cfg.PingPrivileged,
	})

	resultClient := worker.NewResultClient(cfg.APIBaseURL, cfg.WorkerToken)

	spoolDir := os.Getenv("AGENT_SPOOL_DIR")
	if spoolDir == "" {
		spoolDir = "./spool"
	}

	resultSpool, err := spool.New(spoolDir)
	if err != nil {
		logger.Error("failed to create result spool", "error", err)
		os.Exit(1)
	}
	defer resultSpool.Close()

	var sender spool.ResultSender = resultClient

	if mode := os.Getenv("TELEMETRY_PIPELINE_MODE"); mode == "nats" {
		natsURL := os.Getenv("NATS_URL")
		if natsURL == "" {
			natsURL = "nats://localhost:4222"
		}
		natsBus, err := messagebus.NewNATSBus(messagebus.NATSConfig{
			URL:       natsURL,
			Reconnect: true,
			MaxReconn: 10,
		}, logger)
		if err != nil {
			logger.Error("NATS connection failed", "error", err)
			os.Exit(1)
		}
		defer natsBus.Close()
		sender = worker.NewNATSResultSender(natsBus, logger)
	}

	batcher := spool.NewBatcher(resultSpool, sender, spool.DefaultBatcherConfig(), logger)
	go batcher.Run(ctx)

	promRegistry := metrics.NewRegistry()
	workerMetrics := metrics.NewWorkerMetrics(promRegistry)

	service := worker.New(
		probeQueue,
		registry,
		resultSpool,
		cfg.WorkerName,
		cfg.WorkerConcurrency,
		logger,
		workerMetrics,
	)

	go heartbeat.Run(ctx, redisClient, "worker", cfg.WorkerName, 3*time.Second, 10*time.Second)

	healthServer := httpserver.New(cfg.HealthAddress, workerHealthMux(redisClient, metrics.Handler(promRegistry)))
	go func() {
		if err := httpserver.Run(ctx, healthServer, logger); err != nil {
			logger.Error("health server failed", "error", err)
		}
	}()

	if err := service.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("worker failed", "error", err)
		os.Exit(1)
	}

	logger.Info("worker stopped")
}

func workerHealthMux(redisClient *redis.Client, promHandler http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if err := redisClient.Ping(ctx).Err(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable","dependencies":{"redis":"error"}}`))
			return
		}

		_, _ = w.Write([]byte(`{"status":"ok","dependencies":{"redis":"ok"}}`))
	})

	mux.Handle("/metrics", promHandler)

	return mux
}
