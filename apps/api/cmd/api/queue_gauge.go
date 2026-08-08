package main

import (
	"context"
	"time"

	"monitoring-platform/packages/shared/metrics"
	"monitoring-platform/packages/shared/queue"
)

// watchQueueDepth keeps the queue_pending_jobs gauge fresh.
func watchQueueDepth(ctx context.Context, probeQueue *queue.RedisQueue, ingestionMetrics *metrics.IngestionMetrics) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			statsCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
			stats, err := probeQueue.Stats(statsCtx)
			cancel()

			if err == nil {
				ingestionMetrics.QueuePendingJobs.Set(float64(stats.Lag + stats.Pending))
			}
		}
	}
}
