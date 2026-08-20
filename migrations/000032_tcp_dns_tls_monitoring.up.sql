-- 000032_tcp_dns_tls_monitoring: Aligns the seeded TCP, DNS and SSL monitor
-- types with the production-grade probe executors and the health parameter
-- catalog. The original seeds (000022) used legacy metric keys and
-- config fields that the rewritten executors no longer emit/read. All
-- statements are idempotent UPDATEs on the existing rows.

-- ── TCP Port ─────────────────────────────────────────────────────────────
UPDATE monitor_types
SET metric_keys = ARRAY['reachability', 'connect_time_ms'],
    configuration_schema = '{"type":"object","properties":{"port":{"type":"integer","minimum":1,"maximum":65535,"default":80},"timeout_ms":{"type":"integer","minimum":100,"maximum":60000,"default":5000},"ip_version":{"type":"string","enum":["auto","ipv4","ipv6"],"default":"auto"}},"required":[]}'::jsonb,
    default_configuration = '{"port":80,"timeout_ms":5000,"ip_version":"auto"}'::jsonb,
    metric_schema = '{"reachability":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether a TCP connection was established"},"connect_time_ms":{"type":"number","unit":"ms","direction":"HIGHER_IS_WORSE","description":"Time to establish the TCP connection"}}'::jsonb,
    health_parameters = '{"reachability":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0},"connect_time_ms":{"default_profile":"Recommended","warning_threshold":500,"critical_threshold":2000}}'::jsonb
WHERE slug = 'tcp';

-- ── DNS Resolution ────────────────────────────────────────────────────────
UPDATE monitor_types
SET metric_keys = ARRAY['reachability', 'response_time_ms', 'answer_count', 'ttl_seconds', 'expected_record_match'],
    configuration_schema = '{"type":"object","properties":{"record_type":{"type":"string","enum":["A","AAAA","CNAME","MX","TXT","NS"],"default":"A"},"resolver":{"type":"string","title":"Resolver"},"expected_values":{"type":"string","title":"Expected values"},"timeout_ms":{"type":"integer","minimum":100,"maximum":60000,"default":5000},"ip_version":{"type":"string","enum":["auto","ipv4","ipv6"],"default":"auto"}},"required":[]}'::jsonb,
    default_configuration = '{"record_type":"A","timeout_ms":5000,"ip_version":"auto"}'::jsonb,
    metric_schema = '{"reachability":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the DNS query succeeded"},"response_time_ms":{"type":"number","unit":"ms","direction":"HIGHER_IS_WORSE","description":"DNS query duration"},"answer_count":{"type":"number","unit":"","direction":"NONE","description":"Number of answers returned"},"ttl_seconds":{"type":"number","unit":"s","direction":"NONE","description":"Minimum TTL across answers"},"expected_record_match":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the answers matched the expected values"}}'::jsonb,
    health_parameters = '{"reachability":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0},"response_time_ms":{"default_profile":"Recommended","warning_threshold":500,"critical_threshold":2000},"expected_record_match":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0}}'::jsonb
WHERE slug = 'dns';

-- ── SSL Certificate ───────────────────────────────────────────────────────
UPDATE monitor_types
SET metric_keys = ARRAY['reachability', 'handshake_time_ms', 'certificate_expiry_days'],
    configuration_schema = '{"type":"object","properties":{"port":{"type":"integer","minimum":1,"maximum":65535,"default":443},"server_name":{"type":"string","title":"Server name (SNI)"},"verify_tls":{"type":"boolean","default":true},"min_tls_version":{"type":"string","enum":["1.0","1.1","1.2","1.3"],"default":"1.2"},"timeout_ms":{"type":"integer","minimum":100,"maximum":60000,"default":10000},"ip_version":{"type":"string","enum":["auto","ipv4","ipv6"],"default":"auto"}},"required":[]}'::jsonb,
    default_configuration = '{"port":443,"verify_tls":true,"min_tls_version":"1.2","timeout_ms":10000,"ip_version":"auto"}'::jsonb,
    metric_schema = '{"reachability":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the TLS handshake succeeded"},"handshake_time_ms":{"type":"number","unit":"ms","direction":"HIGHER_IS_WORSE","description":"TLS handshake duration"},"certificate_expiry_days":{"type":"number","unit":"days","direction":"LOWER_IS_WORSE","description":"Days until the certificate expires"}}'::jsonb,
    health_parameters = '{"reachability":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0},"handshake_time_ms":{"default_profile":"Recommended","warning_threshold":500,"critical_threshold":2000},"certificate_expiry_days":{"default_profile":"Recommended","warning_threshold":30,"critical_threshold":14}}'::jsonb
WHERE slug = 'ssl';
