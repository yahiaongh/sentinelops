use std::env;
use std::time::Duration;

#[derive(Debug, Clone)]
pub struct Config {
    pub database_url: String,
    pub poll_interval: Duration,
    pub baseline_window_minutes: i64,
    pub z_score_warning: f64,
    pub z_score_critical: f64,
    pub ewma_alpha: f64,
    pub metrics_addr: String,
}

impl Config {
    pub fn load() -> Self {
        Self {
            database_url: env::var("DATABASE_URL").unwrap_or_else(|_| {
                "postgres://sentinelops:devpassword@localhost:5432/sentinelops".to_string()
            }),
            poll_interval: Duration::from_secs(get_env_u64("POLL_INTERVAL_SECS", 15)),
            baseline_window_minutes: get_env_i64("BASELINE_WINDOW_MINUTES", 60),
            z_score_warning: get_env_f64("Z_SCORE_WARNING", 2.5),
            z_score_critical: get_env_f64("Z_SCORE_CRITICAL", 4.0),
            ewma_alpha: get_env_f64("EWMA_ALPHA", 0.3),
            metrics_addr: env::var("METRICS_ADDR").unwrap_or_else(|_| "0.0.0.0:9200".to_string()),
        }
    }
}

fn get_env_u64(key: &str, default: u64) -> u64 {
    env::var(key)
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(default)
}

fn get_env_i64(key: &str, default: i64) -> i64 {
    env::var(key)
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(default)
}

fn get_env_f64(key: &str, default: f64) -> f64 {
    env::var(key)
        .ok()
        .and_then(|v| v.parse().ok())
        .unwrap_or(default)
}
