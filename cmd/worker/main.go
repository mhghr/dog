package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"monitoring-platform/internal/config"
	"monitoring-platform/internal/heartbeat"
	"monitoring-platform/internal/httpserver"
	"monitoring-platform/internal/logging"
	"monitoring-platform/internal/metrics"
	"monitoring-platform/internal/probe"
	"monitoring-platform/internal/queue"
	"monitoring-platform/internal/security"
	"monitoring-platform/internal/worker"
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

	promRegistry := metrics.NewRegistry()
	workerMetrics := metrics.NewWorkerMetrics(promRegistry)

	service := worker.New(
		probeQueue,
		registry,
		resultClient,
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
