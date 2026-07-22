CREATE TYPE health_rule_mode AS ENUM ('INHERIT_DEFAULT', 'USE_PROFILE', 'CUSTOM', 'DISABLED');
CREATE TYPE health_rule_profile AS ENUM ('Sensitive', 'Recommended', 'Relaxed');
CREATE TYPE health_state AS ENUM ('OK', 'WARNING', 'ERROR', 'UNKNOWN');
CREATE TYPE parameter_data_type AS ENUM ('NUMBER', 'BOOLEAN', 'ENUM', 'STRING', 'DURATION', 'PERCENTAGE', 'BYTES', 'TIMESTAMP');
CREATE TYPE parameter_direction AS ENUM ('HIGHER_IS_WORSE', 'LOWER_IS_WORSE', 'BOOLEAN_FAILURE', 'ENUM_STATE', 'RANGE_DEVIATION', 'CHANGE_EVENT', 'RATE', 'COUNT');

CREATE TABLE parameter_catalog (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(100) NOT NULL,
    monitor_type monitor_type NOT NULL,
    name VARCHAR(200) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    data_type parameter_data_type NOT NULL DEFAULT 'NUMBER',
    unit VARCHAR(50) NOT NULL DEFAULT '',
    direction parameter_direction NOT NULL DEFAULT 'HIGHER_IS_WORSE',
    default_profile health_rule_profile NOT NULL DEFAULT 'Recommended',
    UNIQUE(key, monitor_type)
);

CREATE TABLE parameter_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    parameter_key VARCHAR(100) NOT NULL,
    mode health_rule_mode NOT NULL DEFAULT 'INHERIT_DEFAULT',
    profile health_rule_profile,
    aggregation VARCHAR(20) NOT NULL DEFAULT 'avg',
    window_type VARCHAR(20) NOT NULL DEFAULT 'checks',
    window_value INTEGER NOT NULL DEFAULT 3,
    warning_operator VARCHAR(10) NOT NULL DEFAULT 'gte',
    warning_value DOUBLE PRECISION,
    error_operator VARCHAR(10) NOT NULL DEFAULT 'gte',
    error_value DOUBLE PRECISION,
    recovery_operator VARCHAR(10),
    recovery_value DOUBLE PRECISION,
    minimum_samples INTEGER NOT NULL DEFAULT 3,
    consecutive_failures INTEGER,
    consecutive_successes INTEGER,
    missing_data_policy VARCHAR(30) NOT NULL DEFAULT 'IGNORE',
    missed_checks INTEGER NOT NULL DEFAULT 3,
    cooldown_seconds INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(monitor_id, parameter_key)
);

CREATE TABLE health_notification_channels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE notification_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    monitor_id UUID REFERENCES monitors(id) ON DELETE CASCADE,
    parameter_key VARCHAR(100),
    channel_id UUID NOT NULL REFERENCES health_notification_channels(id) ON DELETE CASCADE,
    triggers TEXT[] NOT NULL DEFAULT '{}',
    delay_seconds INTEGER NOT NULL DEFAULT 0,
    repeat_interval_seconds INTEGER NOT NULL DEFAULT 0,
    cooldown_seconds INTEGER NOT NULL DEFAULT 300,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE parameter_health_state (
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    parameter_key VARCHAR(100) NOT NULL,
    current_state health_state NOT NULL DEFAULT 'UNKNOWN',
    current_value DOUBLE PRECISION,
    evaluated_at TIMESTAMPTZ,
    previous_state health_state,
    state_changed_at TIMESTAMPTZ,
    PRIMARY KEY(monitor_id, parameter_key)
);
