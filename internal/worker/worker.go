// Package worker consumes probe jobs, executes probes, and delivers results
// to the ingestion API.
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"

	"monitoring-platform/internal/agent/spool"
	"monitoring-platform/internal/domain"
	"monitoring-platform/internal/metrics"
	"monitoring-platform/internal/probe"
	"monitoring-platform/internal/queue"
)

const (
	consumeBlock     = 5 * time.Second
	autoClaimIdle    = time.Minute
	autoClaimEvery   = 30 * time.Second
	maxDeliveries    = 5
	jobBufferSeconds = 5
)

type Worker struct {
	queue        *queue.RedisQueue
	registry     *probe.Registry
	spool        *spool.Spool
	runningJobs  atomic.Int32
	consumerName string
	concurrency  int
	logger       *slog.Logger
	metrics      *metrics.WorkerMetrics
}

func New(
	probeQueue *queue.RedisQueue,
	registry *probe.Registry,
	spool *spool.Spool,
	consumerName string,
	concurrency int,
	logger *slog.Logger,
	workerMetrics *metrics.WorkerMetrics,
) *Worker {
	if concurrency < 1 {
		concurrency = 1
	}

	return &Worker{
		queue:        probeQueue,
		registry:     registry,
		spool:        spool,
		consumerName: consumerName,
		concurrency:  concurrency,
		logger:       logger,
		metrics:      workerMetrics,
	}
}

func (w *Worker) AvailableSlots() int {
	available := int32(w.concurrency) - w.runningJobs.Load()
	if available < 0 {
		return 0
	}
	return int(available)
}

func (w *Worker) RunningJobs() int {
	return int(w.runningJobs.Load())
}

func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("worker started", "consumer", w.consumerName, "concurrency", w.concurrency)

	go w.reclaimLoop(ctx)

	semaphore := make(chan struct{}, w.concurrency)
	var wg sync.WaitGroup

	for {
		if ctx.Err() != nil {
			wg.Wait()
			return ctx.Err()
		}

		messages, err := w.queue.Consume(ctx, w.consumerName, int64(w.concurrency*2), consumeBlock)
		if err != nil {
			if ctx.Err() != nil {
				wg.Wait()
				return ctx.Err()
			}
			w.logger.Error("consume jobs failed", "error", err)
			time.Sleep(time.Second)
			continue
		}

		for _, message := range messages {
			w.metrics.JobsReceived.Inc()

			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				wg.Wait()
				return ctx.Err()
			}

			wg.Add(1)
			go func(message redis.XMessage) {
				defer wg.Done()
				defer func() { <-semaphore }()

				w.process(ctx, message)
			}(message)
		}
	}
}

func (w *Worker) process(ctx context.Context, message redis.XMessage) {
	w.runningJobs.Add(1)
	defer w.runningJobs.Add(-1)

	if err := w.handleMessage(ctx, message); err != nil {
		w.metrics.JobsFailed.Inc()
		w.logger.Error("handle probe job failed", "message_id", message.ID, "error", err)
		// No ack: the message stays pending and is retried or dead-lettered.
		return
	}

	if err := w.queue.Ack(ctx, message.ID); err != nil {
		w.logger.Error("ack job failed", "message_id", message.ID, "error", err)
		return
	}

	w.metrics.JobsCompleted.Inc()
}

func (w *Worker) handleMessage(ctx context.Context, message redis.XMessage) error {
	rawPayload, ok := message.Values["payload"]
	if !ok {
		return w.poison(ctx, message, "payload is missing")
	}

	var job domain.ProbeJob
	if err := json.Unmarshal([]byte(fmt.Sprint(rawPayload)), &job); err != nil {
		return w.poison(ctx, message, fmt.Sprintf("decode probe job: %v", err))
	}

	if !job.Deadline.IsZero() && time.Now().UTC().After(job.Deadline) {
		w.logger.Warn("job deadline exceeded, skipping",
			"job_id", job.ID,
			"deadline", job.Deadline,
		)
		w.metrics.JobsFailed.Inc()
		return w.poison(ctx, message, "deadline exceeded")
	}

	if job.Attempt == 0 {
		deliveries, err := w.queue.DeliveryCount(ctx, message.ID)
		if err == nil {
			job.Attempt = int(deliveries) + 1
		} else {
			job.Attempt = 1
		}
	}

	executor, ok := w.registry.Get(job.Type)
	if !ok {
		return w.poison(ctx, message, fmt.Sprintf("unsupported monitor type: %s", job.Type))
	}

	if job.TimeoutMillis <= 0 {
		job.TimeoutMillis = 5000
	}

	totalBudget := time.Duration(job.TimeoutMillis)*time.Millisecond*time.Duration(job.Retries+1) +
		time.Duration(job.Retries)*500*time.Millisecond +
		jobBufferSeconds*time.Second

	jobCtx, cancel := context.WithTimeout(ctx, totalBudget)
	defer cancel()

	probeStart := time.Now()
	result := probe.ExecuteWithRetry(jobCtx, executor, job)
	w.metrics.ProbeDuration.
		WithLabelValues(string(job.Type), fmt.Sprintf("%t", result.Success)).
		Observe(time.Since(probeStart).Seconds())

	w.logger.Info(
		"probe completed",
		"job_id", result.JobID,
		"monitor_id", result.MonitorID,
		"type", job.Type,
		"success", result.Success,
		"duration_ms", result.DurationMillis,
		"error_code", result.ErrorCode,
		"attempt", job.Attempt,
	)

	if err := w.spool.Store(message.ID, &result); err != nil {
		return fmt.Errorf("store probe result: %w", err)
	}

	return nil
}

// poison moves undecodable/unsupported messages straight to the dead letter
// stream so they never clog the consumer group.
func (w *Worker) poison(ctx context.Context, message redis.XMessage, reason string) error {
	w.logger.Warn("dead-lettering poison message", "message_id", message.ID, "reason", reason)

	if err := w.queue.DeadLetter(ctx, message, reason); err != nil {
		return fmt.Errorf("dead letter poison message: %w", err)
	}

	return nil
}

// reclaimLoop recovers jobs abandoned by crashed workers and dead-letters
// messages that exceeded the delivery budget.
func (w *Worker) reclaimLoop(ctx context.Context) {
	ticker := time.NewTicker(autoClaimEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			messages, err := w.queue.AutoClaim(ctx, w.consumerName, autoClaimIdle, int64(w.concurrency))
			if err != nil {
				w.logger.Error("auto claim failed", "error", err)
				continue
			}

			for _, message := range messages {
				deliveries, err := w.queue.DeliveryCount(ctx, message.ID)
				if err == nil && deliveries > maxDeliveries {
					if err := w.queue.DeadLetter(ctx, message, fmt.Sprintf("exceeded %d deliveries", maxDeliveries)); err != nil {
						w.logger.Error("dead letter failed", "message_id", message.ID, "error", err)
					}
					continue
				}

				w.metrics.JobsReceived.Inc()
				w.process(ctx, message)
			}
		}
	}
}
