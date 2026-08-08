-- 000026_monitors_unify: Finish evolving the single monitors table into a
-- resource-aware monitoring engine. Relaxes the legacy type enum so monitors
-- are driven by monitor_type_id from the registry.

ALTER TABLE monitors
    ALTER COLUMN type DROP NOT NULL;

ALTER TABLE monitors
    ADD COLUMN IF NOT EXISTS health_profile_id UUID,
    ADD COLUMN IF NOT EXISTS created_by UUID REFERENCES users(id);

CREATE INDEX IF NOT EXISTS monitors_resource_type_idx
    ON monitors(resource_id, monitor_type_id);
