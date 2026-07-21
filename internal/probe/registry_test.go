package probe

import (
	"context"
	"testing"

	"monitoring-platform/internal/domain"
)

type stubExecutor struct{ monitorType domain.MonitorType }

func (s stubExecutor) Type() domain.MonitorType { return s.monitorType }
func (s stubExecutor) Execute(context.Context, domain.ProbeJob) domain.ProbeResult {
	return domain.ProbeResult{}
}

func TestRegistryRejectsDuplicateExecutors(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate executor registration to panic")
		}
	}()

	NewRegistry(stubExecutor{domain.MonitorHTTP}, stubExecutor{domain.MonitorHTTP})
}

func TestDefaultRegistryCoversEveryMonitorType(t *testing.T) {
	t.Parallel()

	registry := DefaultRegistry(Deps{})
	if got, want := len(registry.Types()), len(domain.AllMonitorTypes); got != want {
		t.Fatalf("registered executor count = %d, want %d", got, want)
	}

	for _, monitorType := range domain.AllMonitorTypes {
		if _, ok := registry.Get(monitorType); !ok {
			t.Errorf("monitor type %q has no registered executor", monitorType)
		}
	}
}
