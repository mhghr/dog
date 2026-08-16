-- Update the Ping monitor-type seed thresholds to the spec health rules:
-- packet loss warning >= 5%, critical >= 20%.
UPDATE monitor_types
SET health_parameters = jsonb_set(
        health_parameters,
        '{packet_loss,warning_threshold}',
        '5'
    )
WHERE slug = 'ping';

UPDATE monitor_types
SET health_parameters = jsonb_set(
        health_parameters,
        '{packet_loss,critical_threshold}',
        '20'
    )
WHERE slug = 'ping';
