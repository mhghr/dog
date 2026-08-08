-- 000022_monitors: Evolves the existing monitors table into a
-- resource-aware monitoring engine. Adds monitor type registry,
-- resource binding, probe assignments, and job execution tracking.

-- 1. Monitor type registry — each type defines its config schema, metrics,
--    health parameters, and supported resource types.

CREATE TABLE monitor_types (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    category VARCHAR(100) NOT NULL,
    execution_type VARCHAR(50) NOT NULL DEFAULT 'probe'
        CHECK (execution_type IN ('probe', 'agent', 'collector', 'hybrid')),
    executor_key VARCHAR(100) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    icon VARCHAR(50) NOT NULL DEFAULT 'activity',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    metric_keys TEXT[] NOT NULL DEFAULT '{}',
    configuration_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    default_configuration JSONB NOT NULL DEFAULT '{}'::jsonb,
    metric_schema JSONB NOT NULL DEFAULT '{}'::jsonb,
    health_parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    supported_resource_types TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX monitor_types_category_idx ON monitor_types(category);
CREATE INDEX monitor_types_executor_idx ON monitor_types(executor_key);

-- 2. Evolve the existing monitors table: bind to resources instead of
--    being standalone, and reference the monitor_type registry.

ALTER TABLE monitors ADD COLUMN resource_id UUID REFERENCES resources(id);
ALTER TABLE monitors ADD COLUMN monitor_type_id UUID REFERENCES monitor_types(id);
ALTER TABLE monitors ADD COLUMN severity VARCHAR(20) NOT NULL DEFAULT 'warning'
    CHECK (severity IN ('info', 'warning', 'critical', 'emergency'));

CREATE INDEX monitors_resource_idx ON monitors(resource_id);
CREATE INDEX monitors_monitor_type_idx ON monitors(monitor_type_id);

-- 3. Probe assignments — maps monitors to probe agents that execute them.

CREATE TABLE probe_assignments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    probe_id UUID NOT NULL REFERENCES probe_agents(id) ON DELETE CASCADE,
    priority SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (monitor_id, probe_id)
);

CREATE INDEX probe_assignments_monitor_idx ON probe_assignments(monitor_id);
CREATE INDEX probe_assignments_probe_idx ON probe_assignments(probe_id);

-- 4. Monitor job execution tracking

CREATE TYPE job_status AS ENUM ('pending', 'running', 'success', 'failed', 'timeout');

CREATE TABLE monitor_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    probe_id UUID REFERENCES probe_agents(id),
    scheduled_at TIMESTAMPTZ NOT NULL,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    status job_status NOT NULL DEFAULT 'pending',
    attempt SMALLINT NOT NULL DEFAULT 1,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX monitor_jobs_monitor_idx ON monitor_jobs(monitor_id);
CREATE INDEX monitor_jobs_probe_idx ON monitor_jobs(probe_id);
CREATE INDEX monitor_jobs_scheduled_idx ON monitor_jobs(scheduled_at) WHERE status = 'pending';
CREATE INDEX monitor_jobs_status_idx ON monitor_jobs(status);

-- 5. Seed standard monitor types

