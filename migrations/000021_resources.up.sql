-- 000021_resources: Resource management — the primary entity.
-- Everything monitored is a Resource. Supports dynamic typing
-- via resource_types, flexible tagging, and JSONB metadata.

-- 1. Resource type registry

CREATE TABLE resource_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category VARCHAR(100) NOT NULL,
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    icon VARCHAR(50) NOT NULL DEFAULT 'server',
    capabilities TEXT[] NOT NULL DEFAULT '{}',
    configuration_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX resource_types_category_idx ON resource_types(category);

-- 2. Resources table (the central entity)

CREATE TABLE resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    resource_type_id UUID NOT NULL REFERENCES resource_types(id),
    created_by UUID REFERENCES users(id),
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    target TEXT NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive', 'warning', 'error', 'maintenance')),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX resources_org_idx ON resources(organization_id);
CREATE INDEX resources_workspace_idx ON resources(workspace_id);
CREATE INDEX resources_type_idx ON resources(resource_type_id);
CREATE INDEX resources_status_idx ON resources(status);
CREATE INDEX resources_metadata_idx ON resources USING GIN (metadata);

-- Enable trigram extension for fuzzy text search on resource names
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX resources_name_trgm_idx ON resources USING GIN (name gin_trgm_ops);

-- 3. Flexible tagging system

CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    key VARCHAR(100) NOT NULL,
    value VARCHAR(200) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, key, value)
);

CREATE INDEX tags_org_idx ON tags(organization_id);
CREATE INDEX tags_key_value_idx ON tags(organization_id, key, value);

CREATE TABLE resource_tags (
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (resource_id, tag_id)
);

CREATE INDEX resource_tags_tag_idx ON resource_tags(tag_id);

-- 4. Seed standard resource types

INSERT INTO resource_types (category, name, slug, icon, capabilities, configuration_schema) VALUES
('infrastructure', 'Server', 'server', 'server',
    ARRAY['host-metrics', 'ping', 'process', 'log'],
    '{"type":"object","properties":{"hostname":{"type":"string"},"os":{"type":"string","enum":["linux","windows","macos"]},"arch":{"type":"string"}},"required":["hostname"]}'::jsonb),
('infrastructure', 'Virtual Machine', 'virtual-machine', 'monitor',
    ARRAY['host-metrics', 'ping'],
    '{"type":"object","properties":{"hostname":{"type":"string"},"os":{"type":"string"},"hypervisor":{"type":"string"}},"required":["hostname"]}'::jsonb),
('web', 'Website', 'website', 'globe',
    ARRAY['http', 'ssl', 'dns', 'ping'],
    '{"type":"object","properties":{"url":{"type":"string","format":"uri"},"method":{"type":"string","enum":["GET","POST","HEAD"]}},"required":["url"]}'::jsonb),
('web', 'API Endpoint', 'api-endpoint', 'code',
    ARRAY['http', 'ssl', 'dns'],
    '{"type":"object","properties":{"url":{"type":"string","format":"uri"},"method":{"type":"string"},"headers":{"type":"object"},"body":{"type":"string"}},"required":["url"]}'::jsonb),
('database', 'Database', 'database', 'database',
    ARRAY['tcp', 'query'],
    '{"type":"object","properties":{"host":{"type":"string"},"port":{"type":"integer"},"engine":{"type":"string","enum":["postgresql","mysql","mongodb","redis"]},"database":{"type":"string"}},"required":["host","port","engine"]}'::jsonb),
('container', 'Docker Host', 'docker-host', 'container',
    ARRAY['docker', 'host-metrics'],
    '{"type":"object","properties":{"host":{"type":"string"},"port":{"type":"integer","default":2375},"tls":{"type":"boolean","default":false}},"required":["host"]}'::jsonb),
('container', 'Kubernetes Cluster', 'kubernetes-cluster', 'cloud',
    ARRAY['kubernetes', 'prometheus'],
    '{"type":"object","properties":{"kubeconfig":{"type":"string"},"context":{"type":"string"}},"required":["kubeconfig"]}'::jsonb),
('network', 'Network Device', 'network-device', 'wifi',
    ARRAY['ping', 'snmp', 'tcp'],
    '{"type":"object","properties":{"host":{"type":"string"},"device_type":{"type":"string","enum":["router","switch","firewall","load-balancer"]}},"required":["host"]}'::jsonb),
('network', 'Router', 'router', 'wifi',
    ARRAY['ping', 'snmp'],
    '{"type":"object","properties":{"host":{"type":"string"},"model":{"type":"string"},"firmware":{"type":"string"}},"required":["host"]}'::jsonb),
('network', 'Switch', 'switch', 'layers',
    ARRAY['ping', 'snmp'],
    '{"type":"object","properties":{"host":{"type":"string"},"model":{"type":"string"},"port_count":{"type":"integer"}},"required":["host"]}'::jsonb),
('network', 'Firewall', 'firewall', 'shield',
    ARRAY['ping', 'snmp', 'tcp'],
    '{"type":"object","properties":{"host":{"type":"string"},"model":{"type":"string"}},"required":["host"]}'::jsonb),
('cloud', 'Cloud Service', 'cloud-service', 'cloud',
    ARRAY['http', 'ping'],
    '{"type":"object","properties":{"provider":{"type":"string","enum":["aws","gcp","azure","digitalocean"]},"region":{"type":"string"},"service":{"type":"string"}},"required":["provider","service"]}'::jsonb),
('custom', 'Custom Resource', 'custom-resource', 'box',
    ARRAY[],
    '{"type":"object","properties":{}}'::jsonb);
