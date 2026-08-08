-- 000024_time_series: High-volume time-series metric storage.
-- Optimized for millions of data points with BRIN indexes
-- and partitioning-friendly design.

-- 1. Metric series definition — each unique metric name + label set

CREATE TABLE metric_series (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    metric_name VARCHAR(255) NOT NULL,
    labels JSONB NOT NULL DEFAULT '{}'::jsonb,
    unit VARCHAR(50) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (resource_id, metric_name, labels)
);

CREATE INDEX metric_series_resource_idx ON metric_series(resource_id);
CREATE INDEX metric_series_name_idx ON metric_series(metric_name);

-- 2. Metric data points — append-only, BRIN-indexed for time-range scans

CREATE TABLE metric_points (
    time TIMESTAMPTZ NOT NULL,
    series_id UUID NOT NULL REFERENCES metric_series(id) ON DELETE CASCADE,
    value DOUBLE PRECISION NOT NULL,
    attributes JSONB NOT NULL DEFAULT '{}'::jsonb,
    PRIMARY KEY (time, series_id)
);

CREATE INDEX metric_points_series_time_idx ON metric_points(series_id, time DESC);
CREATE INDEX metric_points_time_brin_idx ON metric_points USING BRIN (time) WITH (pages_per_range = 32);

-- 3. Compressed metric rollup for historical data retention

CREATE TABLE metric_rollups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    series_id UUID NOT NULL REFERENCES metric_series(id) ON DELETE CASCADE,
    bucket TIMESTAMPTZ NOT NULL,
    bucket_width_seconds INTEGER NOT NULL,
    count BIGINT NOT NULL DEFAULT 0,
    sum DOUBLE PRECISION NOT NULL DEFAULT 0,
    min DOUBLE PRECISION,
    max DOUBLE PRECISION,
    p50 DOUBLE PRECISION,
    p95 DOUBLE PRECISION,
    p99 DOUBLE PRECISION,
    last_value DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (series_id, bucket, bucket_width_seconds)
);

CREATE INDEX metric_rollups_series_bucket_idx ON metric_rollups(series_id, bucket DESC);
