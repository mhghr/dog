-- 000034_snmp_monitoring: per-monitor SNMP discovery cache, interface
-- monitoring policy, and the SNMP event stream (traps + poll state changes).

-- 1. Discovery cache — refreshed on enrollment, config change, or manual
--    rediscovery; never on routine polls. Holds the device identity,
--    interface table and environmental sensors as normalized JSON.
CREATE TABLE snmp_discovery (
    monitor_id UUID PRIMARY KEY REFERENCES monitors(id) ON DELETE CASCADE,
    device JSONB NOT NULL DEFAULT '{}'::jsonb,
    interfaces JSONB NOT NULL DEFAULT '[]'::jsonb,
    sensors JSONB NOT NULL DEFAULT '[]'::jsonb,
    discovered_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 2. Per-interface monitoring policy + last known state. The policy columns
--    control which interfaces are monitored, ignored, renamed, or have custom
--    utilization thresholds. The last_* columns cache the latest poll so the
--    device detail page renders without re-reading every probe result.
CREATE TABLE snmp_interfaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    if_index INTEGER NOT NULL,
    if_name TEXT NOT NULL DEFAULT '',
    if_descr TEXT NOT NULL DEFAULT '',
    if_alias TEXT NOT NULL DEFAULT '',
    display_name TEXT NOT NULL DEFAULT '',
    ignore BOOLEAN NOT NULL DEFAULT FALSE,
    monitor BOOLEAN NOT NULL DEFAULT TRUE,
    utilization_warning DOUBLE PRECISION,
    utilization_critical DOUBLE PRECISION,
    oper_down_critical BOOLEAN NOT NULL DEFAULT TRUE,
    last_oper_status INTEGER,
    last_in_bps DOUBLE PRECISION,
    last_out_bps DOUBLE PRECISION,
    last_utilization_percent DOUBLE PRECISION,
    last_check_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (monitor_id, if_index)
);

CREATE INDEX snmp_interfaces_monitor_idx ON snmp_interfaces(monitor_id);

-- 3. SNMP event stream — normalized traps and poll-derived state changes,
--    bound to the resource (and interface when relevant).
CREATE TABLE snmp_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    monitor_id UUID REFERENCES monitors(id) ON DELETE CASCADE,
    probe_id UUID,
    kind VARCHAR(20) NOT NULL DEFAULT 'trap',
    event_type VARCHAR(80) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    source TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    interface_id UUID,
    if_index INTEGER,
    if_name TEXT,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX snmp_events_resource_time_idx ON snmp_events(resource_id, created_at DESC);
CREATE INDEX snmp_events_monitor_time_idx ON snmp_events(monitor_id, created_at DESC);
CREATE INDEX snmp_events_type_idx ON snmp_events(event_type);
