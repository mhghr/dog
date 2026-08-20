-- 000032_tcp_dns_tls_monitoring down: restores the original TCP/DNS/SSL
-- monitor-type seeds from 000022 so rolling back this migration returns the
-- previous schema.

UPDATE monitor_types
SET metric_keys = ARRAY['connected', 'connect_time_ms'],
    configuration_schema = '{"type":"object","properties":{"host":{"type":"string"},"port":{"type":"integer"}},"required":["host","port"]}'::jsonb,
    default_configuration = '{}'::jsonb,
    metric_schema = '{"connected":{"type":"boolean","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the TCP connection succeeded"},"connect_time_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"TCP handshake duration"}}'::jsonb,
    health_parameters = '{"connect_time_ms":{"default_profile":"Recommended","warning_threshold":1000,"critical_threshold":3000}}'::jsonb
WHERE slug = 'tcp';

UPDATE monitor_types
SET metric_keys = ARRAY['response_time_ms', 'resolved', 'records'],
    configuration_schema = '{"type":"object","properties":{"domain":{"type":"string"},"record_type":{"type":"string","enum":["A","AAAA","CNAME","MX","TXT","NS"],"default":"A"},"nameserver":{"type":"string"}},"required":["domain"]}'::jsonb,
    default_configuration = '{"record_type":"A"}'::jsonb,
    metric_schema = '{"response_time_ms":{"type":"number","unit":"ms","direction":"LOWER_IS_BETTER","description":"DNS resolution time"},"resolved":{"type":"boolean","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the domain resolved"}}'::jsonb,
    health_parameters = '{"response_time_ms":{"default_profile":"Recommended","warning_threshold":500,"critical_threshold":2000}}'::jsonb
WHERE slug = 'dns';

UPDATE monitor_types
SET metric_keys = ARRAY['days_remaining', 'valid', 'issuer', 'subject_cn', 'signed_by_known_ca'],
    configuration_schema = '{"type":"object","properties":{"host":{"type":"string"},"port":{"type":"integer","default":443}},"required":["host"]}'::jsonb,
    default_configuration = '{"port":443}'::jsonb,
    metric_schema = '{"days_remaining":{"type":"number","unit":"days","direction":"HIGHER_IS_BETTER","description":"Days until certificate expires"},"valid":{"type":"boolean","unit":"","direction":"BOOLEAN_FAILURE","description":"Whether the certificate is currently valid"}}'::jsonb,
    health_parameters = '{"days_remaining":{"default_profile":"Recommended","warning_threshold":30,"critical_threshold":14}}'::jsonb
WHERE slug = 'ssl';
