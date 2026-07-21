use axum::response::IntoResponse;
use prometheus::{Encoder, IntCounter, IntCounterVec, IntGauge, Opts, TextEncoder};
use std::sync::OnceLock;

pub static DETECTION_CYCLES_TOTAL: OnceLockMetric<IntCounter> = OnceLockMetric::new();
pub static DB_QUERY_ERRORS: OnceLockMetric<IntCounter> = OnceLockMetric::new();
pub static BUCKETS_FETCHED: OnceLockMetric<IntGauge> = OnceLockMetric::new();
pub static ANOMALIES_DETECTED: OnceLockMetric<IntCounterVec> = OnceLockMetric::new();

pub struct OnceLockMetric<T> {
    cell: OnceLock<T>,
}

impl<T> OnceLockMetric<T> {
    const fn new() -> Self {
        Self {
            cell: OnceLock::new(),
        }
    }
    pub fn get(&self) -> &T {
        self.cell
            .get()
            .expect("metric accessed before register() was called")
    }
}

impl std::ops::Deref for OnceLockMetric<IntCounter> {
    type Target = IntCounter;
    fn deref(&self) -> &Self::Target {
        self.get()
    }
}
impl std::ops::Deref for OnceLockMetric<IntGauge> {
    type Target = IntGauge;
    fn deref(&self) -> &Self::Target {
        self.get()
    }
}
impl std::ops::Deref for OnceLockMetric<IntCounterVec> {
    type Target = IntCounterVec;
    fn deref(&self) -> &Self::Target {
        self.get()
    }
}

pub fn register() {
    let cycles = IntCounter::with_opts(Opts::new(
        "sentinelops_anomaly_detection_cycles_total",
        "Total number of detection cycles run.",
    ))
    .unwrap();
    let db_errors = IntCounter::with_opts(Opts::new(
        "sentinelops_anomaly_db_query_errors_total",
        "Total number of failed database queries.",
    ))
    .unwrap();
    let buckets_fetched = IntGauge::with_opts(Opts::new(
        "sentinelops_anomaly_buckets_fetched",
        "Number of per-service-minute buckets fetched in the last cycle.",
    ))
    .unwrap();
    let anomalies_detected = IntCounterVec::new(
        Opts::new(
            "sentinelops_anomaly_detected_total",
            "Total number of anomalies detected, by service, type, and severity.",
        ),
        &["service", "anomaly_type", "severity"],
    )
    .unwrap();

    prometheus::default_registry()
        .register(Box::new(cycles.clone()))
        .unwrap();
    prometheus::default_registry()
        .register(Box::new(db_errors.clone()))
        .unwrap();
    prometheus::default_registry()
        .register(Box::new(buckets_fetched.clone()))
        .unwrap();
    prometheus::default_registry()
        .register(Box::new(anomalies_detected.clone()))
        .unwrap();

    DETECTION_CYCLES_TOTAL.cell.set(cycles).ok();
    DB_QUERY_ERRORS.cell.set(db_errors).ok();
    BUCKETS_FETCHED.cell.set(buckets_fetched).ok();
    ANOMALIES_DETECTED.cell.set(anomalies_detected).ok();
}

pub async fn handler() -> impl IntoResponse {
    let encoder = TextEncoder::new();
    let metric_families = prometheus::default_registry().gather();
    let mut buffer = Vec::new();
    encoder.encode(&metric_families, &mut buffer).unwrap();
    let content_type = encoder.format_type().to_string();
    ([(axum::http::header::CONTENT_TYPE, content_type)], buffer)
}
