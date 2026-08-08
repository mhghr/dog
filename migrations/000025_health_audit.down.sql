ALTER TABLE notification_policies DROP COLUMN workspace_id;

ALTER TABLE alert_events DROP COLUMN metadata;
ALTER TABLE alert_events DROP COLUMN acknowledged_by;
ALTER TABLE alert_events DROP COLUMN acknowledged_at;
ALTER TABLE alert_events DROP COLUMN resource_id;
ALTER TABLE alert_events DROP COLUMN workspace_id;

DROP TABLE IF EXISTS audit_logs;
DROP TYPE IF EXISTS audit_action;

DROP TABLE IF EXISTS resource_health_state;
DROP TYPE IF EXISTS resource_health;
