-- SentinelOps: core log events hypertable

CREATE EXTENSION IF NOT EXISTS timescaledb;

CREATE TABLE IF NOT EXISTS log_events (
    event_id         UUID NOT NULL,
    ts               TIMESTAMPTZ NOT NULL,
    service          TEXT NOT NULL,
    endpoint         TEXT NOT NULL,
    level            TEXT NOT NULL,
    status_code      INTEGER NOT NULL,
    latency_ms       DOUBLE PRECISION NOT NULL,
    message          TEXT NOT NULL,
    trace_id         UUID NOT NULL,
    anomaly_injected BOOLEAN NOT NULL DEFAULT FALSE,
    ingested_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, ts)
);

-- Convert to a hypertable partitioned by time (1-day chunks)
SELECT create_hypertable(
    'log_events', 'ts',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

-- Query acceleration indexes
CREATE INDEX IF NOT EXISTS idx_log_events_service_ts
    ON log_events (service, ts DESC);

CREATE INDEX IF NOT EXISTS idx_log_events_level_ts
    ON log_events (level, ts DESC)
    WHERE level = 'ERROR';

CREATE INDEX IF NOT EXISTS idx_log_events_anomaly
    ON log_events (anomaly_injected, ts DESC)
    WHERE anomaly_injected = TRUE;

-- Continuous aggregate: per-minute latency/error stats per service
-- (powers the anomaly-detection engine in Milestone 2 without scanning raw rows)
CREATE MATERIALIZED VIEW IF NOT EXISTS log_events_1min
WITH (timescaledb.continuous) AS
SELECT
    time_bucket('1 minute', ts) AS bucket,
    service,
    count(*) AS event_count,
    count(*) FILTER (WHERE level = 'ERROR') AS error_count,
    avg(latency_ms) AS avg_latency_ms,
    percentile_cont(0.95) WITHIN GROUP (ORDER BY latency_ms) AS p95_latency_ms,
    max(latency_ms) AS max_latency_ms
FROM log_events
GROUP BY bucket, service
WITH NO DATA;

SELECT add_continuous_aggregate_policy(
    'log_events_1min',
    start_offset => INTERVAL '10 minutes',
    end_offset => INTERVAL '1 minute',
    schedule_interval => INTERVAL '1 minute',
    if_not_exists => TRUE
);