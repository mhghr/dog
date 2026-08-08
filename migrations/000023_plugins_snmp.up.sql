-- 000023_plugins_snmp: Plugin architecture and SNMP monitoring.
-- Plugins are the extensibility layer — probe plugins, agent plugins,
-- and collector plugins. SNMP tables support enterprise network monitoring.

-- 1. Plugin registry

CREATE TYPE plugin_type AS ENUM ('probe', 'agent', 'collector');

CREATE TABLE plugins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    type plugin_type NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    icon VARCHAR(50) NOT NULL DEFAULT 'box',
    category VARCHAR(100) NOT NULL DEFAULT 'general',
    configuration_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX plugins_type_idx ON plugins(type);
CREATE INDEX plugins_category_idx ON plugins(category);
CREATE INDEX plugins_enabled_idx ON plugins(enabled) WHERE enabled = TRUE;

-- 2. SNMP credential store — encrypted at the application layer

CREATE TABLE snmp_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    version VARCHAR(10) NOT NULL DEFAULT '2c'
        CHECK (version IN ('1', '2c', '3')),
    community VARCHAR(255),
    username VARCHAR(255),
    authentication_protocol VARCHAR(20)
        CHECK (authentication_protocol IS NULL OR authentication_protocol IN ('MD5', 'SHA', 'SHA-256')),
    authentication_passphrase TEXT,
    privacy_protocol VARCHAR(20)
        CHECK (privacy_protocol IS NULL OR privacy_protocol IN ('DES', 'AES', 'AES-256')),
    privacy_passphrase TEXT,
    security_level VARCHAR(20) NOT NULL DEFAULT 'noAuthNoPriv'
        CHECK (security_level IN ('noAuthNoPriv', 'authNoPriv', 'authPriv')),
    context_name VARCHAR(255),
    encrypted_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX snmp_credentials_workspace_idx ON snmp_credentials(workspace_id);

-- 3. SNMP device with custom OID monitoring

CREATE TABLE snmp_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    credential_id UUID NOT NULL REFERENCES snmp_credentials(id),
    transport VARCHAR(10) NOT NULL DEFAULT 'udp'
        CHECK (transport IN ('udp', 'tcp')),
    port INTEGER NOT NULL DEFAULT 161,
    max_repetitions INTEGER NOT NULL DEFAULT 10,
    timeout_seconds INTEGER NOT NULL DEFAULT 5,
    retries INTEGER NOT NULL DEFAULT 1,
    oids JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (resource_id, credential_id)
);

CREATE INDEX snmp_devices_resource_idx ON snmp_devices(resource_id);

-- 4. Seed built-in plugins

INSERT INTO plugins (name, slug, description, type, version, icon, category, configuration_schema, enabled) VALUES
-- Probe plugins
('Ping Plugin', 'ping-plugin', 'ICMP ping probe for reachability and latency measurement', 'probe', '1.0.0', 'activity', 'network',
    '{"type":"object","properties":{"count":{"type":"integer","default":3},"packet_size":{"type":"integer","default":56},"privileged":{"type":"boolean","default":false}},"required":[]}'::jsonb, TRUE),
('HTTP Plugin', 'http-plugin', 'HTTP/HTTPS request probe with TLS, redirect, and header support', 'probe', '1.0.0', 'globe', 'web',
    '{"type":"object","properties":{"method":{"type":"string"},"headers":{"type":"object"},"body":{"type":"string"},"follow_redirects":{"type":"boolean"},"verify_ssl":{"type":"boolean"}},"required":[]}'::jsonb, TRUE),
('DNS Plugin', 'dns-plugin', 'DNS resolution probe with support for all record types and custom nameservers', 'probe', '1.0.0', 'search', 'network',
    '{"type":"object","properties":{"record_type":{"type":"string","default":"A"},"nameserver":{"type":"string"}},"required":[]}'::jsonb, TRUE),
