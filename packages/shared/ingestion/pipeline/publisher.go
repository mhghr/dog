package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/ingestion/messagebus"
)

// IngestionPublisher authenticates, enriches, validates, and publishes metric
// batches to the message bus.
type IngestionPublisher struct {
	bus        messagebus.MessageBus
	auth       Authenticator
	tenantRes  *TenantResolver
	normalizer *MetricNormalizer
	enricher   *MetricEnricher
	logger     *slog.Logger
}

// NewIngestionPublisher creates an ingestion publisher.
func NewIngestionPublisher(
	bus messagebus.MessageBus,
	auth Authenticator,
	tenantRes *TenantResolver,
	normalizer *MetricNormalizer,
	enricher *MetricEnricher,
	logger *slog.Logger,
) *IngestionPublisher {
	return &IngestionPublisher{
		bus:        bus,
		auth:       auth,
		tenantRes:  tenantRes,
		normalizer: normalizer,
		enricher:   enricher,
		logger:     logger,
	}
}

// PublishBatch authenticates the request context, enriches the batch with
// tenant/agent identity, validates it, and publishes to the message bus.
func (p *IngestionPublisher) PublishBatch(ctx context.Context, batch *domain.MetricBatch) error {
	identity, err := p.auth.Authenticate(ctx)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	tenantID, err := p.tenantRes.Resolve(ctx, identity.AgentID)
	if err != nil {
		return fmt.Errorf("tenant resolution failed: %w", err)
	}
	identity.TenantID = tenantID

	p.normalizer.Normalize(batch)
	p.enricher.Enrich(batch, identity)

	validation := ValidateMetrics(batch)
	if !validation.Valid && len(batch.Samples) == 0 {
		return fmt.Errorf("all metrics invalid: %v", validation.Errors)
	}

	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("marshal batch: %w", err)
	}

	subject := fmt.Sprintf("metrics.%s.%s", tenantID, identity.AgentID)
	if err := p.bus.Publish(ctx, messagebus.PublishOptions{
		Subject: subject,
		Data:    data,
		Headers: map[string]string{
			"tenant_id": tenantID,
			"agent_id":  identity.AgentID,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		},
	}); err != nil {
		return fmt.Errorf("publish to message bus: %w", err)
	}

	if validation.Skipped > 0 {
		p.logger.Warn("metrics published with drops",
			"agent_id", identity.AgentID,
			"tenant_id", tenantID,
			"samples", len(batch.Samples),
			"skipped", validation.Skipped,
			"subject", subject,
		)
	}

	p.logger.Info("metrics published",
		"agent_id", identity.AgentID,
		"tenant_id", tenantID,
		"samples", len(batch.Samples),
		"subject", subject,
	)

	return nil
}
