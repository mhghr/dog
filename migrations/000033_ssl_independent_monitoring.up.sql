-- 000033_ssl_independent_monitoring: extends the SSL Certificate monitor type
-- with the certificate validation metrics the TLS executor now emits
-- (certificate_valid, hostname_match, chain_valid) and keeps the health
-- parameters aligned with the ssl.* health catalog. Idempotent UPDATE.

UPDATE monitor_types
SET metric_keys = ARRAY[
        'reachability',
        'handshake_time_ms',
        'certificate_expiry_days',
        'certificate_valid',
        'hostname_match',
        'chain_valid'
    ],
    metric_schema = '{"reachability":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the TLS handshake succeeded"},"handshake_time_ms":{"type":"number","unit":"ms","direction":"HIGHER_IS_WORSE","description":"TLS handshake duration"},"certificate_expiry_days":{"type":"number","unit":"days","direction":"LOWER_IS_WORSE","description":"Days until the certificate expires"},"certificate_valid":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the certificate is currently time-valid"},"hostname_match":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the certificate matches the expected hostname (SNI)"},"chain_valid":{"type":"number","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the certificate chain is trusted"}}'::jsonb,
    health_parameters = '{"reachability":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0},"handshake_time_ms":{"default_profile":"Recommended","warning_threshold":500,"critical_threshold":2000},"certificate_expiry_days":{"default_profile":"Recommended","warning_threshold":30,"critical_threshold":14},"certificate_valid":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0},"hostname_match":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0},"chain_valid":{"default_profile":"Recommended","warning_threshold":0,"critical_threshold":0}}'::jsonb
WHERE slug = 'ssl';
