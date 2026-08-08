ALTER TABLE monitors
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS health_profile_id;

DROP INDEX IF EXISTS monitors_resource_type_idx;

ALTER TABLE monitors
    ALTER COLUMN type SET NOT NULL;
