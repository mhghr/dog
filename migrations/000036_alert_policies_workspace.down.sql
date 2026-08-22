-- 000036_alert_policies_workspace down: no-op. On a correctly migrated
-- database the up migration is a no-op (workspace_id came from 000020), so
-- dropping the column here would undo 000020's schema.

SELECT 1;
