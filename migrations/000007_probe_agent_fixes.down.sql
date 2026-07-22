-- 000007_probe_agent_fixes.down.sql

DROP INDEX IF EXISTS probe_results_job_location_attempt_idx;

ALTER TABLE probe_agents DROP COLUMN IF EXISTS agent_gateway_cert;
ALTER TABLE probe_agents DROP COLUMN IF EXISTS running_jobs;
ALTER TABLE probe_agents DROP COLUMN IF EXISTS spool_bytes;

-- Restore old single-column index.
CREATE UNIQUE INDEX IF NOT EXISTS probe_results_job_id_idx ON probe_results(job_id);
