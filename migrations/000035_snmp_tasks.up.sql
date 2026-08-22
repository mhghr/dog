-- 000035_snmp_tasks: on-demand SNMP operations (test connection / discovery)
-- executed by the SNMP collector. The API creates a task, the collector runs
-- it (via the worker in NATS mode, inline otherwise), and the result is
-- stored here for the frontend to poll.

CREATE TABLE snmp_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID REFERENCES workspaces(id) ON DELETE CASCADE,
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    monitor_id UUID REFERENCES monitors(id) ON DELETE CASCADE,
    kind VARCHAR(20) NOT NULL CHECK (kind IN ('test', 'discovery')),
    config JSONB NOT NULL DEFAULT '{}'::jsonb,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'success', 'failed')),
    result JSONB,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    finished_at TIMESTAMPTZ
);

CREATE INDEX snmp_tasks_resource_idx ON snmp_tasks(resource_id, created_at DESC);
CREATE INDEX snmp_tasks_monitor_idx ON snmp_tasks(monitor_id, created_at DESC);
