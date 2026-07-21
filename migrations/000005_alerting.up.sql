-- Alerting Phase 3 — per Sections 126-149 of monitoring.md.
-- Core tables: alert_policies (rule definitions), notification_channels
-- (where alerts go), and incidents (stateful event tracking).

CREATE TABLE notification_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('email', 'webhook', 'telegram')),
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX notification_channels_org_idx ON notification_channels(organization_id);

CREATE TABLE alert_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    scope JSONB NOT NULL DEFAULT '{}',        -- {monitor_ids: [], tags: {key:val}}
    conditions JSONB NOT NULL DEFAULT '{}',    -- {consecutive_failures:3, high_latency_ms:2000, ...}
    severity VARCHAR(50) NOT NULL DEFAULT 'warning' CHECK (severity IN ('info', 'warning', 'critical')),
    opening_failures INT NOT NULL DEFAULT 3,   -- anti-flapping: open after N failures
    resolving_successes INT NOT NULL DEFAULT 2, -- anti-flapping: resolve after N successes
    cooldown_seconds INT NOT NULL DEFAULT 300,
    renotify_seconds INT NOT NULL DEFAULT 0,   -- 0 = disabled
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    channel_ids UUID[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX alert_policies_org_idx ON alert_policies(organization_id);

CREATE TABLE alert_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    policy_id UUID NOT NULL REFERENCES alert_policies(id) ON DELETE CASCADE,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    state VARCHAR(50) NOT NULL CHECK (state IN ('pending', 'firing', 'recovering', 'resolved', 'suppressed')),
    severity VARCHAR(50) NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    consecutive_failures INT NOT NULL DEFAULT 0,
    consecutive_successes INT NOT NULL DEFAULT 0,
    opened_at TIMESTAMPTZ,
    resolved_at TIMESTAMPTZ,
    last_notified_at TIMESTAMPTZ,
    notification_count INT NOT NULL DEFAULT 0,
    dedup_key VARCHAR(100) NOT NULL,  -- unique: policy_id + monitor_id
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (dedup_key)
);

CREATE INDEX alert_events_org_idx ON alert_events(organization_id);
CREATE INDEX alert_events_policy_idx ON alert_events(policy_id);
CREATE INDEX alert_events_monitor_idx ON alert_events(monitor_id);
CREATE INDEX alert_events_state_idx ON alert_events(state);
