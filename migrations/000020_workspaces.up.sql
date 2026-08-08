-- 000020_workspaces: Replaces the simple projects table with a full
-- workspace model including membership, roles, plans, and settings.
--
-- Data migration path: projects rows → workspaces rows.
-- After this migration, every entity references workspace_id instead of
-- project_id. The deprecated projects table is dropped.

-- 1. Core workspace types and tables

CREATE TYPE workspace_role AS ENUM ('owner', 'admin', 'editor', 'viewer');

CREATE TABLE workspaces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    plan VARCHAR(50) NOT NULL DEFAULT 'free',
    settings JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, slug)
);

CREATE INDEX workspaces_org_idx ON workspaces(organization_id);

CREATE TABLE workspace_members (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role workspace_role NOT NULL DEFAULT 'viewer',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (workspace_id, user_id)
);

CREATE INDEX workspace_members_user_idx ON workspace_members(user_id);

-- 2. Migrate existing projects into workspaces

INSERT INTO workspaces (id, organization_id, name, slug, created_at, updated_at)
SELECT id, organization_id, name, name, created_at, updated_at
FROM projects;

-- 3. Wire monitors: add workspace_id, migrate from project_id, drop old FK

ALTER TABLE monitors ADD COLUMN workspace_id UUID REFERENCES workspaces(id);

UPDATE monitors m
SET workspace_id = p.organization_id
FROM projects p
WHERE m.project_id IS NOT NULL
  AND m.project_id = p.id;

DROP INDEX IF EXISTS monitors_project_idx;
ALTER TABLE monitors DROP COLUMN project_id;

-- 4. Wire alert_policies the same way

ALTER TABLE alert_policies ADD COLUMN workspace_id UUID REFERENCES workspaces(id);

UPDATE alert_policies ap
SET workspace_id = p.organization_id
FROM projects p
WHERE ap.project_id IS NOT NULL
  AND ap.project_id = p.id;

ALTER TABLE alert_policies DROP COLUMN project_id;

-- 5. Drop the deprecated projects table and its FK on users

ALTER TABLE users DROP COLUMN IF EXISTS organization_id;

ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS status VARCHAR(20) NOT NULL DEFAULT 'active'
    CHECK (status IN ('active', 'inactive', 'suspended'));

DROP TABLE projects;
