package alerting

import (
	"context"
	"testing"

	"monitoring-platform/packages/shared/domain"
)

// fakeAlertRepo is a minimal in-memory AlertRepository for testing the status
// down/up transition. It implements the methods the Engine calls; ListFiring
// and RecordNotification are unused stubs required to satisfy the interface.
type fakeAlertRepo struct {
	alerts map[string]domain.Alert
}

func (f *fakeAlertRepo) ListActivePolicies(ctx context.Context, monitorID string) ([]domain.AlertPolicy, error) {
	return []domain.AlertPolicy{{
		ID:                 "policy-1",
		OrganizationID:     "org-1",
		Name:               "Ping Down",
		Severity:           "critical",
		OpeningFailures:    1,
		ResolvingSuccesses: 1,
	}}, nil
}

func (f *fakeAlertRepo) FindByDedup(ctx context.Context, dedupKey string) (domain.Alert, error) {
	if a, ok := f.alerts[dedupKey]; ok {
		return a, nil
	}
	return domain.Alert{}, domain.ErrNotFound
}

func (f *fakeAlertRepo) UpsertAlert(ctx context.Context, alert *domain.Alert) error {
	if f.alerts == nil {
		f.alerts = map[string]domain.Alert{}
	}
	f.alerts[alert.DedupKey] = *alert
	return nil
}

func (f *fakeAlertRepo) ListFiring(ctx context.Context) ([]domain.Alert, error) {
	return nil, nil
}

func (f *fakeAlertRepo) RecordNotification(ctx context.Context, alertID string) error {
	return nil
}

func TestAlertEngineFiresOnPingDown(t *testing.T) {
	repo := &fakeAlertRepo{}
	engine := NewEngine(repo, nil)

	result := domain.ProbeResult{
		ID: "r1", MonitorID: "m1", MonitorName: "server1",
		Status: domain.StatusDown, Success: false,
	}

	events := engine.Evaluate(context.Background(), result)
	if len(events) != 1 {
		t.Fatalf("expected 1 firing event on down, got %d", len(events))
	}
	if events[0].NewState != "firing" {
		t.Fatalf("expected firing state, got %s", events[0].NewState)
	}
}

func TestAlertEngineResolvesOnPingUp(t *testing.T) {
	repo := &fakeAlertRepo{}
	engine := NewEngine(repo, nil)

	down := domain.ProbeResult{ID: "r1", MonitorID: "m1", MonitorName: "server1", Status: domain.StatusDown, Success: false}
	engine.Evaluate(context.Background(), down)

	up := domain.ProbeResult{ID: "r2", MonitorID: "m1", MonitorName: "server1", Status: domain.StatusUp, Success: true}
	events := engine.Evaluate(context.Background(), up)
	if len(events) != 1 || events[0].NewState != "recovering" {
		t.Fatalf("expected recovering event on up, got %+v", events)
	}
}
