package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"monitoring-platform/internal/logging"
	"monitoring-platform/internal/telemetry/ingest"
)

func main() {
	logger := logging.New(
		getEnv("LOG_LEVEL", "info"),
		getEnv("LOG_FORMAT", "json"),
		"telemetry-ingest",
	)

	cfg := ingest.Config{
		NATSURL:     requireEnv("NATS_URL"),
		VMURL:       requireEnv("VICTORIA_URL"),
		DatabaseURL: requireEnv("DATABASE_URL"),
		StreamName:      getEnv("TELEMETRY_STREAM", "TELEMETRY_EVENTS"),
		ConsumerName:    getEnv("TELEMETRY_CONSUMER", "telemetry-ingest"),
		QueueName:       getEnv("TELEMETRY_QUEUE", "telemetry-ingest"),
		DLQStreamName:   getEnv("TELEMETRY_DLQ_STREAM", "TELEMETRY_DLQ"),
		Workers:         getEnvInt("TELEMETRY_WORKERS", 4),
		BatchSize:       getEnvInt("TELEMETRY_BATCH_SIZE", 1000),
		FlushInterval:   getEnvDuration("TELEMETRY_FLUSH_INTERVAL", 5*time.Second),
		HTTPTimeout:     getEnvDuration("TELEMETRY_HTTP_TIMEOUT", 30*time.Second),
		ShutdownTimeout: getEnvDuration("TELEMETRY_SHUTDOWN_TIMEOUT", 30*time.Second),
		CircuitBreaker: ingest.CircuitBreakerConfig{
			FailureThreshold: getEnvInt("TELEMETRY_CB_FAILURE_THRESHOLD", 5),
			OpenDuration:     getEnvDuration("TELEMETRY_CB_OPEN_DURATION", 30*time.Second),
			HalfOpenMaxReqs:  getEnvInt("TELEMETRY_CB_HALF_OPEN_MAX", 1),
		},
		Stream: ingest.StreamConfig{
			MaxAge:   getEnvDuration("TELEMETRY_STREAM_MAX_AGE", 24*time.Hour),
			MaxBytes: getEnvInt64("TELEMETRY_STREAM_MAX_BYTES", 50_000_000_000),
			Replicas: getEnvInt("TELEMETRY_STREAM_REPLICAS", 3),
		},
	}

	svc, err := ingest.NewService(cfg, logger)
	if err != nil {
		logger.Error("failed to create service", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := svc.Start(ctx); err != nil {
		logger.Error("failed to start service", "error", err)
		os.Exit(1)
	}
	logger.Info("telemetry ingestion service started")

	<-ctx.Done()
	if err := svc.Shutdown(context.Background()); err != nil {
		logger.Error("shutdown error", "error", err)
		os.Exit(1)
	}
	logger.Info("telemetry ingestion service stopped")
}

func requireEnv(key string) string {
	val := os.Getenv(key)
	if val == "" {
		fmt.Fprintf(os.Stderr, "required env %s is not set\n", key)
		os.Exit(1)
	}
	return val
}

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func getEnvInt(key string, defaultVal int) int {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			return n
		}
	}
	return defaultVal
}

func getEnvInt64(key string, defaultVal int64) int64 {
	if val := os.Getenv(key); val != "" {
		if n, err := strconv.ParseInt(val, 10, 64); err == nil && n > 0 {
			return n
		}
	}
	return defaultVal
}

func getEnvDuration(key string, defaultVal time.Duration) time.Duration {
	if val := os.Getenv(key); val != "" {
		if d, err := time.ParseDuration(val); err == nil && d > 0 {
			return d
		}
	}
	return defaultVal
}
