-- Supports latest-per-probe lookups and the bounded series selection used by
-- resource charts. The existing monitor/time index cannot satisfy both the
-- DISTINCT ON probe location ordering and the monitor range predicate.
CREATE INDEX IF NOT EXISTS probe_results_monitor_probe_time_idx
    ON probe_results (monitor_id, probe_location_id, started_at DESC);
