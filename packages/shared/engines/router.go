// Package engines routes successfully-persisted probe results to the health
// and alert engines. Each engine runs either inline (in-process, the default)
// or in the standalone monitor-engine / alert-engine binaries consuming from a
// NATS subject (mode "nats").
package engines

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"

	"monitoring-platform/packages/shared/alerting"
	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/health"
	"monitoring-platform/packages/shared/ingestion/messagebus"
	"monitoring-platform/packages/shared/telemetry/ingest"
)

// NATS subjects used to route results to the standalone engines.
const (
	HealthSubject = "engine.health.eval"
	AlertSubject  = "engine.alert.eval"
)

// Engine execution modes.
const (
	ModeInline = "inline"
	ModeNATS   = "nats"
)

// Publisher is the NATS publish seam the router uses to hand results to the
// standalone engines. *messagebus.NATSBus implements it.
type Publisher interface {
	Publish(ctx context.Context, opts messagebus.PublishOptions) error
}

// Router dispatches a persisted probe result to the health and alert engines.
// Mode is controlled per engine; a nil or unrecognized mode falls back to
// inline execution.
type Router struct {
	healthMode     string
	alertMode      string
	bus            Publisher
	healthEngine   *health.Engine
	healthNotifier *health.NotificationEngine
	alertEngine    *alerting.Engine
	alertNotifier  *alerting.Notifier
	logger         *slog.Logger
}

func NewRouter(
	healthMode, alertMode string,
	bus Publisher,
	healthEngine *health.Engine,
	healthNotifier *health.NotificationEngine,
	alertEngine *alerting.Engine,
	alertNotifier *alerting.Notifier,
	logger *slog.Logger,
) *Router {
	if healthMode == "" {
		healthMode = ModeInline
	}
	if alertMode == "" {
		alertMode = ModeInline
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &Router{
		healthMode:     healthMode,
		alertMode:      alertMode,
		bus:            bus,
		healthEngine:   healthEngine,
		healthNotifier: healthNotifier,
		alertEngine:    alertEngine,
		alertNotifier:  alertNotifier,
		logger:         logger,
	}
}

// RouteResult evaluates (or routes) a successfully-persisted probe result
// against both engines. It never returns an error: engine failures are logged
// and must not fail the ingestion path.
func (r *Router) RouteResult(ctx context.Context, result *domain.ProbeResult) {
	r.routeHealth(ctx, result)
	r.routeAlert(ctx, result)
}

func (r *Router) routeHealth(ctx context.Context, result *domain.ProbeResult) {
	if r.healthMode == ModeNATS {
		if err := r.publish(ctx, HealthSubject, result); err != nil {
			r.logger.Error("health engine: publish failed", "error", err)
		}
		return
	}

	if r.healthEngine == nil {
		return
	}

	outcomes, err := r.healthEngine.EvaluateResult(ctx, result)
	if err != nil {
		r.logger.Warn("health engine: evaluation failed", "error", err)
		return
	}
	if r.healthNotifier == nil {
		return
	}
	for i := range outcomes {
		r.healthNotifier.Evaluate(ctx, outcomes[i])
	}
}

func (r *Router) routeAlert(ctx context.Context, result *domain.ProbeResult) {
	if r.alertMode == ModeNATS {
		if err := r.publish(ctx, AlertSubject, result); err != nil {
			r.logger.Error("alert engine: publish failed", "error", err)
		}
		return
	}

	if r.alertEngine == nil {
		return
	}

	events := r.alertEngine.Evaluate(ctx, *result)
	if r.alertNotifier == nil {
		return
	}
	for _, evt := range events {
		r.alertNotifier.Dispatch(ctx, evt, evt.ChannelIDs)
	}
}

// publish wraps a probe result in the shared telemetry envelope and publishes
// it to the given NATS subject.
func (r *Router) publish(ctx context.Context, subject string, result *domain.ProbeResult) error {
	if r.bus == nil {
		return errors.New("nats bus is nil")
	}
	envelope, err := ingest.NewProbeResultEnvelope(uuid.NewString(), "api", result.MonitorID, "", "", result)
	if err != nil {
		return err
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return r.bus.Publish(ctx, messagebus.PublishOptions{Subject: subject, Data: data})
}
