CREATE TABLE monitoring_bootstrap_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    token_hash      TEXT NOT NULL UNIQUE,
    description     TEXT NOT NULL DEFAULT '',
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    created_by      UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mon_bootstrap_tokens_lookup ON monitoring_bootstrap_tokens(token_hash)
    WHERE used_at IS NULL AND revoked_at IS NULL AND expires_at > NOW();

CREATE TABLE monitoring_agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    external_id     TEXT NOT NULL UNIQUE,
    hostname        TEXT NOT NULL,
    os              TEXT NOT NULL,
    arch            TEXT NOT NULL,
    version         TEXT NOT NULL,
    agent_id        TEXT NOT NULL UNIQUE,
    secret_hash     TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'inactive', 'draining', 'removed')),
    last_seen_at    TIMESTAMPTZ,
    registered_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    labels          JSONB NOT NULL DEFAULT '{}'::jsonb,
    capabilities    TEXT[] NOT NULL DEFAULT '{}',
    private_ips     TEXT[] NOT NULL DEFAULT '{}',
    bootstrap_token_id UUID REFERENCES monitoring_bootstrap_tokens(id)
);

CREATE INDEX idx_mon_agents_tenant ON monitoring_agents(tenant_id);
CREATE INDEX idx_mon_agents_agent_id ON monitoring_agents(agent_id);
CREATE INDEX idx_mon_agents_status ON monitoring_agents(status);
CREATE INDEX idx_mon_agents_last_seen ON monitoring_agents(last_seen_at) WHERE status = 'active';

CREATE TABLE monitoring_agent_heartbeats (
    id                  BIGSERIAL,
    agent_id            TEXT NOT NULL REFERENCES monitoring_agents(agent_id) ON DELETE CASCADE,
    tenant_id           UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    cpu_percent         DOUBLE PRECISION,
    memory_percent      DOUBLE PRECISION,
    disk_percent        DOUBLE PRECISION,
    uptime_seconds      BIGINT,
    metrics_sent        BIGINT DEFAULT 0,
    metrics_queued      BIGINT DEFAULT 0,
    collector_uptime_seconds BIGINT DEFAULT 0,
    public_ip           TEXT,
    recorded_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (recorded_at);

CREATE TABLE monitoring_agent_heartbeats_default
    PARTITION OF monitoring_agent_heartbeats
    FOR VALUES FROM ('2026-01-01') TO ('2027-01-01');

CREATE INDEX idx_mon_heartbeats_agent_time ON monitoring_agent_heartbeats(agent_id, recorded_at DESC);
CREATE INDEX idx_mon_heartbeats_tenant_time ON monitoring_agent_heartbeats(tenant_id, recorded_at DESC);

CREATE TABLE monitoring_agent_configs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        TEXT NOT NULL REFERENCES monitoring_agents(agent_id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    version         INTEGER NOT NULL,
    config_json     JSONB NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(agent_id, version)
);

CREATE INDEX idx_mon_configs_active ON monitoring_agent_configs(agent_id) WHERE is_active = TRUE;

CREATE TABLE monitoring_agent_updates (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id        TEXT NOT NULL REFERENCES monitoring_agents(agent_id) ON DELETE CASCADE,
    from_version    TEXT NOT NULL,
    to_version      TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending'
                    CHECK (status IN ('pending', 'downloading', 'applying', 'success', 'rollback', 'failed')),
    error_message   TEXT,
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE OR REPLACE FUNCTION update_mon_agents_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_mon_agents_updated_at
    BEFORE UPDATE ON monitoring_agents
    FOR EACH ROW EXECUTE FUNCTION update_mon_agents_updated_at();
