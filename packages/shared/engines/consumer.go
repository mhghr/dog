package engines

import (
	"context"
	"encoding/json"
	"log/slog"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/ingestion/messagebus"
	"monitoring-platform/packages/shared/telemetry/ingest"
)

// ConsumerOptions configures a standalone engine's NATS result consumer.
type ConsumerOptions struct {
	Subject string
	Queue   string
	Durable string
	Stream  string
	Workers int
}

// StartResultConsumer subscribes a durable, at-least-once queue-group consumer
// to a result subject and hands each decoded probe result to handler. Malformed
// messages are acknowledged without redelivery; handler errors are nacked and
// redelivered by JetStream.
func StartResultConsumer(
	ctx context.Context,
	bus *messagebus.NATSBus,
	opts ConsumerOptions,
	handler func(ctx context.Context, result *domain.ProbeResult) error,
	logger *slog.Logger,
) error {
	if opts.Subject == "" {
		return nil
	}
	if opts.Stream == "" {
		opts.Stream = "ENGINE_EVENTS"
	}
	if opts.Queue == "" {
		opts.Queue = "engine-consumers"
	}
	if opts.Durable == "" {
		opts.Durable = "engine-consumer"
	}
	if opts.Workers < 1 {
		opts.Workers = 1
	}

	for i := 0; i < opts.Workers; i++ {
		workerID := i
		go func() {
			err := bus.Subscribe(ctx, messagebus.SubscribeOptions{
				Subject:    opts.Subject,
				Queue:      opts.Queue,
				Durable:    opts.Durable,
				Stream:     opts.Stream,
				DeliverNew: true,
			}, func(ctx context.Context, msg messagebus.Message) error {
				if err := processResultMessage(ctx, msg, handler, logger); err != nil {
					logger.Error("engine consumer: handler failed",
						"error", err,
						"subject", msg.Subject,
						"worker_id", workerID,
					)
					return err
				}
				logger.Debug("engine consumer: processed",
					"subject", msg.Subject,
					"worker_id", workerID,
				)
				return nil
			})
			if err != nil {
				logger.Error("engine consumer: subscribe failed", "error", err, "subject", opts.Subject)
				return
			}
		}()
	}

	return nil
}

// processResultMessage decodes a probe-result envelope and hands the result to
// handler. Malformed or invalid messages are acknowledged (return nil); handler
// errors are propagated so JetStream redelivers.
func processResultMessage(ctx context.Context, msg messagebus.Message, handler func(ctx context.Context, result *domain.ProbeResult) error, logger *slog.Logger) error {
	if logger == nil {
		logger = slog.Default()
	}

	var envelope ingest.Envelope
	if err := json.Unmarshal(msg.Data, &envelope); err != nil {
		logger.Error("engine consumer: unmarshal envelope failed", "error", err, "subject", msg.Subject)
		return nil
	}
	if err := envelope.Validate(); err != nil {
		logger.Error("engine consumer: invalid envelope", "error", err, "subject", msg.Subject)
		return nil
	}

	var result domain.ProbeResult
	if err := envelope.UnmarshalValue(&result); err != nil {
		logger.Error("engine consumer: unmarshal probe result failed", "error", err, "subject", msg.Subject)
		return nil
	}

	return handler(ctx, &result)
}
