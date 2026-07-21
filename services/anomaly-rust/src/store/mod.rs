use chrono::{DateTime, Utc};
use sqlx::postgres::PgPoolOptions;
use sqlx::{FromRow, PgPool};
use uuid::Uuid;

#[derive(Debug, Clone, FromRow)]
pub struct MinuteBucket {
    pub bucket: DateTime<Utc>,
    pub service: String,
    pub event_count: i64,
    pub error_count: i64,
    // Reserved for Milestone 3 (LLM incident summaries) — not yet consumed
    // by detection logic, which uses p95/max instead of the mean.
    #[allow(dead_code)]
    pub avg_latency_ms: Option<f64>,
    pub p95_latency_ms: Option<f64>,
    pub max_latency_ms: Option<f64>,
}

#[derive(Debug, Clone)]
pub struct AnomalyRecord {
    pub bucket_ts: DateTime<Utc>,
    pub service: String,
    pub anomaly_type: String,
    pub metric_value: f64,
    pub baseline_mean: f64,
    pub baseline_stddev: f64,
    pub z_score: f64,
    pub severity: String,
}

pub struct Store {
    pool: PgPool,
}

impl Store {
    pub async fn connect(database_url: &str) -> anyhow::Result<Self> {
        let pool = PgPoolOptions::new()
            .max_connections(5)
            .connect(database_url)
            .await?;
        Ok(Self { pool })
    }

    /// Applies the embedded schema migration. Idempotent: safe to run on
    /// every startup regardless of whether the volume is fresh or existing.
    pub async fn migrate(&self) -> anyhow::Result<()> {
        let sql = include_str!("../../migrations/002_anomalies_table.sql");
        sqlx::raw_sql(sql).execute(&self.pool).await?;
        Ok(())
    }

    /// Fetches the last `window_minutes` of per-service, per-minute stats
    /// from the log_events_1min continuous aggregate, ordered oldest-first
    /// so the caller can treat the last row per service as "current" and
    /// everything before it as "baseline history".
    pub async fn fetch_recent_buckets(
        &self,
        window_minutes: i64,
    ) -> anyhow::Result<Vec<MinuteBucket>> {
        let rows = sqlx::query_as::<_, MinuteBucket>(
            r#"
            SELECT bucket, service, event_count, error_count,
                   avg_latency_ms, p95_latency_ms, max_latency_ms
            FROM log_events_1min
            WHERE bucket >= now() - ($1 || ' minutes')::interval
            ORDER BY service, bucket ASC
            "#,
        )
        .bind(window_minutes.to_string())
        .fetch_all(&self.pool)
        .await?;
        Ok(rows)
    }

    pub async fn insert_anomaly(&self, a: &AnomalyRecord) -> anyhow::Result<()> {
        sqlx::query(
            r#"
            INSERT INTO anomalies
                (id, bucket_ts, service, anomaly_type, metric_value, baseline_mean, baseline_stddev, z_score, severity)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
            "#,
        )
        .bind(Uuid::new_v4())
        .bind(a.bucket_ts)
        .bind(&a.service)
        .bind(&a.anomaly_type)
        .bind(a.metric_value)
        .bind(a.baseline_mean)
        .bind(a.baseline_stddev)
        .bind(a.z_score)
        .bind(&a.severity)
        .execute(&self.pool)
        .await?;
        Ok(())
    }
}
