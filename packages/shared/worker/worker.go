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

	"monitoring-platform/packages/shared/agents/agent/spool"
	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/metrics"
	"monitoring-platform/packages/shared/probe"
	"monitoring-platform/packages/shared/queue"
)

const (
	consumeBlock     = 5 * time.Second
	autoClaimIdle    = time.Minute
	autoClaimEvery   = 30 * time.Second
	maxDeliveries    = 5
	jobBufferSeconds = 5
)

// Limits configures the worker's concurrency controls. The global limit is
// always enforced; per-type and per-workspace limits (when set) bound how much
// of that global capacity a single check type or tenant may consume, so one
// workspace cannot saturate the probe fleet.
type Limits struct {
	// Global is the overall number of concurrent probe jobs (the existing
	// worker concurrency).
	Global int
	// PerType bounds concurrent jobs of a given monitor type (e.g. http: 500).
	PerType map[domain.MonitorType]int
	// PerWorkspace bounds concurrent jobs per workspace. Empty disables it.
	PerWorkspace int
}

type Worker struct {
	queue        queue.JobQueue
	registry     *probe.Registry
	spool        *spool.Spool
	runningJobs  atomic.Int32
	consumerName string
	limits       Limits
	typeSems     map[domain.MonitorType]chan struct{}
	logger       *slog.Logger
	metrics      *metrics.WorkerMetrics

	// wsSems guards per-workspace concurrency (when configured).
	wsMu  sync.Mutex
	wsSems map[string]chan struct{}
}

func New(
	probeQueue queue.JobQueue,
	registry *probe.Registry,
	spool *spool.Spool,
	consumerName string,
	concurrency int,
	logger *slog.Logger,
	workerMetrics *metrics.WorkerMetrics,
) *Worker {
	return NewWithLimits(probeQueue, registry, spool, consumerName, Limits{Global: concurrency}, logger, workerMetrics)
}

// NewWithLimits builds a worker with per-type/per-workspace concurrency
// controls layered on top of the global limit.
func NewWithLimits(
	probeQueue queue.JobQueue,
	registry *probe.Registry,
	spool *spool.Spool,
	consumerName string,
	limits Limits,
	logger *slog.Logger,
	workerMetrics *metrics.WorkerMetrics,
) *Worker {
	if limits.Global < 1 {
		limits.Global = 1
	}

	typeSems := make(map[domain.MonitorType]chan struct{}, len(limits.PerType))
	for monitorType, limit := range limits.PerType {
		if limit < 1 {
			continue
		}
		typeSems[monitorType] = make(chan struct{}, limit)
	}

	return &Worker{
		queue:        probeQueue,
		registry:     registry,
		spool:        spool,
		consumerName: consumerName,
		limits:       limits,
		typeSems:     typeSems,
		wsSems:       map[string]chan struct{}{},
		logger:       logger,
		metrics:      workerMetrics,
	}
}

func (w *Worker) AvailableSlots() int {
	available := int32(w.limits.Global) - w.runningJobs.Load()
	if available < 0 {
		return 0
	}
	return int(available)
}

func (w *Worker) RunningJobs() int {
	return int(w.runningJobs.Load())
}

func (w *Worker) Run(ctx context.Context) error {
	w.logger.Info("worker started",
		"consumer", w.consumerName,
		"concurrency", w.limits.Global,
		"per_type_limits", len(w.typeSems),
	)

	go w.reclaimLoop(ctx)

	semaphore := make(chan struct{}, w.limits.Global)
	var wg sync.WaitGroup

	for {
		if ctx.Err() != nil {
			wg.Wait()
			return ctx.Err()
		}

		messages, err := w.queue.Consume(ctx, w.consumerName, int64(w.limits.Global*2), consumeBlock)
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
			go func(message queue.Message) {
				defer wg.Done()
				defer func() { <-semaphore }()

				w.process(ctx, message)
			}(message)
		}
	}
}

func (w *Worker) process(ctx context.Context, message queue.Message) {
	// Decode early so per-type / per-workspace concurrency limits can be
	// enforced before the executor runs.
	var job domain.ProbeJob
	if raw, ok := message.Values["payload"]; ok {
		if err := json.Unmarshal([]byte(fmt.Sprint(raw)), &job); err != nil {
			// Un-decodable payload: poison immediately (handleMessage would
			// do the same after limits, but failing before limits is fine
			// because this message will never succeed).
			_ = w.poison(ctx, message, fmt.Sprintf("decode probe job: %v", err))
			return
		}
	}

	typeSem := w.typeSemaphore(job.Type)
	if typeSem != nil {
		select {
		case typeSem <- struct{}{}:
		case <-ctx.Done():
			return
		}
		defer func() { <-typeSem }()
	}

	wsSem := w.workspaceSemaphore(job.WorkspaceID)
	if wsSem != nil {
		select {
		case wsSem <- struct{}{}:
		case <-ctx.Done():
			return
		}
		defer func() { <-wsSem }()
	}

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

// typeSemaphore returns the per-type semaphore for the given monitor type,
// or nil when no limit is configured for it.
func (w *Worker) typeSemaphore(monitorType domain.MonitorType) chan struct{} {
	return w.typeSems[monitorType]
}

// workspaceSemaphore returns (creating on first use) a per-workspace
// semaphore when per-workspace limits are configured. Workspace ids live in a
// bounded map so an unbounded number of tenants cannot exhaust memory; the
// most-active workspaces keep their slots, older ones are evicted and will
// recreate their limiter on the next job.
func (w *Worker) workspaceSemaphore(workspaceID string) chan struct{} {
	if w.limits.PerWorkspace < 1 {
		return nil
	}
	if workspaceID == "" {
		return nil
	}

	w.wsMu.Lock()
	defer w.wsMu.Unlock()

	if sem, ok := w.wsSems[workspaceID]; ok {
		return sem
	}
	sem := make(chan struct{}, w.limits.PerWorkspace)

	// Bound the map so a flood of distinct workspace ids cannot grow it
	// without limit.
	const maxWorkspaceLimiters = 10_000
	if len(w.wsSems) >= maxWorkspaceLimiters {
		for key := range w.wsSems {
			delete(w.wsSems, key)
			break
		}
	}

	w.wsSems[workspaceID] = sem
	return sem
}

func (w *Worker) handleMessage(ctx context.Context, message queue.Message) error {
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
func (w *Worker) poison(ctx context.Context, message queue.Message, reason string) error {
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
			messages, err := w.queue.AutoClaim(ctx, w.consumerName, autoClaimIdle, int64(w.limits.Global))
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
