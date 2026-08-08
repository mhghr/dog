package probe

import (
	"context"
	"time"

	"monitoring-platform/packages/shared/domain"
)

const retryBackoff = 500 * time.Millisecond

// ExecuteWithRetry runs a probe up to retries+1 times with a per-attempt
// timeout. The first successful attempt wins; otherwise the last failure
// is returned with attempt bookkeeping attached.
func ExecuteWithRetry(parentCtx context.Context, executor Executor, job domain.ProbeJob) domain.ProbeResult {
	attempts := job.Retries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastResult domain.ProbeResult

	for attempt := 1; attempt <= attempts; attempt++ {
		ctx, cancel := context.WithTimeout(
			parentCtx,
			time.Duration(job.TimeoutMillis)*time.Millisecond,
		)

		lastResult = executor.Execute(ctx, job)
		cancel()

		if lastResult.Attributes == nil {
			lastResult.Attributes = map[string]any{}
		}
		lastResult.Attributes["attempt"] = attempt
		lastResult.Attributes["max_attempts"] = attempts

		if lastResult.Success {
			return lastResult
		}

		if attempt < attempts {
			select {
			case <-parentCtx.Done():
				return lastResult
			case <-time.After(retryBackoff):
			}
		}
	}

	return lastResult
}
