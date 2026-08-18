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

func TestEvaluateThresholds(t *testing.T) {
	compare := compareValue

	cases := []struct {
		name          string
		rule          ParameterRule
		value         float64
		expectedState HealthState
	}{
		{
			name: "below error threshold",
			rule: ParameterRule{
				ErrorOperator: "gte",
				ErrorValue:    floatPtr(200),
				WarningValue:  floatPtr(100),
			},
			value:         50,
			expectedState: HealthOK,
		},
		{
			name: "warning threshold",
			rule: ParameterRule{
				ErrorOperator: "gte",
				ErrorValue:    floatPtr(200),
				WarningValue:  floatPtr(100),
			},
			value:         150,
			expectedState: HealthWarning,
		},
		{
			name: "error threshold",
			rule: ParameterRule{
				ErrorOperator: "gte",
				ErrorValue:    floatPtr(200),
				WarningValue:  floatPtr(100),
			},
			value:         250,
			expectedState: HealthError,
		},
		{
			name: "warning takes precedence only when ok",
			rule: ParameterRule{
				ErrorOperator: "gte",
				ErrorValue:    floatPtr(100),
				WarningValue:  floatPtr(200),
			},
			value:         250,
			expectedState: HealthError,
		},
		{
			name: "no thresholds",
			rule: ParameterRule{
				WarningOperator: "gte",
			},
			value:         250,
			expectedState: HealthOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := evaluateThresholds(tc.value, &tc.rule, stubHealthRepo{}, context.Background(), "m1", "param", compare)
			if got != tc.expectedState {
				t.Fatalf("expected %s, got %s", tc.expectedState, got)
			}
		})
	}
}

func TestEvaluateDirectionalLowerIsWorse(t *testing.T) {
	rule := ParameterRule{
		ErrorOperator:   "gte",
		ErrorValue:      floatPtr(20),
		WarningOperator: "gte",
		WarningValue:    floatPtr(50),
	}
	if got := evaluateDirectional([]float64{10}, &rule, ParameterDefinition{}, stubHealthRepo{}, context.Background(), "m1", "param", compareValueLower); got != HealthError {
		t.Fatalf("low value should be HealthError, got %s", got)
	}
	if got := evaluateDirectional([]float64{30}, &rule, ParameterDefinition{}, stubHealthRepo{}, context.Background(), "m1", "param", compareValueLower); got != HealthWarning {
		t.Fatalf("mid value should be HealthWarning, got %s", got)
	}
	if got := evaluateDirectional([]float64{80}, &rule, ParameterDefinition{}, stubHealthRepo{}, context.Background(), "m1", "param", compareValueLower); got != HealthOK {
		t.Fatalf("high value should be HealthOK, got %s", got)
	}
}

func TestEvaluateDirectionalMissingData(t *testing.T) {
	rule := ParameterRule{MissingDataPolicy: "IGNORE"}
	if got := evaluateDirectional(nil, &rule, ParameterDefinition{}, stubHealthRepo{}, context.Background(), "m1", "param", compareValue); got != HealthUnknown {
		t.Fatalf("IGNORE policy should be HealthUnknown, got %s", got)
	}

	rule = ParameterRule{MissingDataPolicy: "ERROR"}
	if got := evaluateDirectional(nil, &rule, ParameterDefinition{}, stubHealthRepo{}, context.Background(), "m1", "param", compareValue); got != HealthError {
		t.Fatalf("ERROR policy should be HealthError, got %s", got)
	}

	rule = ParameterRule{MissingDataPolicy: "WARNING"}
	if got := evaluateDirectional(nil, &rule, ParameterDefinition{}, stubHealthRepo{}, context.Background(), "m1", "param", compareValue); got != HealthWarning {
		t.Fatalf("WARNING policy should be HealthWarning, got %s", got)
	}
}

func TestAggregateValues(t *testing.T) {
	values := []float64{10, 20, 30}
	if got := aggregateValues(values, "avg"); got != 20 {
		t.Errorf("avg: expected 20, got %f", got)
	}
	if got := aggregateValues(values, "min"); got != 10 {
		t.Errorf("min: expected 10, got %f", got)
	}
	if got := aggregateValues(values, "max"); got != 30 {
		t.Errorf("max: expected 30, got %f", got)
	}
	if got := aggregateValues(values, "sum"); got != 60 {
		t.Errorf("sum: expected 60, got %f", got)
	}
	if got := aggregateValues(values, "last"); got != 30 {
		t.Errorf("last: expected 30, got %f", got)
	}
	if got := aggregateValues(nil, "avg"); got != 0 {
		t.Errorf("empty: expected 0, got %f", got)
	}
}

func TestToFloat64(t *testing.T) {
	cases := []struct {
		input    any
		expected float64
		ok       bool
	}{
		{float64(1.5), 1.5, true},
		{float32(1.5), 1.5, true},
		{int(7), 7, true},
		{int64(8), 8, true},
		{int32(9), 9, true},
		{true, 1.0, true},
		{false, 0.0, true},
		{"nope", 0, false},
		{nil, 0, false},
	}
	for _, tc := range cases {
		got, ok := toFloat64(tc.input)
		if ok != tc.ok || (ok && got != tc.expected) {
			t.Errorf("toFloat64(%v) = %f,%v; want %f,%v", tc.input, got, ok, tc.expected, tc.ok)
		}
	}
}

func floatPtr(v float64) *float64 { return &v }
