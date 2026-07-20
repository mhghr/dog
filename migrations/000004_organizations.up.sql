CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, name)
);

CREATE INDEX projects_org_idx ON projects(organization_id);

-- Every user creates exactly one organization at sign-up (simplest
-- viable multi-tenancy). Later phases can add invitations.
ALTER TABLE users ADD COLUMN organization_id UUID REFERENCES organizations(id);

-- Scoped query helpers: monitors (and any future org-scoped resources)
-- carry the project that owns them.
ALTER TABLE monitors ADD COLUMN project_id UUID REFERENCES projects(id);
CREATE INDEX monitors_project_idx ON monitors(project_id);
