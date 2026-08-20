package probe

import (
	"context"
	"log/slog"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/security"
)

// Executor runs a single probe attempt for one monitor type.
type Executor interface {
	Type() domain.MonitorType
	Execute(ctx context.Context, job domain.ProbeJob) domain.ProbeResult
}

// SecretResolver resolves secret references (`${secret:name}`) at execution
// time. Implementations back onto a vault/KMS/env provider so raw credentials
// never live in monitor configuration, logs, or result attributes. A nil
// resolver leaves references unresolved (an explicit failure) rather than
// silently emitting the reference itself.
type SecretResolver interface {
	// Resolve returns the secret value for name, or an error when it is
	// missing. Implementations must never log or return the raw value in
	// error messages.
	Resolve(ctx context.Context, name string) (string, error)
}

// Deps carries shared infrastructure into executors.
type Deps struct {
	Guard          *security.Guard
	Logger         *slog.Logger
	PingPrivileged bool
	Secrets        SecretResolver
}

type Registry struct {
	executors map[domain.MonitorType]Executor
}

func NewRegistry(executors ...Executor) *Registry {
	registry := &Registry{
		executors: make(map[domain.MonitorType]Executor),
	}

	for _, executor := range executors {
		if executor == nil {
			panic("probe: cannot register a nil executor")
		}
		if _, exists := registry.executors[executor.Type()]; exists {
			panic("probe: duplicate executor for monitor type " + string(executor.Type()))
		}
		registry.executors[executor.Type()] = executor
	}

	return registry
}

// Types returns the registered monitor types. The returned slice is detached
// from the registry and is intended for diagnostics and completeness tests.
func (r *Registry) Types() []domain.MonitorType {
	types := make([]domain.MonitorType, 0, len(r.executors))
	for monitorType := range r.executors {
		types = append(types, monitorType)
	}
	return types
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
