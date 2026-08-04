DROP INDEX IF EXISTS idx_probe_results_event_id;
ALTER TABLE probe_results DROP COLUMN IF EXISTS event_id;
DROP TABLE IF EXISTS telemetry_dlq_events;
DROP TABLE IF EXISTS telemetry_event_dedup;
DROP INDEX IF EXISTS idx_dedup_processed_at;
DROP INDEX IF EXISTS idx_dlq_event_id;