INSERT INTO monitor_types (name, slug, category, execution_type, executor_key, description, icon, enabled, metric_keys, configuration_schema, default_configuration, metric_schema, health_parameters, supported_resource_types) VALUES
('Ping', 'ping', 'network', 'probe', 'ping',
    'ICMP echo request to measure reachability and latency',
    'activity', TRUE,
    ARRAY['latency_ms', 'packet_loss', 'jitter_ms'],
    '{"type":"object","properties":{"count":{"type":"integer","default":3},"packet_size":{"type":"integer","default":56}},"required":[]}'::jsonb,
    '{"count":3,"packet_size":56}'::jsonb,
    '{"latency_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"Round-trip time in milliseconds"},"packet_loss":{"type":"percentage","unit":"%","direction":"LOWER_IS_BETTER","description":"Percentage of packets lost"},"jitter_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"Packet delay variation"}}'::jsonb,
    '{"latency_ms":{"default_profile":"Recommended","warning_threshold":200,"critical_threshold":500},"packet_loss":{"default_profile":"Sensitive","warning_threshold":1,"critical_threshold":5}}'::jsonb,
    ARRAY['server', 'virtual-machine', 'network-device', 'router', 'switch', 'firewall', 'website', 'api-endpoint', 'cloud-service']
),
('HTTP Check', 'http', 'web', 'probe', 'http',
    'HTTP/HTTPS request to verify availability and response',
    'globe', TRUE,
    ARRAY['status_code', 'response_time_ms', 'response_size_bytes', 'tls_days_remaining'],
    '{"type":"object","properties":{"url":{"type":"string","format":"uri"},"method":{"type":"string","enum":["GET","POST","PUT","HEAD","PATCH","DELETE"],"default":"GET"},"headers":{"type":"object"},"body":{"type":"string"},"expected_status":{"type":"integer","default":200},"follow_redirects":{"type":"boolean","default":true},"verify_ssl":{"type":"boolean","default":true}},"required":["url"]}'::jsonb,
    '{"method":"GET","expected_status":200,"follow_redirects":true,"verify_ssl":true}'::jsonb,
    '{"status_code":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"HTTP response status code"},"response_time_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"Time to first byte"},"tls_days_remaining":{"type":"number","unit":"days","direction":"HIGHER_IS_BETTER","description":"Days until TLS certificate expires"}}'::jsonb,
    '{"response_time_ms":{"default_profile":"Recommended","warning_threshold":2000,"critical_threshold":5000},"tls_days_remaining":{"default_profile":"Recommended","warning_threshold":30,"critical_threshold":14}}'::jsonb,
    ARRAY['website', 'api-endpoint', 'cloud-service']
),
('TCP Port', 'tcp', 'network', 'probe', 'tcp',
    'TCP connection check to verify port reachability',
    'activity', TRUE,
    ARRAY['connected', 'connect_time_ms'],
    '{"type":"object","properties":{"host":{"type":"string"},"port":{"type":"integer"}},"required":["host","port"]}'::jsonb,
    '{}'::jsonb,
    '{"connected":{"type":"boolean","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the TCP connection succeeded"},"connect_time_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"TCP handshake duration"}}'::jsonb,
    '{"connect_time_ms":{"default_profile":"Recommended","warning_threshold":1000,"critical_threshold":3000}}'::jsonb,
    ARRAY['server', 'virtual-machine', 'database', 'network-device', 'firewall']
),
('DNS Resolution', 'dns', 'network', 'probe', 'dns',
    'DNS query to verify domain resolution and response time',
    'search', TRUE,
    ARRAY['response_time_ms', 'resolved', 'records'],
    '{"type":"object","properties":{"domain":{"type":"string"},"record_type":{"type":"string","enum":["A","AAAA","CNAME","MX","TXT","NS"],"default":"A"},"nameserver":{"type":"string"}},"required":["domain"]}'::jsonb,
    '{"record_type":"A"}'::jsonb,
    '{"response_time_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"DNS resolution time"},"resolved":{"type":"boolean","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the domain resolved"}}'::jsonb,
    '{"response_time_ms":{"default_profile":"Recommended","warning_threshold":500,"critical_threshold":2000}}'::jsonb,
    ARRAY['website', 'api-endpoint', 'server', 'cloud-service']
),
('SSL Certificate', 'ssl', 'security', 'probe', 'tls',
    'TLS/SSL certificate validity and expiration monitoring',
    'shield', TRUE,
    ARRAY['days_remaining', 'valid', 'issuer', 'subject_cn', 'signed_by_known_ca'],
    '{"type":"object","properties":{"host":{"type":"string"},"port":{"type":"integer","default":443}},"required":["host"]}'::jsonb,
    '{"port":443}'::jsonb,
    '{"days_remaining":{"type":"number","unit":"days","direction":"HIGHER_IS_BETTER","description":"Days until certificate expires"},"valid":{"type":"boolean","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the certificate is currently valid"}}'::jsonb,
    '{"days_remaining":{"default_profile":"Recommended","warning_threshold":30,"critical_threshold":14}}'::jsonb,
    ARRAY['website', 'api-endpoint', 'server']
),
('Domain Expiry', 'domain-expiration', 'security', 'probe', 'domain_expiration',
    'WHOIS-based domain expiration monitoring',
    'calendar', TRUE,
    ARRAY['days_remaining', 'expiration_date', 'registrar'],
    '{"type":"object","properties":{"domain":{"type":"string"}},"required":["domain"]}'::jsonb,
    '{}'::jsonb,
    '{"days_remaining":{"type":"number","unit":"days","direction":"HIGHER_IS_BETTER","description":"Days until domain expires"}}'::jsonb,
    '{"days_remaining":{"default_profile":"Recommended","warning_threshold":45,"critical_threshold":14}}'::jsonb,
    ARRAY['website']
),
('Host Metrics', 'host-metrics', 'infrastructure', 'agent', '',
    'CPU, memory, disk, and network utilization via agent',
    'server', TRUE,
    ARRAY['cpu_percent', 'memory_percent', 'disk_percent', 'network_rx_bytes', 'network_tx_bytes', 'load_average'],
    '{"type":"object","properties":{"collectors":{"type":"array","items":{"type":"string","enum":["cpu","memory","disk","network","load"]}},"disk_paths":{"type":"array","items":{"type":"string"}},"network_interfaces":{"type":"array","items":{"type":"string"}}},"required":[]}'::jsonb,
    '{"collectors":["cpu","memory","disk","network","load"]}'::jsonb,
    '{"cpu_percent":{"type":"percentage","unit":"%","direction":"HIGHER_IS_WORSE","description":"CPU utilization percentage"},"memory_percent":{"type":"percentage","unit":"%","direction":"HIGHER_IS_WORSE","description":"Memory utilization percentage"},"disk_percent":{"type":"percentage","unit":"%","direction":"HIGHER_IS_WORSE","description":"Disk utilization percentage"}}'::jsonb,
    '{"cpu_percent":{"default_profile":"Recommended","warning_threshold":80,"critical_threshold":95},"memory_percent":{"default_profile":"Recommended","warning_threshold":85,"critical_threshold":95},"disk_percent":{"default_profile":"Recommended","warning_threshold":85,"critical_threshold":95}}'::jsonb,
    ARRAY['server', 'virtual-machine']
),
('Docker Monitor', 'docker', 'container', 'agent', '',
    'Docker container and host metrics via agent',
    'container', TRUE,
    ARRAY['container_count', 'running_containers', 'stopped_containers', 'cpu_percent', 'memory_percent'],
    '{"type":"object","properties":{"container_names":{"type":"array","items":{"type":"string"}},"include_stopped":{"type":"boolean","default":false}},"required":[]}'::jsonb,
    '{"include_stopped":false}'::jsonb,
    '{"container_count":{"type":"number","unit":"","direction":"NONE","description":"Total number of containers"},"cpu_percent":{"type":"percentage","unit":"%","direction":"HIGHER_IS_WORSE","description":"Container CPU utilization"}}'::jsonb,
    '{"cpu_percent":{"default_profile":"Recommended","warning_threshold":80,"critical_threshold":95}}'::jsonb,
    ARRAY['docker-host']
),
('Kubernetes Monitor', 'kubernetes', 'container', 'agent', '',
    'Kubernetes cluster health and resource metrics',
    'cloud', TRUE,
    ARRAY['pod_count', 'node_count', 'cpu_percent', 'memory_percent', 'restart_count'],
    '{"type":"object","properties":{"namespaces":{"type":"array","items":{"type":"string"}},"collect_pod_metrics":{"type":"boolean","default":true}},"required":[]}'::jsonb,
    '{"collect_pod_metrics":true}'::jsonb,
    '{"pod_count":{"type":"number","unit":"","direction":"NONE","description":"Total pods in the cluster"},"node_count":{"type":"number","unit":"","direction":"NONE","description":"Total nodes in the cluster"},"restart_count":{"type":"number","unit":"","direction":"HIGHER_IS_WORSE","description":"Pod restart count in the window"}}'::jsonb,
    '{"restart_count":{"default_profile":"Recommended","warning_threshold":5,"critical_threshold":20}}'::jsonb,
    ARRAY['kubernetes-cluster']
),
('Database Monitor', 'database', 'database', 'agent', '',
    'Database connection pool, query latency, and health metrics',
    'database', TRUE,
    ARRAY['connections_active', 'connections_idle', 'query_latency_ms', 'replication_lag_seconds', 'deadlocks'],
    '{"type":"object","properties":{"engine":{"type":"string","enum":["postgresql","mysql","mongodb","redis"]},"metrics":{"type":"array","items":{"type":"string","enum":["connections","queries","replication","locks","cache"]}}},"required":["engine"]}'::jsonb,
    '{"metrics":["connections","queries"]}'::jsonb,
    '{"connections_active":{"type":"number","unit":"","direction":"HIGHER_IS_WORSE","description":"Number of active connections"},"query_latency_ms":{"type":"number","unit":"ms","direction":"HIGHER_IS_WORSE","description":"Average query latency"}}'::jsonb,
    '{"connections_active":{"default_profile":"Recommended","warning_threshold":80,"critical_threshold":100},"query_latency_ms":{"default_profile":"Recommended","warning_threshold":500,"critical_threshold":2000}}'::jsonb,
    ARRAY['database']
),
('SNMP Monitor', 'snmp', 'network', 'probe', 'snmp',
    'SNMP-based device monitoring for routers, switches, and appliances',
    'wifi', TRUE,
    ARRAY['if_in_octets', 'if_out_octets', 'if_oper_status', 'cpu_percent', 'memory_percent', 'uptime_seconds'],
    '{"type":"object","properties":{"host":{"type":"string"},"port":{"type":"integer","default":161},"version":{"type":"string","enum":["1","2c","3"],"default":"2c"},"community":{"type":"string"},"oids":{"type":"array","items":{"type":"object","properties":{"name":{"type":"string"},"oid":{"type":"string"},"type":{"type":"string","enum":["gauge","counter","string"]}}}},"required":["host"]}'::jsonb,
    '{"port":161,"version":"2c"}'::jsonb,
    '{"if_oper_status":{"type":"enum","unit":"","direction":"ENUM_STATE","description":"Interface operational status (1=up,2=down)"},"cpu_percent":{"type":"percentage","unit":"%","direction":"HIGHER_IS_WORSE","description":"Device CPU utilization"}}'::jsonb,
    '{"cpu_percent":{"default_profile":"Recommended","warning_threshold":80,"critical_threshold":95}}'::jsonb,
    ARRAY['network-device', 'router', 'switch', 'firewall']
);
