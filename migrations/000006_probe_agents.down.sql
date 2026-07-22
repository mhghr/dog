-- 000006_probe_agents.down.sql

ALTER TABLE probe_results DROP CONSTRAINT IF EXISTS probe_results_agent_id_fkey;
DROP INDEX IF EXISTS probe_results_job_location_attempt_idx;
ALTER TABLE probe_results DROP COLUMN IF EXISTS attempt;
ALTER TABLE probe_results DROP COLUMN IF EXISTS agent_id;
CREATE UNIQUE INDEX probe_results_job_id_idx ON probe_results(job_id);

DROP INDEX IF EXISTS probe_agent_audit_agent_idx;
DROP TABLE IF EXISTS probe_agent_audit_log;
DROP TABLE IF EXISTS probe_agent_enrollment_tokens;
DROP INDEX IF EXISTS probe_agents_status_idx;
DROP INDEX IF EXISTS probe_agents_last_seen_idx;
DROP INDEX IF EXISTS probe_agents_location_status_idx;
DROP TABLE IF EXISTS probe_agents;
DROP TYPE IF EXISTS probe_agent_status;
