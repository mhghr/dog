// Package scheduler finds due monitors and publishes probe jobs.
package scheduler

import (
	"context"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"monitoring-platform/internal/domain"
	"monitoring-platform/internal/metrics"
	"monitoring-platform/internal/queue"
	"monitoring-platform/internal/repository"
)

type Scheduler struct {
	monitors   repository.MonitorRepository
	queue      *queue.RedisQueue
	locationID string
	batchSize  int
	interval   time.Duration
	logger     *slog.Logger
	metrics    *metrics.SchedulerMetrics
}

func New(
	monitors repository.MonitorRepository,
	probeQueue *queue.RedisQueue,
	locationID string,
	batchSize int,
	interval time.Duration,
	logger *slog.Logger,
	schedulerMetrics *metrics.SchedulerMetrics,
) *Scheduler {
	if batchSize <= 0 {
		batchSize = 100
	}
	if interval <= 0 {
		interval = time.Second
	}

	return &Scheduler{
		monitors:   monitors,
		queue:      probeQueue,
		locationID: locationID,
		batchSize:  batchSize,
		interval:   interval,
		logger:     logger,
		metrics:    schedulerMetrics,
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logger.Info("scheduler started", "batch_size", s.batchSize, "interval", s.interval.String())

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case <-ticker.C:
			if err := s.scheduleBatch(ctx); err != nil {
				s.logger.Error("scheduler batch failed", "error", err)
			}
		}
	}
}

func (s *Scheduler) scheduleBatch(ctx context.Context) error {
	batchStart := time.Now()

	published, err := s.monitors.ClaimDue(ctx, s.batchSize, func(monitor domain.Monitor) error {
		job := domain.ProbeJob{
			ID:              uuid.NewString(),
			MonitorID:       monitor.ID,
			Type:            monitor.Type,
			Target:          monitor.Target,
			TimeoutMillis:   monitor.TimeoutMillis,
			Retries:         monitor.Retries,
			Config:          monitor.Config,
			ProbeLocationID: s.locationID,
			ScheduledAt:     time.Now().UTC(),
		}

		if err := s.queue.Publish(ctx, job); err != nil {
			s.metrics.PublishErrors.Inc()
			s.logger.Error("publish probe job failed", "monitor_id", monitor.ID, "error", err)
			return err
		}

		s.metrics.JobsPublished.Inc()
		return nil
	})

	s.metrics.BatchDuration.Observe(time.Since(batchStart).Seconds())

	if err != nil {
		return err
	}

	if published > 0 {
		s.logger.Debug("scheduler batch published", "jobs", published)
	}

	return nil
}
