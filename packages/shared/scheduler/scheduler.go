// Package scheduler finds due monitors and publishes probe jobs.
package scheduler

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/metrics"
	"monitoring-platform/packages/shared/queue"
	"monitoring-platform/packages/shared/repository"
)

type Scheduler struct {
	monitors  repository.MonitorRepository
	locations []domain.ProbeLocation
	queue     *queue.RedisQueue
	batchSize int
	interval  time.Duration
	logger    *slog.Logger
	metrics   *metrics.SchedulerMetrics
}

func New(
	monitors repository.MonitorRepository,
	probeQueue *queue.RedisQueue,
	locations []domain.ProbeLocation,
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
		monitors:  monitors,
		queue:     probeQueue,
		locations: locations,
		batchSize: batchSize,
		interval:  interval,
		logger:    logger,
		metrics:   schedulerMetrics,
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	s.logger.Info("scheduler started",
		"batch_size", s.batchSize,
		"interval", s.interval.String(),
		"locations", len(s.locations),
	)

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

func (s *Scheduler) ScheduleForLocation(ctx context.Context, locationCode string, limit int) error {
	batchSize := s.batchSize
	if limit > 0 {
		batchSize = limit
	}

	loc, found := s.findLocation(locationCode)
	if !found {
		s.logger.Warn("location not found", "code", locationCode)
		return nil
	}

	published, err := s.monitors.ClaimDue(ctx, batchSize, func(monitor domain.Monitor) error {
		return s.publishToLocation(ctx, monitor, loc)
	})

	if err != nil {
		return err
	}

	if published > 0 {
		s.logger.Debug("scheduler batch published for location",
			"jobs", published, "location", locationCode)
	}

	return nil
}

func (s *Scheduler) scheduleBatch(ctx context.Context) error {
	batchStart := time.Now()

	published, err := s.monitors.ClaimDue(ctx, s.batchSize, func(monitor domain.Monitor) error {
		return s.publishToAllLocations(ctx, monitor)
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

func (s *Scheduler) publishToAllLocations(ctx context.Context, monitor domain.Monitor) error {
	var lastErr error
	successCount := 0

	for _, loc := range s.locations {
		if err := s.publishToLocation(ctx, monitor, loc); err != nil {
			s.metrics.PublishErrors.Inc()
			s.logger.Error("publish probe job failed",
				"monitor_id", monitor.ID, "location", loc.Code, "error", err)
			lastErr = err
			continue
		}
		successCount++
	}

	if successCount > 0 {
		s.metrics.JobsPublished.Add(float64(successCount))
		return nil
	}

	return lastErr
}

func (s *Scheduler) publishToLocation(ctx context.Context, monitor domain.Monitor, loc domain.ProbeLocation) error {
	job := domain.ProbeJob{
		ID:              uuid.NewString(),
		MonitorID:       monitor.ID,
		Type:            monitor.ProbeType,
		Target:          monitor.ResourceTarget,
		TimeoutMillis:   monitor.TimeoutMillis,
		Retries:         monitor.Retries,
		Config:          monitor.Configuration,
		ProbeLocationID: loc.ID,
		ScheduledAt:     time.Now().UTC(),
	}

	if err := s.queue.Publish(ctx, job); err != nil {
		return err
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}

	return s.queue.PublishToLocation(ctx, loc.Code, payload)
}

func (s *Scheduler) findLocation(code string) (domain.ProbeLocation, bool) {
	for _, loc := range s.locations {
		if loc.Code == code {
			return loc, true
		}
	}
	return domain.ProbeLocation{}, false
}
