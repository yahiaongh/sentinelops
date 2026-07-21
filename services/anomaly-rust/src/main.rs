mod config;
mod detector;
mod metrics;
mod store;

use axum::routing::get;
use axum::Router;
use detector::{detect, DetectionConfig, ErrorRateBaseline};
use std::sync::Arc;
use tokio::net::TcpListener;
use tokio::time::interval;
use tracing::{error, info, warn};
use tracing_subscriber::EnvFilter;

#[tokio::main]
async fn main() -> anyhow::Result<()> {
    tracing_subscriber::fmt()
        .json()
        .with_env_filter(EnvFilter::from_default_env().add_directive("info".parse()?))
        .init();

    let cfg = config::Config::load();
    info!(?cfg, "loading configuration");

    let store = Arc::new(store::Store::connect(&cfg.database_url).await?);
    info!("connected to TimescaleDB");

    store.migrate().await?;
    info!("schema migration applied successfully");

    metrics::register();

    // Metrics + health HTTP server
    let metrics_addr = cfg.metrics_addr.clone();
    tokio::spawn(async move {
        let app = Router::new()
            .route("/metrics", get(metrics::handler))
            .route("/healthz", get(|| async { "ok" }));
        let listener = match TcpListener::bind(&metrics_addr).await {
            Ok(l) => l,
            Err(e) => {
                error!(error = %e, "failed to bind metrics server");
                return;
            }
        };
        info!(addr = %metrics_addr, "metrics server listening");
        if let Err(e) = axum::serve(listener, app).await {
            error!(error = %e, "metrics server error");
        }
    });

    let detection_cfg = DetectionConfig {
        z_score_warning: cfg.z_score_warning,
        z_score_critical: cfg.z_score_critical,
        ewma_alpha: cfg.ewma_alpha,
    };
    let mut error_baseline = ErrorRateBaseline::default();
    let mut ticker = interval(cfg.poll_interval);

    info!(
        poll_interval = ?cfg.poll_interval,
        baseline_window_minutes = cfg.baseline_window_minutes,
        "sentinelops anomaly engine started"
    );

    loop {
        ticker.tick().await;

        let buckets = match store
            .fetch_recent_buckets(cfg.baseline_window_minutes)
            .await
        {
            Ok(b) => b,
            Err(e) => {
                error!(error = %e, "failed to fetch recent buckets");
                metrics::DB_QUERY_ERRORS.inc();
                continue;
            }
        };

        metrics::BUCKETS_FETCHED.set(buckets.len() as i64);

        let anomalies = detect(&buckets, &mut error_baseline, &detection_cfg);

        for anomaly in &anomalies {
            warn!(
                service = %anomaly.service,
                anomaly_type = %anomaly.anomaly_type,
                z_score = anomaly.z_score,
                severity = %anomaly.severity,
                "anomaly detected"
            );
            metrics::ANOMALIES_DETECTED
                .with_label_values(&[&anomaly.service, &anomaly.anomaly_type, &anomaly.severity])
                .inc();

            if let Err(e) = store.insert_anomaly(anomaly).await {
                error!(error = %e, "failed to persist anomaly");
                metrics::DB_QUERY_ERRORS.inc();
            }
        }

        metrics::DETECTION_CYCLES_TOTAL.inc();
    }
}
