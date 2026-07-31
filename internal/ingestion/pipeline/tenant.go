package pipeline

import (
	"context"
	"fmt"
	"sync"
)

// TenantResolver maps agent IDs to tenant IDs with a cached lookup.
type TenantResolver struct {
	mu      sync.RWMutex
	cache   map[string]string
	resolve func(ctx context.Context, agentID string) (string, error)
}

// NewTenantResolver creates a resolver backed by the provided resolve function.
func NewTenantResolver(resolveFn func(ctx context.Context, agentID string) (string, error)) *TenantResolver {
	return &TenantResolver{
		cache:   make(map[string]string),
		resolve: resolveFn,
	}
}

// Resolve returns the tenant ID for an agent, caching the result.
func (r *TenantResolver) Resolve(ctx context.Context, agentID string) (string, error) {
	r.mu.RLock()
	if tid, ok := r.cache[agentID]; ok {
		r.mu.RUnlock()
		return tid, nil
	}
	r.mu.RUnlock()

	tid, err := r.resolve(ctx, agentID)
	if err != nil {
		return "", fmt.Errorf("resolve tenant: %w", err)
	}

	r.mu.Lock()
	r.cache[agentID] = tid
	r.mu.Unlock()
	return tid, nil
}

// Invalidate removes a cached mapping (e.g. after agent re-registration).
func (r *TenantResolver) Invalidate(agentID string) {
	r.mu.Lock()
	delete(r.cache, agentID)
	r.mu.Unlock()
}
