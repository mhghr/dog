-- Undo workspace migration: restore the projects table from workspace data.

-- 1. Restore projects table and repopulate it from workspaces
CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, name)
);

INSERT INTO projects (id, organization_id, name, created_at, updated_at)
SELECT id, organization_id, name, created_at, updated_at
FROM workspaces;

-- 2. Restore project_id on monitors
ALTER TABLE monitors ADD COLUMN project_id UUID REFERENCES projects(id);

UPDATE monitors m
SET project_id = w.id
FROM workspaces w
WHERE m.workspace_id = w.id;

ALTER TABLE monitors DROP COLUMN workspace_id;

-- 3. Restore project_id on alert_policies
ALTER TABLE alert_policies ADD COLUMN project_id UUID REFERENCES projects(id);

UPDATE alert_policies ap
SET project_id = w.id
FROM workspaces w
WHERE ap.workspace_id = w.id;

ALTER TABLE alert_policies DROP COLUMN workspace_id;

-- 4. Restore organization_id on users from workspaces (via workspace_membership or first workspace)
ALTER TABLE users DROP COLUMN IF EXISTS status;
ALTER TABLE users DROP COLUMN IF EXISTS password_hash;

-- 5. Drop workspace tables
DROP TABLE IF EXISTS workspace_members;
DROP TABLE IF EXISTS workspaces;
DROP TYPE IF EXISTS workspace_role;
