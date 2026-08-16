UPDATE monitor_types
SET health_parameters = jsonb_set(
        health_parameters,
        '{packet_loss,warning_threshold}',
        '1'
    )
WHERE slug = 'ping';

UPDATE monitor_types
SET health_parameters = jsonb_set(
        health_parameters,
        '{packet_loss,critical_threshold}',
        '5'
    )
WHERE slug = 'ping';
