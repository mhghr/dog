package processor

import (
	"context"
	"encoding/json"
	"log/slog"

	"monitoring-platform/internal/domain"
	"monitoring-platform/internal/ingestion/messagebus"
)

// DLQConsumer subscribes to the metric dead-letter stream and logs rejected
// batches so they can be diagnosed or replayed manually.
type DLQConsumer struct {
	bus    messagebus.MessageBus
	logger *slog.Logger
}

// NewDLQConsumer creates a DLQ consumer bound to a message bus.
func NewDLQConsumer(bus messagebus.MessageBus, logger *slog.Logger) *DLQConsumer {
	return &DLQConsumer{bus: bus, logger: logger}
}

// Start subscribes to metrics.dlq.> and logs every dead-lettered batch.
func (c *DLQConsumer) Start(ctx context.Context) error {
	return c.bus.Subscribe(ctx, messagebus.SubscribeOptions{
		Subject:    "metrics.dlq.>",
		Queue:      "dlq-consumers",
		Durable:    "metric-dlq-consumer",
		DeliverNew: true,
		Stream:     "METRICS_DLQ",
	}, func(ctx context.Context, msg messagebus.Message) error {
		var batch domain.MetricBatch
		if err := json.Unmarshal(msg.Data, &batch); err != nil {
			c.logger.Error("dlq: unparseable message", "subject", msg.Subject, "error", err)
			return nil // do not redeliver undecodable messages
		}

		c.logger.Error("dlq: metric batch rejected after max deliveries",
			"subject", msg.Subject,
			"agent_id", batch.AgentID,
			"tenant_id", batch.TenantID,
			"samples", len(batch.Samples),
			"first_metric", firstMetricName(batch),
		)
		return nil
	})
}

func firstMetricName(batch domain.MetricBatch) string {
	if len(batch.Samples) > 0 {
		return batch.Samples[0].Name
	}
	return ""
}
