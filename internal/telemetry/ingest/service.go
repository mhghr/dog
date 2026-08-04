package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"monitoring-platform/internal/domain"
	"monitoring-platform/internal/infrastructure/postgres"
	"monitoring-platform/internal/ingestion/messagebus"
)

type Service struct {
	consumer    *Consumer
	batchWriter *BatchWriter
	cb          *CircuitBreaker
	metrics     *TelemetryMetrics
	logger      *slog.Logger
	cfg         Config
	pool        *pgxpool.Pool
	bus         messagebus.MessageBus
}

func NewService(cfg Config, logger *slog.Logger) (*Service, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}

	bus, err := messagebus.NewNATSBus(messagebus.NATSConfig{
		URL:       cfg.NATSURL,
		Reconnect: true,
		MaxReconn: 10,
	}, logger)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	cb := NewCircuitBreaker(cfg.CircuitBreaker)
	batchWriter := NewBatchWriter(cfg.VMURL, BatchWriterConfig{
		BatchSize:     cfg.BatchSize,
		FlushInterval: cfg.FlushInterval,
		HTTPTimeout:   cfg.HTTPTimeout,
	})

	metrics := NewTelemetryMetrics(nil)

	resultRepo := postgres.NewResultRepository(pool)
	monitorV2Repo := postgres.NewMonitorV2Repository(pool)
	ingestionProc := &ingestionAdapter{
		resultRepo:    resultRepo,
		monitorV2Repo: monitorV2Repo,
	}

	dedupStore := postgres.NewDedupStore(pool)
	dedup := NewDeduplicator(dedupStore, 10000, 300)

	consumer := NewConsumer(bus, ingestionProc, batchWriter, dedup, cb, metrics, logger, cfg)

	return &Service{
		consumer:    consumer,
		batchWriter: batchWriter,
		cb:          cb,
		metrics:     metrics,
		logger:      logger,
		cfg:         cfg,
		pool:        pool,
		bus:         bus,
	}, nil
}

func (s *Service) Start(ctx context.Context) error {
	s.logger.Info("starting telemetry ingestion service",
		"workers", s.cfg.Workers,
		"batch_size", s.cfg.BatchSize,
		"flush_interval", s.cfg.FlushInterval,
	)
	return s.consumer.Start(ctx)
}

func (s *Service) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down telemetry ingestion service")
	shutdownCtx, cancel := context.WithTimeout(ctx, s.cfg.ShutdownTimeout)
	defer cancel()
	if err := s.batchWriter.Shutdown(shutdownCtx); err != nil {
		s.logger.Warn("batch writer shutdown error", "error", err)
	}
	if err := s.bus.Close(); err != nil {
		s.logger.Warn("NATS close error", "error", err)
	}
	s.pool.Close()
	s.logger.Info("telemetry ingestion service stopped")
	return nil
}

type ingestionAdapter struct {
	resultRepo    *postgres.ResultRepository
	monitorV2Repo *postgres.MonitorV2Repository
}

func (a *ingestionAdapter) IngestProbeResult(ctx context.Context, result *domain.ProbeResult) (bool, error) {
	return a.resultRepo.InsertAndUpdateMonitor(ctx, result)
}

func (a *ingestionAdapter) UpdateMonitorV2(ctx context.Context, monitorID, status string, finishedAt time.Time) error {
	return a.monitorV2Repo.UpdateRunResult(ctx, monitorID, domain.MonitorStatus(status), finishedAt)
}
