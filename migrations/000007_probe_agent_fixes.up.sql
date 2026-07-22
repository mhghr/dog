-- 000007_probe_agent_fixes.up.sql
-- Fixes migration order issue: the unique index on attempt column
-- was created before the column itself in 000006.
-- Also adds agent_gateway_cert column for certificate delivery.

ALTER TABLE probe_agents
    ADD COLUMN IF NOT EXISTS agent_gateway_cert TEXT NOT NULL DEFAULT '';

ALTER TABLE probe_agents
    ADD COLUMN IF NOT EXISTS running_jobs INTEGER NOT NULL DEFAULT 0;

ALTER TABLE probe_agents
    ADD COLUMN IF NOT EXISTS spool_bytes BIGINT NOT NULL DEFAULT 0;

-- Recreate the unique index if it was created incorrectly in 000006.
-- Drop the old index and create a proper one.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE indexname = 'probe_results_job_location_attempt_idx'
    ) THEN
        DROP INDEX probe_results_job_location_attempt_idx;
    END IF;
END
$$;

-- Ensure attempt column exists before creating index.
-- ALTER TABLE with IF NOT EXISTS is safe for idempotent runs.
ALTER TABLE probe_results
    ADD COLUMN IF NOT EXISTS attempt INTEGER NOT NULL DEFAULT 1;

ALTER TABLE probe_results
    ADD COLUMN IF NOT EXISTS agent_id UUID REFERENCES probe_agents(id);

-- Now create the proper unique index (attempt column must exist first).
CREATE UNIQUE INDEX probe_results_job_location_attempt_idx
    ON probe_results(
        job_id,
        COALESCE(probe_location_id, '00000000-0000-0000-0000-000000000000'::uuid),
        attempt
    );
