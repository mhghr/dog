package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"monitoring-platform/packages/shared/config"
	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/heartbeat"
	"monitoring-platform/packages/shared/httpserver"
	"monitoring-platform/packages/shared/logging"
	"monitoring-platform/packages/shared/metrics"
	"monitoring-platform/packages/shared/infrastructure/postgres"
	"monitoring-platform/packages/shared/ingestion/messagebus"
	"monitoring-platform/packages/shared/queue"
	"monitoring-platform/packages/shared/scheduler"
)

func main() {
	cfg := config.Load()

	httpserver.RunHealthcheckCommand(cfg.HealthAddress)

	logger := logging.New(cfg.LogLevel, cfg.LogFormat, "scheduler")

	if err := cfg.Require("DATABASE_URL", "REDIS_ADDRESS"); err != nil {
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
		Stream:         cfg.QueueStream,
		Group:          cfg.QueueGroup,
		DeadLetter:     cfg.QueueDeadLetter,
		MaxLen:         cfg.QueueMaxLen,
		LocationPrefix: cfg.QueueLocationPrefix,
	}, logger)

	if err := probeQueue.EnsureGroup(ctx); err != nil {
		logger.Error("ensure consumer group failed", "error", err)
		os.Exit(1)
	}

	var jobQueue queue.JobQueue = probeQueue

	// Enterprise execution backbone: publish probe jobs over NATS JetStream
	// (PROBE_JOBS stream) instead of Redis Streams. Redis remains the cache/
	// session/lock store; only the job backbone switches.
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

		natsQueue, err := queue.NewNATSQueue(natsBus, "scheduler", logger)
		if err != nil {
			logger.Error("NATS queue setup failed", "error", err)
			os.Exit(1)
		}
		if err := natsQueue.EnsureGroup(ctx); err != nil {
			logger.Error("NATS queue ensure group failed", "error", err)
			os.Exit(1)
		}
		jobQueue = natsQueue
		logger.Info("probe job backbone switched to NATS JetStream")
	}

	enabledLocations, err := loadProbeLocations(ctx, cfg, pool)
	if err != nil {
		logger.Error("load probe locations failed", "error", err)
		os.Exit(1)
	}
	if len(enabledLocations) == 0 {
		logger.Warn("no active probe locations found; scheduler will publish to the default queue only")
	}
	logger.Info("probe locations loaded", "count", len(enabledLocations))

	registry := metrics.NewRegistry()
	schedulerMetrics := metrics.NewSchedulerMetrics(registry)

	monitorRepo := postgres.NewMonitorRepository(pool)

	service := scheduler.New(
		monitorRepo,
		jobQueue,
		enabledLocations,
		cfg.SchedulerBatchSize,
		cfg.SchedulerInterval,
		logger,
		schedulerMetrics,
	)

	go heartbeat.Run(ctx, redisClient, "scheduler", "scheduler", 3*time.Second, 10*time.Second)

	healthServer := httpserver.New(cfg.HealthAddress, healthMux(pool, redisClient, metrics.Handler(registry)))
	go func() {
		if err := httpserver.Run(ctx, healthServer, logger); err != nil {
			logger.Error("health server failed", "error", err)
		}
	}()

	if err := service.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Error("scheduler failed", "error", err)
		os.Exit(1)
	}

	logger.Info("scheduler stopped")
}

func loadProbeLocations(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool) ([]domain.ProbeLocation, error) {
	locations := postgres.NewLocationRepository(pool)
	all, err := locations.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list locations: %w", err)
	}

	enabled := make([]domain.ProbeLocation, 0, len(all))
	for _, loc := range all {
		if !loc.Enabled {
			continue
		}
		if cfg.ProbeLocationID != "" && loc.ID != cfg.ProbeLocationID {
			continue
		}
		enabled = append(enabled, loc)
	}

	return enabled, nil
}

func healthMux(pool *pgxpool.Pool, redisClient *redis.Client, promHandler http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		w.Header().Set("Content-Type", "application/json")

		if err := pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable","dependencies":{"postgres":"error"}}`))
			return
		}

		if err := redisClient.Ping(ctx).Err(); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"status":"unavailable","dependencies":{"redis":"error"}}`))
			return
		}

		_, _ = w.Write([]byte(`{"status":"ok","dependencies":{"postgres":"ok","redis":"ok"}}`))
	})

	mux.Handle("/metrics", promHandler)

	return mux
}