('TCP Plugin', 'tcp-plugin', 'TCP port connectivity probe with TLS support', 'probe', '1.0.0', 'activity', 'network',
    '{"type":"object","properties":{"port":{"type":"integer"},"tls":{"type":"boolean","default":false}},"required":["port"]}'::jsonb, TRUE),
('SSL Plugin', 'ssl-plugin', 'TLS/SSL certificate validation and expiration monitoring', 'probe', '1.0.0', 'shield', 'security',
    '{"type":"object","properties":{"port":{"type":"integer","default":443},"sni":{"type":"string"}},"required":[]}'::jsonb, TRUE),
('SNMP Plugin', 'snmp-plugin', 'SNMP v1/v2c/v3 network device monitoring', 'probe', '1.0.0', 'wifi', 'network',
    '{"type":"object","properties":{"version":{"type":"string"},"port":{"type":"integer","default":161},"transport":{"type":"string","default":"udp"}},"required":["version"]}'::jsonb, TRUE),
-- Agent plugins
('Host Metrics Plugin', 'host-metrics-plugin', 'CPU, memory, disk, network, and load metrics via agent', 'agent', '1.0.0', 'server', 'infrastructure',
    '{"type":"object","properties":{"collectors":{"type":"array","items":{"type":"string"}},"interval_seconds":{"type":"integer","default":60}},"required":[]}'::jsonb, TRUE),
('Process Plugin', 'process-plugin', 'Process-level metrics: count, CPU, memory per process', 'agent', '1.0.0', 'list', 'infrastructure',
    '{"type":"object","properties":{"process_names":{"type":"array","items":{"type":"string"}},"include_children":{"type":"boolean","default":true}},"required":[]}'::jsonb, TRUE),
('Log Plugin', 'log-plugin', 'Log tailing and forwarding with pattern matching', 'agent', '1.0.0', 'file-text', 'observability',
    '{"type":"object","properties":{"paths":{"type":"array","items":{"type":"string"}},"patterns":{"type":"array","items":{"type":"string"}},"multiline":{"type":"boolean","default":false}},"required":["paths"]}'::jsonb, TRUE),
('Trace Plugin', 'trace-plugin', 'Distributed tracing via OpenTelemetry protocol', 'agent', '1.0.0', 'activity', 'observability',
    '{"type":"object","properties":{"otlp_endpoint":{"type":"string"},"sampling_rate":{"type":"number","default":0.1,"minimum":0,"maximum":1}},"required":["otlp_endpoint"]}'::jsonb, TRUE),
('Docker Plugin', 'docker-plugin', 'Docker daemon metrics: containers, images, volumes, events', 'agent', '1.0.0', 'container', 'container',
    '{"type":"object","properties":{"socket_path":{"type":"string","default":"/var/run/docker.sock"},"include_stopped":{"type":"boolean","default":false}},"required":[]}'::jsonb, TRUE),
('Kubernetes Plugin', 'kubernetes-plugin', 'Kubernetes cluster metrics via kubelet and API server', 'agent', '1.0.0', 'cloud', 'container',
    '{"type":"object","properties":{"kubeconfig_path":{"type":"string"},"in_cluster":{"type":"boolean","default":false},"namespaces":{"type":"array","items":{"type":"string"}}},"required":[]}'::jsonb, TRUE),
('Database Plugin', 'database-plugin', 'Database connection, query, and replication metrics', 'agent', '1.0.0', 'database', 'database',
    '{"type":"object","properties":{"engines":{"type":"array","items":{"type":"string","enum":["postgresql","mysql","mongodb","redis"]}},"connection_strings":{"type":"object"}},"required":["engines"]}'::jsonb, TRUE),
('Prometheus Plugin', 'prometheus-plugin', 'Prometheus metrics scraping and forwarding', 'collector', '1.0.0', 'bar-chart', 'observability',
    '{"type":"object","properties":{"scrape_targets":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"url":{"type":"string","format":"uri"}}}},"scrape_interval_seconds":{"type":"integer","default":30}},"required":["scrape_targets"]}'::jsonb, TRUE);
