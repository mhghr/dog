-- 000027_resource_types_repair: Repair the resource_types table for
-- environments where 000021_resources was applied against an older schema
-- variant (missing slug + configuration_schema, capabilities stored as jsonb
-- instead of text[]). Idempotent: safe to run on any environment.

-- 1. slug
ALTER TABLE resource_types ADD COLUMN IF NOT EXISTS slug VARCHAR(100);
UPDATE resource_types
SET slug = lower(regexp_replace(trim(name), '[^a-zA-Z0-9]+', '-', 'g'))
WHERE slug IS NULL OR slug = '';
ALTER TABLE resource_types ALTER COLUMN slug SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS resource_types_slug_idx ON resource_types(slug);

-- 2. configuration_schema
ALTER TABLE resource_types
    ADD COLUMN IF NOT EXISTS configuration_schema JSONB NOT NULL DEFAULT '{}'::jsonb;

-- 3. capabilities: jsonb -> text[] (only when it is still jsonb)
DO $$
DECLARE
    is_jsonb boolean;
BEGIN
    SELECT (data_type = 'jsonb') INTO is_jsonb
    FROM information_schema.columns
    WHERE table_name = 'resource_types' AND column_name = 'capabilities';

    IF is_jsonb THEN
        CREATE OR REPLACE FUNCTION pg_temp.jsonb_to_text_array(j jsonb) RETURNS text[]
        LANGUAGE sql IMMUTABLE AS
        $fn$ SELECT CASE WHEN jsonb_typeof(j) = 'array'
                         THEN ARRAY(SELECT jsonb_array_elements_text(j))
                         ELSE '{}'::text[] END $fn$;

        ALTER TABLE resource_types ALTER COLUMN capabilities DROP DEFAULT;
        ALTER TABLE resource_types ALTER COLUMN capabilities TYPE text[]
            USING pg_temp.jsonb_to_text_array(capabilities);
        ALTER TABLE resource_types ALTER COLUMN capabilities SET DEFAULT '{}'::text[];
        ALTER TABLE resource_types ALTER COLUMN capabilities SET NOT NULL;
    END IF;
END $$;
