-- 000025_health_audit: Resource-level health engine, alert enrichment,
-- and comprehensive audit logging.

-- 1. Resource health state — roll-up of all monitors under a resource

CREATE TYPE resource_health AS ENUM ('healthy', 'degraded', 'warning', 'critical', 'down', 'unknown');

CREATE TABLE resource_health_state (
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    state resource_health NOT NULL DEFAULT 'unknown',
    score DOUBLE PRECISION NOT NULL DEFAULT 0,
    active_alerts INTEGER NOT NULL DEFAULT 0,
    active_warnings INTEGER NOT NULL DEFAULT 0,
    last_evaluated_at TIMESTAMPTZ,
    state_changed_at TIMESTAMPTZ,
    PRIMARY KEY (resource_id)
);

-- 2. Extend alert_events with workspace scoping and richer metadata

ALTER TABLE alert_events ADD COLUMN workspace_id UUID REFERENCES workspaces(id);
ALTER TABLE alert_events ADD COLUMN resource_id UUID REFERENCES resources(id);
ALTER TABLE alert_events ADD COLUMN acknowledged_at TIMESTAMPTZ;
ALTER TABLE alert_events ADD COLUMN acknowledged_by UUID REFERENCES users(id);
ALTER TABLE alert_events ADD COLUMN metadata JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX alert_events_workspace_idx ON alert_events(workspace_id);
CREATE INDEX alert_events_resource_idx ON alert_events(resource_id);

-- 3. Notification policies per workspace

ALTER TABLE notification_policies ADD COLUMN workspace_id UUID REFERENCES workspaces(id);
CREATE INDEX notification_policies_workspace_idx ON notification_policies(workspace_id);

-- 4. Comprehensive audit log

CREATE TYPE audit_action AS ENUM (
    'resource.created', 'resource.updated', 'resource.deleted', 'resource.status_changed',
    'monitor.created', 'monitor.updated', 'monitor.deleted', 'monitor.enabled', 'monitor.disabled',
    'monitor.status_changed',
    'workspace.created', 'workspace.updated', 'workspace.deleted',
    'member.invited', 'member.removed', 'member.role_changed',
    'alert.created', 'alert.acknowledged', 'alert.resolved',
    'agent.enrolled', 'agent.approved', 'agent.revoked', 'agent.status_changed',
    'probe.enrolled', 'probe.updated', 'probe.status_changed',
    'plugin.enabled', 'plugin.disabled',
    'credential.created', 'credential.updated', 'credential.deleted',
    'snmp.device_added', 'snmp.device_removed',
    'config.changed', 'settings.updated',
    'user.login', 'user.logout', 'user.password_changed',
    'token.created', 'token.revoked'
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES workspaces(id),
    actor_user_id UUID REFERENCES users(id),
    actor_agent_id UUID REFERENCES probe_agents(id),
    action audit_action NOT NULL,
    resource_type VARCHAR(100) NOT NULL DEFAULT '',
    resource_id UUID,
    details JSONB NOT NULL DEFAULT '{}'::jsonb,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX audit_logs_org_idx ON audit_logs(organization_id, created_at DESC);
CREATE INDEX audit_logs_actor_idx ON audit_logs(actor_user_id, created_at DESC);
CREATE INDEX audit_logs_workspace_idx ON audit_logs(workspace_id, created_at DESC);
CREATE INDEX audit_logs_resource_idx ON audit_logs(resource_id, created_at DESC);
CREATE INDEX audit_logs_action_idx ON audit_logs(action, created_at DESC);

-- 5. Seed workspace_id on existing alert_events from alert_policies

UPDATE alert_events ae
SET workspace_id = ap.workspace_id
FROM alert_policies ap
WHERE ae.policy_id = ap.id
  AND ap.workspace_id IS NOT NULL;
