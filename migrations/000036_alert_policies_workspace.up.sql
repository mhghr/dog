-- 000036_alert_policies_workspace: Reconcile workspace scoping on
-- alert_policies. Migration 000020 adds workspace_id, but some databases
-- drifted and never landed the column while 000020 is already recorded as
-- applied. ADD COLUMN IF NOT EXISTS keeps this a no-op on correctly migrated
-- databases while repairing drifted ones.

ALTER TABLE alert_policies ADD COLUMN IF NOT EXISTS workspace_id UUID REFERENCES workspaces(id);
