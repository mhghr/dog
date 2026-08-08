package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/ingestion/messagebus"
)

type Consumer struct {
	bus         messagebus.MessageBus
	ingestion   IngestProcessor
	batchWriter *BatchWriter
	dedup       *Deduplicator
	cb          *CircuitBreaker
	metrics     *TelemetryMetrics
	logger      *slog.Logger
	cfg         Config
}

type IngestProcessor interface {
	IngestProbeResult(ctx context.Context, result *domain.ProbeResult) (bool, error)
	UpdateMonitor(ctx context.Context, monitorID, status string, finishedAt time.Time) error
}

func NewConsumer(
	bus messagebus.MessageBus,
	ingestion IngestProcessor,
	batchWriter *BatchWriter,
	dedup *Deduplicator,
	cb *CircuitBreaker,
	metrics *TelemetryMetrics,
	logger *slog.Logger,
	cfg Config,
) *Consumer {
	return &Consumer{
		bus:         bus,
		ingestion:   ingestion,
		batchWriter: batchWriter,
		dedup:       dedup,
		cb:          cb,
		metrics:     metrics,
		logger:      logger,
		cfg:         cfg,
	}
}

func (c *Consumer) Start(ctx context.Context) error {
	for i := 0; i < c.cfg.Workers; i++ {
		workerID := i
		go c.run(ctx, workerID)
	}
	return nil
}

func (c *Consumer) run(ctx context.Context, workerID int) {
	logger := c.logger.With("worker_id", workerID)

	subOpts := messagebus.SubscribeOptions{
		Subject:    "telemetry.>",
		Queue:      c.cfg.QueueName,
		Durable:    c.cfg.ConsumerName,
		DeliverNew: true,
		Stream:     c.cfg.StreamName,
	}

	err := c.bus.Subscribe(ctx, subOpts, func(ctx context.Context, msg messagebus.Message) error {
		if c.metrics != nil {
			c.metrics.MessagesReceived.Inc()
		}

		if c.cb.State() == CBOpen {
			logger.Warn("circuit breaker open, NAKing message")
			return fmt.Errorf("circuit breaker is open")
		}

		var envelope Envelope
		if err := json.Unmarshal(msg.Data, &envelope); err != nil {
			logger.Error("failed to unmarshal envelope", "error", err)
			if c.metrics != nil {
				c.metrics.MessagesFailed.Inc()
			}
			return nil
		}

		if err := envelope.Validate(); err != nil {
			logger.Error("invalid envelope", "error", err, "event_id", envelope.EventID)
			if c.metrics != nil {
				c.metrics.MessagesFailed.Inc()
			}
			return nil
		}

		isDup, err := c.dedup.IsDuplicate(ctx, envelope.EventID)
		if err != nil {
			logger.Error("dedup check failed", "error", err)
			return fmt.Errorf("dedup check: %w", err)
		}
		if isDup {
			if c.metrics != nil {
				c.metrics.MessagesDuplicate.Inc()
			}
			return nil
		}

		if err := c.processEnvelope(ctx, &envelope); err != nil {
			logger.Error("process failed", "error", err, "event_id", envelope.EventID)
			if c.metrics != nil {
				c.metrics.MessagesFailed.Inc()
			}
			return fmt.Errorf("process: %w", err)
		}

		if c.metrics != nil {
			c.metrics.MessagesProcessed.Inc()
		}
		return nil
	})
	if err != nil {
		logger.Error("subscribe failed", "error", err)
		return
	}

	<-ctx.Done()
}

func (c *Consumer) processEnvelope(ctx context.Context, env *Envelope) error {
	switch env.Type {
	case TypeProbeResult:
		return c.processProbeResult(ctx, env)
	case TypeMetricBatch:
		return c.processMetricBatch(ctx, env)
	default:
		c.logger.Warn("skipping unhandled envelope type", "type", env.Type)
		return nil
	}
}

