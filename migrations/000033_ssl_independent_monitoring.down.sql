-- 000033_ssl_independent_monitoring down: restores the 000032 SSL monitor-type
-- seed (without the certificate validation metric keys).

UPDATE monitor_types
SET metric_keys = ARRAY['reachability', 'handshake_time_ms', 'certificate_expiry_days'],
    metric_schema = '{"reachability":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the TLS handshake succeeded"},"handshake_time_ms":{"type":"number","unit":"ms","direction":"HIGHER_IS_WORSE","description":"TLS handshake duration"},"certificate_expiry_days":{"type":"number","unit":"days","direction":"LOWER_IS_WORSE","description":"Days until the certificate expires"}}'::jsonb,
    health_parameters = '{"reachability":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0},"handshake_time_ms":{"default_profile":"Recommended","warning_threshold":500,"critical_threshold":2000},"certificate_expiry_days":{"default_profile":"Recommended","warning_threshold":30,"critical_threshold":14}}'::jsonb
WHERE slug = 'ssl';
