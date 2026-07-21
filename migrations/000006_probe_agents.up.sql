-- 000006_probe_agents.up.sql
-- Phase 1: Probe Agent enrollment, identity, and lifecycle management.
-- Section 234/235 — Production-grade Probe Agent architecture.

CREATE TYPE probe_agent_status AS ENUM (
    'pending',
    'approved',
    'active',
    'offline',
    'disabled',
    'rejected',
    'revoked',
    'draining',
    'updating'
);

CREATE TABLE probe_agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    location_id UUID REFERENCES probe_locations(id),
    name VARCHAR(100) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    machine_fingerprint VARCHAR(255) NOT NULL UNIQUE,
    public_key TEXT NOT NULL,
    certificate_serial VARCHAR(255),
    version VARCHAR(50) NOT NULL,
    operating_system VARCHAR(50) NOT NULL DEFAULT '',
    architecture VARCHAR(50) NOT NULL DEFAULT '',
    public_ip INET,
    private_ips INET[] NOT NULL DEFAULT '{}',
    capabilities TEXT[] NOT NULL DEFAULT '{}',
    max_concurrency INTEGER NOT NULL DEFAULT 50,
    status probe_agent_status NOT NULL DEFAULT 'pending',
    approved_by UUID REFERENCES users(id),
    approved_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX probe_agents_location_status_idx
    ON probe_agents(location_id, status);

CREATE INDEX probe_agents_last_seen_idx
    ON probe_agents(last_seen_at);

CREATE INDEX probe_agents_status_idx
    ON probe_agents(status);

CREATE TABLE probe_agent_enrollment_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash BYTEA NOT NULL UNIQUE,
    requested_location_id UUID REFERENCES probe_locations(id),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE probe_agent_audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES probe_agents(id) ON DELETE CASCADE,
    actor_user_id UUID REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    previous_state JSONB,
    next_state JSONB,
    remote_ip INET,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX probe_agent_audit_agent_idx
    ON probe_agent_audit_log(agent_id, created_at DESC);

-- Fix idempotency: unique constraint on job_id + probe_location_id + attempt.
-- Drop the old unique index that only covered job_id.
DROP INDEX IF EXISTS probe_results_job_id_idx;
CREATE UNIQUE INDEX probe_results_job_location_attempt_idx
    ON probe_results(job_id, probe_location_id, COALESCE(attributes->>'attempt', '1'));

-- Add attempt column to probe_results for multi-attempt tracking.
ALTER TABLE probe_results
    ADD COLUMN IF NOT EXISTS attempt INTEGER NOT NULL DEFAULT 1;

ALTER TABLE probe_results
    ADD COLUMN IF NOT EXISTS agent_id UUID REFERENCES probe_agents(id);

-- Add next_run_at index for scheduler (was using monitors_next_run_idx already).
-- Ensure the existing index covers the scheduler query pattern.