func (c *Consumer) processProbeResult(ctx context.Context, env *Envelope) error {
	var result domain.ProbeResult
	if err := env.UnmarshalValue(&result); err != nil {
		c.logger.Error("unmarshal probe result", "error", err)
		return nil
	}

	inserted, err := c.ingestion.IngestProbeResult(ctx, &result)
	if err != nil {
		return fmt.Errorf("ingest probe result: %w", err)
	}
	if !inserted {
		c.logger.Debug("probe result skipped (duplicate job_id)", "job_id", result.JobID)
		return nil
	}

	lines := c.buildVMLines(&result)
	ids, err := c.batchWriter.Add(env.EventID, lines)
	if err != nil {
		return fmt.Errorf("batch writer add: %w", err)
	}
	if ids == nil {
		return nil
	}

	for _, id := range ids {
		if err := c.dedup.MarkProcessed(ctx, id, nil); err != nil {
			c.logger.Warn("mark processed failed", "event_id", id, "error", err)
		}
	}
	return nil
}

func (c *Consumer) processMetricBatch(ctx context.Context, env *Envelope) error {
	var batch domain.MetricBatch
	if err := env.UnmarshalValue(&batch); err != nil {
		c.logger.Error("unmarshal metric batch", "error", err)
		return nil
	}

	lines := c.buildBatchVMLines(&batch)
	ids, err := c.batchWriter.Add(env.EventID, lines)
	if err != nil {
		return fmt.Errorf("batch writer add: %w", err)
	}
	if ids == nil {
		return nil
	}

	for _, id := range ids {
		if err := c.dedup.MarkProcessed(ctx, id, nil); err != nil {
			c.logger.Warn("mark processed failed", "event_id", id, "error", err)
		}
	}
	return nil
}

func (c *Consumer) buildVMLines(result *domain.ProbeResult) []string {
	labels := fmt.Sprintf(`monitor_id=%q`, result.MonitorID)
	if result.ProbeLocationID != "" {
		labels += fmt.Sprintf(`,probe_location=%q`, result.ProbeLocationID)
	}
	timestamp := result.FinishedAt.UnixMilli()
	successValue := 0
	if result.Success {
		successValue = 1
	}
	return []string{
		fmt.Sprintf(`monitor_probe_success{%s} %d %d`, labels, successValue, timestamp),
		fmt.Sprintf(`monitor_probe_duration_seconds{%s} %f %d`, labels, float64(result.DurationMillis)/1000, timestamp),
	}
}

func (c *Consumer) buildBatchVMLines(batch *domain.MetricBatch) []string {
	lines := make([]string, 0, len(batch.Samples))
	for _, s := range batch.Samples {
		labelParts := make([]string, 0, len(s.Labels)+3)
		for k, v := range s.Labels {
			escaped := strings.ReplaceAll(v, `\`, `\\`)
			escaped = strings.ReplaceAll(escaped, `"`, `\"`)
			escaped = strings.ReplaceAll(escaped, "\n", `\n`)
			labelParts = append(labelParts, fmt.Sprintf(`%s="%s"`, k, escaped))
		}
		if s.TenantID != "" {
			labelParts = append(labelParts, fmt.Sprintf(`tenant_id="%s"`, s.TenantID))
		}
		if s.AgentID != "" {
			labelParts = append(labelParts, fmt.Sprintf(`agent_id="%s"`, s.AgentID))
		}
		if s.Hostname != "" {
			labelParts = append(labelParts, fmt.Sprintf(`hostname="%s"`, s.Hostname))
		}
		labelsStr := ""
		if len(labelParts) > 0 {
			labelsStr = "{" + strings.Join(labelParts, ",") + "}"
		}
		timestamp := s.Timestamp.UnixMilli()
		lines = append(lines, fmt.Sprintf(`%s%s %g %d`, s.Name, labelsStr, s.Value, timestamp))
	}
	return lines
}
