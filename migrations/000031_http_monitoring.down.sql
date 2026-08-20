-- 000031_http_monitoring down: restores the original HTTP monitor-type seed
-- from 000022 so rolling back this migration returns the previous schema.

UPDATE monitor_types
SET metric_keys = ARRAY['status_code', 'response_time_ms', 'response_size_bytes', 'tls_days_remaining'],
    configuration_schema = '{"type":"object","properties":{"url":{"type":"string","format":"uri"},"method":{"type":"string","enum":["GET","POST","PUT","HEAD","PATCH","DELETE"],"default":"GET"},"headers":{"type":"object"},"body":{"type":"string"},"expected_status":{"type":"integer","default":200},"follow_redirects":{"type":"boolean","default":true},"verify_ssl":{"type":"boolean","default":true}},"required":["url"]}'::jsonb,
    default_configuration = '{"method":"GET","expected_status":200,"follow_redirects":true,"verify_ssl":true}'::jsonb,
    metric_schema = '{"status_code":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"HTTP response status code"},"response_time_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"Time to first byte"},"tls_days_remaining":{"type":"number","unit":"days","direction":"HIGHER_IS_BETTER","description":"Days until TLS certificate expires"}}'::jsonb,
    health_parameters = '{"response_time_ms":{"default_profile":"Recommended","warning_threshold":2000,"critical_threshold":5000},"tls_days_remaining":{"default_profile":"Recommended","warning_threshold":30,"critical_threshold":14}}'::jsonb
WHERE slug = 'http';
