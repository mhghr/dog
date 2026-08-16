-- 000027_resource_types_repair (down): revert the repair columns.

ALTER TABLE resource_types DROP COLUMN IF EXISTS configuration_schema;
ALTER TABLE resource_types DROP COLUMN IF EXISTS slug;
