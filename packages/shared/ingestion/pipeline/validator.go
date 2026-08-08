package pipeline

import (
	"fmt"
	"regexp"
	"strings"

	"monitoring-platform/packages/shared/domain"
)

var (
	validMetricName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_./-]*$`)
	validLabelName  = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)
	// reservedLabels cannot be set by agents; they are enforced by the enricher.
	reservedLabels = map[string]bool{
		"tenant_id": true,
		"agent_id":  true,
		"hostname":  true,
	}
)

// ValidationResult summarizes the outcome of validating a metric batch.
type ValidationResult struct {
	Valid   bool
	Skipped int
	Errors  []string
}

// ValidateMetrics validates metric names and label keys/values in a batch.
// Invalid samples are dropped from the batch in place.
func ValidateMetrics(batch *domain.MetricBatch) ValidationResult {
	result := ValidationResult{Valid: true}

	kept := batch.Samples[:0]
	for i, sample := range batch.Samples {
		if !validMetricName.MatchString(sample.Name) {
			result.Errors = append(result.Errors, fmt.Sprintf("sample[%d]: invalid metric name %q", i, sample.Name))
			result.Valid = false
			continue
		}
		if len(sample.Name) > 256 {
			result.Errors = append(result.Errors, fmt.Sprintf("sample[%d]: metric name too long", i))
			result.Valid = false
			continue
		}

		if sample.Labels == nil {
			sample.Labels = make(map[string]string)
		}

		invalid := false
		for k, v := range sample.Labels {
			if reservedLabels[k] {
				delete(sample.Labels, k)
				result.Skipped++
				continue
			}
			if !validLabelName.MatchString(k) {
				result.Errors = append(result.Errors, fmt.Sprintf("sample[%d]: invalid label name %q", i, k))
				delete(sample.Labels, k)
				result.Valid = false
				invalid = true
				continue
			}
			if len(k) > 256 {
				result.Errors = append(result.Errors, fmt.Sprintf("sample[%d]: label name too long", i))
				delete(sample.Labels, k)
				result.Valid = false
				invalid = true
				continue
			}
			if len(v) > 1024 {
				result.Errors = append(result.Errors, fmt.Sprintf("sample[%d]: label value too long, truncated", i))
				sample.Labels[k] = v[:1024]
			}
		}

		if invalid {
			result.Skipped++
			continue
		}
		kept = append(kept, sample)
	}

	batch.Samples = kept
	return result
}

// NormalizeMetricName lowercases, trims, and replaces spaces/dots with underscores.
func NormalizeMetricName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, ".", "_")
	return name
}
