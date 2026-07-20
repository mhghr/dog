CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TYPE monitor_type AS ENUM (
    'http',
    'tcp',
    'dns',
    'ping',
    'tls',
    'domain_expiration',
    'smtp',
    'ntp'
);

CREATE TYPE monitor_status AS ENUM (
    'up',
    'down',
    'unknown',
    'paused'
);

CREATE TABLE probe_locations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    code VARCHAR(50) NOT NULL UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE monitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    type monitor_type NOT NULL,
    target TEXT NOT NULL,
    interval_seconds INTEGER NOT NULL CHECK (interval_seconds >= 10),
    timeout_millis INTEGER NOT NULL DEFAULT 5000
        CHECK (timeout_millis BETWEEN 100 AND 60000),
    retries INTEGER NOT NULL DEFAULT 1
        CHECK (retries BETWEEN 0 AND 5),
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_status monitor_status NOT NULL DEFAULT 'unknown',
    last_checked_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX monitors_next_run_idx
    ON monitors(next_run_at)
    WHERE enabled = TRUE;

CREATE INDEX monitors_type_idx ON monitors(type);
CREATE INDEX monitors_last_status_idx ON monitors(last_status);

CREATE TABLE probe_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id UUID NOT NULL,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    probe_location_id UUID REFERENCES probe_locations(id),
    status monitor_status NOT NULL,
    success BOOLEAN NOT NULL,
    error_code VARCHAR(100),
    error_message TEXT,
    duration_millis BIGINT NOT NULL,
    metrics JSONB NOT NULL DEFAULT '{}'::jsonb,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX probe_results_job_id_idx
    ON probe_results(job_id);

CREATE INDEX probe_results_monitor_time_idx
    ON probe_results(monitor_id, started_at DESC);

CREATE INDEX probe_results_started_at_idx
    ON probe_results(started_at);

INSERT INTO probe_locations(name, code)
VALUES ('Local Development', 'local-dev');
