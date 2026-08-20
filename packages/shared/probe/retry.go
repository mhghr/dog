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
//
// Every attempt is tagged with its attempt number; the final result carries
// `attempts`, `max_attempts` and `first_attempt_failed` so consumers can
// distinguish a clean success from one that only recovered after retries
// (retries must never fabricate availability — the success flag still comes
// from the executor, and the attempt count is stored alongside it).
func ExecuteWithRetry(parentCtx context.Context, executor Executor, job domain.ProbeJob) domain.ProbeResult {
	attempts := job.Retries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastResult domain.ProbeResult
	var firstAttemptFailed bool

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
		// execution_id: the job ID is the stable identifier for this scheduled
		// run across retries, so ingestion can deduplicate retried results.
		lastResult.Attributes["execution_id"] = job.ID
		lastResult.Attributes["attempt"] = attempt
		lastResult.Attributes["max_attempts"] = attempts

		if lastResult.Success {
			lastResult.Attributes["attempts"] = attempt
			lastResult.Attributes["first_attempt_failed"] = firstAttemptFailed
			return lastResult
		}

		firstAttemptFailed = true
		lastResult.Attributes["attempts"] = attempt
		lastResult.Attributes["first_attempt_failed"] = firstAttemptFailed

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
