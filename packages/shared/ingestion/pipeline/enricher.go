package pipeline

import "monitoring-platform/packages/shared/domain"

// MetricEnricher attaches tenant and host identity to metric samples.
type MetricEnricher struct{}

// NewMetricEnricher creates a metric enricher.
func NewMetricEnricher() *MetricEnricher {
	return &MetricEnricher{}
}

// Enrich adds tenant_id, agent_id, and hostname labels and fields to every sample.
func (e *MetricEnricher) Enrich(batch *domain.MetricBatch, identity *AgentIdentity) {
	for i := range batch.Samples {
		s := &batch.Samples[i]
		if s.Labels == nil {
			s.Labels = make(map[string]string)
		}
		s.Labels["tenant_id"] = identity.TenantID
		s.Labels["agent_id"] = identity.AgentID
		if identity.Hostname != "" {
			s.Labels["hostname"] = identity.Hostname
		}

		s.TenantID = identity.TenantID
		s.AgentID = identity.AgentID
		s.Hostname = identity.Hostname
	}
	batch.TenantID = identity.TenantID
	batch.AgentID = identity.AgentID
}
