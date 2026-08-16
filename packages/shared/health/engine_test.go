package health

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"monitoring-platform/packages/shared/domain"
)

type stubHealthRepo struct{}

func (stubHealthRepo) ListParameterCatalog(ctx context.Context, monitorType string) ([]ParameterDefinition, error) {
	return AllParameters[monitorType], nil
}
func (stubHealthRepo) GetParameterDefinition(ctx context.Context, monitorType, key string) (ParameterDefinition, error) {
	for _, p := range AllParameters[monitorType] {
		if p.Key == key {
			return p, nil
		}
	}
	return ParameterDefinition{}, domain.ErrNotFound
}
func (stubHealthRepo) ListParameterRules(ctx context.Context, monitorID string) ([]ParameterRule, error) {
	return nil, nil
}
func (stubHealthRepo) GetParameterRule(ctx context.Context, monitorID, parameterKey string) (ParameterRule, error) {
	return ParameterRule{}, domain.ErrNotFound
}
func (stubHealthRepo) UpsertParameterRule(ctx context.Context, rule *ParameterRule) error { return nil }
func (stubHealthRepo) DeleteParameterRule(ctx context.Context, monitorID, parameterKey string) error {
	return nil
}
func (stubHealthRepo) UpsertHealthState(ctx context.Context, state *ParameterHealthState) error { return nil }
func (stubHealthRepo) GetHealthState(ctx context.Context, monitorID, parameterKey string) (ParameterHealthState, error) {
	return ParameterHealthState{}, domain.ErrNotFound
}
func (stubHealthRepo) ListHealthStates(ctx context.Context, monitorID string) ([]ParameterHealthState, error) {
	return nil, nil
}
func (stubHealthRepo) ListNotificationChannels(ctx context.Context) ([]HealthNotificationChannel, error) {
	return nil, nil
}
func (stubHealthRepo) GetNotificationChannel(ctx context.Context, id string) (HealthNotificationChannel, error) {
	return HealthNotificationChannel{}, domain.ErrNotFound
}
func (stubHealthRepo) CreateNotificationChannel(ctx context.Context, ch *HealthNotificationChannel) error {
	return nil
}
func (stubHealthRepo) UpdateNotificationChannel(ctx context.Context, ch *HealthNotificationChannel) error {
	return nil
}
func (stubHealthRepo) DeleteNotificationChannel(ctx context.Context, id string) error { return nil }
func (stubHealthRepo) ListNotificationPolicies(ctx context.Context, monitorID string) ([]NotificationPolicy, error) {
	return nil, nil
}
func (stubHealthRepo) GetNotificationPolicy(ctx context.Context, id string) (NotificationPolicy, error) {
	return NotificationPolicy{}, domain.ErrNotFound
}
func (stubHealthRepo) CreateNotificationPolicy(ctx context.Context, policy *NotificationPolicy) error {
	return nil
}
func (stubHealthRepo) UpdateNotificationPolicy(ctx context.Context, policy *NotificationPolicy) error {
	return nil
}
func (stubHealthRepo) DeleteNotificationPolicy(ctx context.Context, id string) error { return nil }

func TestEvaluateBooleanFailure(t *testing.T) {
	if got := evaluateBooleanFailure([]float64{1}, nil); got != HealthOK {
		t.Fatalf("value 1 (reachable) should be HealthOK, got %s", got)
	}
	if got := evaluateBooleanFailure([]float64{0}, nil); got != HealthError {
		t.Fatalf("value 0 (down) should be HealthError, got %s", got)
	}
	if got := evaluateBooleanFailure(nil, nil); got != HealthUnknown {
		t.Fatalf("no values should be HealthUnknown, got %s", got)
	}
}

func TestEngineEvaluatePingReachability(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	engine := NewEngine(stubHealthRepo{}, logger)

	def := pingReachabilityDef()
	rule := defaultRuleFromCatalog(def, "m1")

	state := engine.EvaluateParameter(context.Background(), "m1", "ping.reachability", []float64{0}, &rule, def)
	if state != HealthError {
		t.Fatalf("down ping reachability should be HealthError, got %s", state)
	}
	state = engine.EvaluateParameter(context.Background(), "m1", "ping.reachability", []float64{1}, &rule, def)
	if state != HealthOK {
		t.Fatalf("up ping reachability should be HealthOK, got %s", state)
	}
}

func pingReachabilityDef() ParameterDefinition {
	for _, p := range PingParameters {
		if p.Key == "ping.reachability" {
			return p
		}
	}
	panic("ping.reachability not found")
}
