package probe

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"monitoring-platform/packages/shared/domain"
	"monitoring-platform/packages/shared/security"
)

func newBaseResult(job domain.ProbeJob) domain.ProbeResult {
	attrs := map[string]any{"monitor_type": string(job.Type)}
	if job.ResourceID != "" {
		attrs["resource_id"] = job.ResourceID
	}
	if job.WorkspaceID != "" {
		attrs["workspace_id"] = job.WorkspaceID
	}

	return domain.ProbeResult{
		ID:              uuid.NewString(),
		JobID:           job.ID,
		MonitorID:       job.MonitorID,
		ProbeLocationID: job.ProbeLocationID,
		Status:          domain.StatusDown,
		Success:         false,
		Metrics:         map[string]any{},
		Attributes:      attrs,
		StartedAt:       time.Now().UTC(),
	}
}

func finishFailure(result domain.ProbeResult, code string, err error) domain.ProbeResult {
	finishedAt := time.Now().UTC()

	result.Success = false
	result.Status = domain.StatusDown
	result.ErrorCode = mapErrorCode(code, err)
	if err != nil {
		result.ErrorMessage = err.Error()
	}
	result.FinishedAt = finishedAt
	result.DurationMillis = finishedAt.Sub(result.StartedAt).Milliseconds()
	result.Metrics["total_duration_ms"] = result.DurationMillis

	return result
}

func finishSuccess(result domain.ProbeResult) domain.ProbeResult {
	finishedAt := time.Now().UTC()

	result.Success = true
	result.Status = domain.StatusUp
	result.FinishedAt = finishedAt
	result.DurationMillis = finishedAt.Sub(result.StartedAt).Milliseconds()
	result.Metrics["total_duration_ms"] = result.DurationMillis

	return result
}

// mapErrorCode promotes transport-level failures to the standard error codes.
func mapErrorCode(code string, err error) string {
	if err == nil {
		return code
	}

	var blocked *security.BlockedTargetError
	if errors.As(err, &blocked) {
		return "blocked_target"
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout"
	}

	if strings.Contains(strings.ToLower(err.Error()), "permission denied") ||
		strings.Contains(strings.ToLower(err.Error()), "operation not permitted") {
		return "permission_denied"
	}

	return code
}

func stringConfig(config map[string]any, key, defaultValue string) string {
	value, ok := config[key]
	if !ok {
		return defaultValue
	}

	result, ok := value.(string)
	if !ok || result == "" {
		return defaultValue
	}

	return result
}

func boolConfig(config map[string]any, key string, defaultValue bool) bool {
	value, ok := config[key]
	if !ok {
		return defaultValue
	}

	result, ok := value.(bool)
	if !ok {
		return defaultValue
	}

	return result
}

func intConfig(config map[string]any, key string, defaultValue int) int {
	value, ok := config[key]
	if !ok {
		return defaultValue
	}

	switch number := value.(type) {
	case int:
		return number
	case int64:
		return int(number)
	case float64:
		return int(number)
	case string:
		parsed, err := strconv.Atoi(number)
		if err != nil {
			return defaultValue
		}
		return parsed
	default:
		return defaultValue
	}
}

func intSliceConfig(config map[string]any, key string, defaultValue []int) []int {
	value, ok := config[key]
	if !ok {
		return defaultValue
	}

	values, ok := value.([]any)
	if !ok {
		return defaultValue
	}

	result := make([]int, 0, len(values))
	for _, item := range values {
		switch number := item.(type) {
		case float64:
			result = append(result, int(number))
		case int:
			result = append(result, number)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}

	return result
}

func stringSliceConfig(config map[string]any, key string, defaultValue []string) []string {
	value, ok := config[key]
	if !ok {
		return defaultValue
	}

	values, ok := value.([]any)
	if !ok {
		return defaultValue
	}

	result := make([]string, 0, len(values))
	for _, item := range values {
		if text, ok := item.(string); ok {
			result = append(result, text)
		}
	}

	if len(result) == 0 {
		return defaultValue
	}

	return result
}

func containsInt(values []int, expected int) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}

	return false
}
