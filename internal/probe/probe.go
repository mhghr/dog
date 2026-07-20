package probe

import (
	"context"
	"log/slog"

	"monitoring-platform/internal/domain"
	"monitoring-platform/internal/security"
)

// Executor runs a single probe attempt for one monitor type.
type Executor interface {
	Type() domain.MonitorType
	Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult
}

// Deps carries shared infrastructure into executors.
type Deps struct {
	Guard          *security.Guard
	Logger         *slog.Logger
	PingPrivileged bool
}

type Registry struct {
	executors map[domain.MonitorType]Executor
}

func NewRegistry(executors ...Executor) *Registry {
	registry := &Registry{
		executors: make(map[domain.MonitorType]Executor),
	}

	for _, executor := range executors {
		registry.executors[executor.Type()] = executor
	}

	return registry
}

func (r *Registry) Get(monitorType domain.MonitorType) (Executor, bool) {
	executor, ok := r.executors[monitorType]
	return executor, ok
}

// DefaultRegistry wires all eight phase-one executors.
func DefaultRegistry(deps Deps) *Registry {
	return NewRegistry(
		NewHTTPExecutor(deps),
		NewTCPExecutor(deps),
		NewDNSExecutor(deps),
		NewPingExecutor(deps),
		NewTLSExecutor(deps),
		NewDomainExpirationExecutor(deps),
		NewSMTPExecutor(deps),
		NewNTPExecutor(deps),
	)
}
