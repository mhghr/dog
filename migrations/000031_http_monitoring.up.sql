-- 000031_http_monitoring: Aligns the seeded HTTP monitor type with the probe
-- executor configuration keys and the health parameter catalog. The original
-- seed (000022) used verify_ssl/expected_status while the executor reads
-- verify_tls/expected_status_codes and the health engine catalogs http.*
-- parameters. All statements are idempotent UPDATEs on the existing row.

UPDATE monitor_types
SET metric_keys = ARRAY[
        'reachability',
        'status_code',
        'response_time_ms',
        'request_write_ms',
        'ttfb_ms',
        'dns_duration_ms',
        'connect_duration_ms',
        'tls_duration_ms',
        'download_time_ms',
        'response_size_bytes',
        'content_assertion'
    ],
    configuration_schema = '{"type":"object","properties":{"method":{"type":"string","enum":["GET","POST","PUT","PATCH","DELETE","HEAD","OPTIONS"],"default":"GET"},"follow_redirects":{"type":"boolean","default":true},"max_redirects":{"type":"integer","default":5},"verify_tls":{"type":"boolean","default":true},"expected_status_codes":{"type":"string","default":"200"},"headers":{"type":"object"},"request_body":{"type":"string"},"body_contains":{"type":"string","default":""},"ip_version":{"type":"string","enum":["auto","ipv4","ipv6"],"default":"auto"},"max_response_size_bytes":{"type":"integer","default":10485760}},"required":[]}'::jsonb,
    default_configuration = '{"method":"GET","follow_redirects":true,"max_redirects":5,"verify_tls":true,"expected_status_codes":"200","ip_version":"auto","max_response_size_bytes":10485760}'::jsonb,
    metric_schema = '{"reachability":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the endpoint responded"},"status_code":{"type":"number","unit":"","direction":"ENUM_STATE","description":"HTTP response status code"},"response_time_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"Total request duration"},"request_write_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"Time to write the request"},"ttfb_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"Time to first byte"},"dns_duration_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"DNS resolution duration"},"connect_duration_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"TCP connect duration"},"tls_duration_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"TLS handshake duration"},"download_time_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"Body download duration"},"response_size_bytes":{"type":"number","unit":"bytes","direction":"NONE","description":"Response body size in bytes"},"content_assertion":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the response body matched the expected text"}}'::jsonb,
    health_parameters = '{"reachability":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0},"status_code":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0},"response_time_ms":{"default_profile":"Recommended","warning_threshold":2000,"critical_threshold":5000},"ttfb_ms":{"default_profile":"Recommended","warning_threshold":1000,"critical_threshold":3000},"dns_duration_ms":{"default_profile":"Recommended","warning_threshold":500,"critical_threshold":2000},"connect_duration_ms":{"default_profile":"Recommended","warning_threshold":500,"critical_threshold":2000},"tls_duration_ms":{"default_profile":"Recommended","warning_threshold":500,"critical_threshold":2000},"content_assertion":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0}}'::jsonb
WHERE slug = 'http';
