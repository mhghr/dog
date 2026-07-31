package pipeline

import "monitoring-platform/internal/domain"

// MetricNormalizer normalizes metric and label names in a batch.
type MetricNormalizer struct{}

// NewMetricNormalizer creates a metric normalizer.
func NewMetricNormalizer() *MetricNormalizer {
	return &MetricNormalizer{}
}

// Normalize lowercases and sanitizes metric and label names in place.
func (n *MetricNormalizer) Normalize(batch *domain.MetricBatch) {
	for i := range batch.Samples {
		s := &batch.Samples[i]
		s.Name = NormalizeMetricName(s.Name)

		normalized := make(map[string]string, len(s.Labels))
		for k, v := range s.Labels {
			normalized[NormalizeMetricName(k)] = v
		}
		s.Labels = normalized
	}
}
