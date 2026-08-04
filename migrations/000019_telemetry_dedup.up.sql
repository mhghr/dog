-- Telemetry event deduplication table
CREATE TABLE IF NOT EXISTS telemetry_event_dedup (
    event_id     UUID PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_dedup_processed_at ON telemetry_event_dedup (processed_at);

-- Telemetry dead-letter queue events
CREATE TABLE IF NOT EXISTS telemetry_dlq_events (
    id              SERIAL PRIMARY KEY,
    event_id        UUID NOT NULL,
    type            TEXT NOT NULL,
    error_reason    TEXT,
    retry_count     INT,
    first_failed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_failed_at  TIMESTAMPTZ,
    payload         JSONB
);
CREATE INDEX IF NOT EXISTS idx_dlq_event_id ON telemetry_dlq_events (event_id);

-- Add event_id to probe_results for cross-reference
ALTER TABLE probe_results ADD COLUMN IF NOT EXISTS event_id UUID;
CREATE UNIQUE INDEX IF NOT EXISTS idx_probe_results_event_id ON probe_results (event_id) WHERE event_id IS NOT NULL;
