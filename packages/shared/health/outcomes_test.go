package health

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"monitoring-platform/packages/shared/domain"
)

// recordingHealthRepo persists health states in memory so transition outcomes
// can be observed across evaluations.
type recordingHealthRepo struct {
	stubHealthRepo
	states map[string]ParameterHealthState
}

func newRecordingHealthRepo() *recordingHealthRepo {
	return &recordingHealthRepo{states: map[string]ParameterHealthState{}}
}

func (r *recordingHealthRepo) GetHealthState(ctx context.Context, monitorID, parameterKey string) (ParameterHealthState, error) {
	if s, ok := r.states[monitorID+":"+parameterKey]; ok {
		return s, nil
	}
	return ParameterHealthState{}, domain.ErrNotFound
}

func (r *recordingHealthRepo) UpsertHealthState(ctx context.Context, state *ParameterHealthState) error {
	r.states[state.MonitorID+":"+state.ParameterKey] = *state
	return nil
}

func TestEvaluateParameterEmitsOutcomeOnTransition(t *testing.T) {
	repo := newRecordingHealthRepo()
	engine := NewEngine(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	def := httpDef("http.total_duration_ms")
	rule := defaultRuleFromCatalog(def, "m1")
	ctx := context.Background()

	state, outcome := engine.EvaluateParameter(ctx, "m1", "http.total_duration_ms", []float64{10000}, &rule, def)
	if state != HealthError {
		t.Fatalf("first evaluation should be HealthError, got %s", state)
	}
	if outcome == nil {
		t.Fatal("expected an outcome on the first transition")
	}
	if outcome.OldState != HealthUnknown || outcome.NewState != HealthError {
		t.Fatalf("unexpected outcome: old=%s new=%s", outcome.OldState, outcome.NewState)
	}

	state, outcome = engine.EvaluateParameter(ctx, "m1", "http.total_duration_ms", []float64{10000}, &rule, def)
	if state != HealthError {
		t.Fatalf("stable evaluation should stay HealthError, got %s", state)
	}
	if outcome != nil {
		t.Fatal("expected no outcome on a stable state")
	}

	state, outcome = engine.EvaluateParameter(ctx, "m1", "http.total_duration_ms", []float64{1000}, &rule, def)
	if state != HealthOK {
		t.Fatalf("recovery should be HealthOK, got %s", state)
	}
	if outcome == nil {
		t.Fatal("expected an outcome on recovery")
	}
	if outcome.OldState != HealthError || outcome.NewState != HealthOK {
		t.Fatalf("unexpected outcome: old=%s new=%s", outcome.OldState, outcome.NewState)
	}
}

func TestEvaluateResultReturnsOutcomesForTransitionedParameters(t *testing.T) {
	repo := newRecordingHealthRepo()
	engine := NewEngine(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))

	result := &domain.ProbeResult{
		MonitorID: "m1",
		Attributes: map[string]any{"monitor_type": "http"},
		Metrics: map[string]any{
			"http.total_duration_ms": 10000.0,
		},
	}

	outcomes, err := engine.EvaluateResult(context.Background(), result)
	if err != nil {
		t.Fatalf("EvaluateResult failed: %v", err)
	}
	if len(outcomes) == 0 {
		t.Fatal("expected at least one outcome for a transitioned parameter")
	}

	found := false
	for _, o := range outcomes {
		if o.ParameterKey == "http.total_duration_ms" && o.NewState == HealthError {
			found = true
		}
	}
	if !found {
		t.Fatal("expected an outcome for http.total_duration_ms -> HealthError")
	}

	// Re-evaluating the same result must not emit outcomes (stable state).
	outcomes, err = engine.EvaluateResult(context.Background(), result)
	if err != nil {
		t.Fatalf("EvaluateResult failed: %v", err)
	}
	for _, o := range outcomes {
		if o.ParameterKey == "http.total_duration_ms" {
			t.Fatalf("expected no outcome for stable parameter, got %+v", o)
		}
	}
}
