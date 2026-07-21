-- SentinelOps: detected anomalies table

CREATE TABLE IF NOT EXISTS anomalies (
    id                UUID NOT NULL DEFAULT gen_random_uuid(),
    detected_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    bucket_ts         TIMESTAMPTZ NOT NULL,
    service           TEXT NOT NULL,
    anomaly_type      TEXT NOT NULL,           -- 'latency_spike' | 'error_burst'
    metric_value      DOUBLE PRECISION NOT NULL,
    baseline_mean     DOUBLE PRECISION NOT NULL,
    baseline_stddev   DOUBLE PRECISION NOT NULL,
    z_score           DOUBLE PRECISION NOT NULL,
    severity          TEXT NOT NULL,           -- 'warning' | 'critical'
    PRIMARY KEY (id, detected_at)
);

SELECT create_hypertable(
    'anomalies', 'detected_at',
    chunk_time_interval => INTERVAL '1 day',
    if_not_exists => TRUE
);

CREATE INDEX IF NOT EXISTS idx_anomalies_service_ts
    ON anomalies (service, detected_at DESC);

CREATE INDEX IF NOT EXISTS idx_anomalies_severity
    ON anomalies (severity, detected_at DESC)
    WHERE severity = 'critical';