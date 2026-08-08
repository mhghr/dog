package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/ingestion/messagebus"
	"monitoring-platform/packages/shared/telemetry/ingest"
)

type NATSResultSender struct {
	bus     messagebus.MessageBus
	logger  *slog.Logger
	subject string
}

func NewNATSResultSender(bus messagebus.MessageBus, logger *slog.Logger) *NATSResultSender {
	return &NATSResultSender{
		bus:     bus,
		logger:  logger,
		subject: "telemetry.probe.result",
	}
}

func (s *NATSResultSender) SendBatch(ctx context.Context, results []*domain.ProbeResult) ([]string, error) {
	stored := make([]string, 0, len(results))

	for _, result := range results {
		eventID := uuid.NewString()
		envelope, err := ingest.NewProbeResultEnvelope(eventID, "worker", result.MonitorID, "", "", result)
		if err != nil {
			s.logger.Error("failed to create envelope", "error", err, "monitor_id", result.MonitorID)
			continue
		}
		data, err := json.Marshal(envelope)
		if err != nil {
			s.logger.Error("failed to marshal envelope", "error", err)
			continue
		}
		err = s.bus.Publish(ctx, messagebus.PublishOptions{
			Subject: s.subject,
			Data:    data,
		})
		if err != nil {
			s.logger.Error("failed to publish to NATS", "error", err, "monitor_id", result.MonitorID)
			continue
		}
		stored = append(stored, result.ID)
	}

	if len(stored) == 0 && len(results) > 0 {
		return nil, fmt.Errorf("failed to publish any of %d results to NATS", len(results))
	}

	return stored, nil
}
