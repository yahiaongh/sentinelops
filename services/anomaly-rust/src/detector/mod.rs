use crate::store::{AnomalyRecord, MinuteBucket};
use std::collections::HashMap;

/// Computes population mean and standard deviation. Returns (mean, stddev).
/// stddev is 0.0 for fewer than 2 samples, which callers must guard against
/// (a z-score against zero variance is undefined and always "infinitely anomalous",
/// which is noise, not signal).
fn mean_stddev(values: &[f64]) -> (f64, f64) {
    let n = values.len() as f64;
    if n == 0.0 {
        return (0.0, 0.0);
    }
    let mean = values.iter().sum::<f64>() / n;
    if n < 2.0 {
        return (mean, 0.0);
    }
    let variance = values.iter().map(|v| (v - mean).powi(2)).sum::<f64>() / (n - 1.0);
    (mean, variance.sqrt())
}

/// Bidirectional severity check — flags large deviations in either direction.
/// Appropriate for metrics where both increases and decreases are meaningful
/// (e.g. error rate: a sudden drop can indicate a silently-failing health check).
fn severity_for(z: f64, warning: f64, critical: f64) -> Option<&'static str> {
    severity_for_directional(z.abs(), warning, critical)
}

/// One-directional severity check — only flags positive deviations.
/// Appropriate for metrics where only "worse" has one sign (e.g. latency:
/// a lower-than-baseline max latency is good news, not an anomaly).
fn severity_for_directional(z: f64, warning: f64, critical: f64) -> Option<&'static str> {
    if z >= critical {
        Some("critical")
    } else if z >= warning {
        Some("warning")
    } else {
        None
    }
}

/// Tracks a per-service exponentially-weighted moving average of error rate.
/// EWMA is preferred over a flat-window average for error rate specifically
/// because it adapts smoothly as traffic patterns genuinely shift over time,
/// rather than treating a 3-day-old spike and a 3-minute-old spike as equally
/// relevant "history".
#[derive(Default)]
pub struct ErrorRateBaseline {
    ewma: HashMap<String, f64>,
}

impl ErrorRateBaseline {
    pub fn update_and_check(
        &mut self,
        service: &str,
        current_error_rate: f64,
        alpha: f64,
        warning: f64,
        critical: f64,
    ) -> Option<(f64, &'static str)> {
        let baseline = *self.ewma.get(service).unwrap_or(&current_error_rate);
        let deviation = current_error_rate - baseline;

        // Update EWMA regardless of whether this reading is anomalous, so the
        // baseline keeps tracking the service's real long-term behavior.
        let updated = alpha * current_error_rate + (1.0 - alpha) * baseline;
        self.ewma.insert(service.to_string(), updated);

        // Deviation is expressed as a pseudo-z-score against the EWMA baseline
        // scaled by baseline magnitude, since error rate is bounded [0,1] and
        // a raw stddev-based z-score is less meaningful for a ratio like this.
        let scale = baseline.max(0.01); // avoid division blowing up near zero
        let z = deviation / scale;

        severity_for(z, warning, critical).map(|sev| (z, sev))
    }
}

pub struct DetectionConfig {
    pub z_score_warning: f64,
    pub z_score_critical: f64,
    pub ewma_alpha: f64,
}

/// Runs latency z-score detection and error-rate EWMA detection across all
/// services present in `buckets`. `buckets` must be ordered oldest-first per
/// service (as returned by Store::fetch_recent_buckets) — the last bucket per
/// service is treated as "current", everything before it as "baseline".
pub fn detect(
    buckets: &[MinuteBucket],
    error_baseline: &mut ErrorRateBaseline,
    cfg: &DetectionConfig,
) -> Vec<AnomalyRecord> {
    let mut by_service: HashMap<&str, Vec<&MinuteBucket>> = HashMap::new();
    for b in buckets {
        by_service.entry(&b.service).or_default().push(b);
    }

    let mut anomalies = Vec::new();

    for (service, series) in by_service {
        if series.len() < 3 {
            // Not enough history yet to establish a meaningful baseline.
            continue;
        }

        let (history, current) = series.split_at(series.len() - 1);
        let current = current[0];

        // --- Latency z-score detection (p95-based, catches sustained shifts).
        // Directional: only flags getting SLOWER than baseline. ---
        let p95_history: Vec<f64> = history.iter().filter_map(|b| b.p95_latency_ms).collect();
        if let Some(current_p95) = current.p95_latency_ms {
            let (mean, stddev) = mean_stddev(&p95_history);
            if stddev > 0.0 {
                let z = (current_p95 - mean) / stddev;
                if let Some(sev) =
                    severity_for_directional(z, cfg.z_score_warning, cfg.z_score_critical)
                {
                    anomalies.push(AnomalyRecord {
                        bucket_ts: current.bucket,
                        service: service.to_string(),
                        anomaly_type: "latency_spike".to_string(),
                        metric_value: current_p95,
                        baseline_mean: mean,
                        baseline_stddev: stddev,
                        z_score: z,
                        severity: sev.to_string(),
                    });
                }
            }
        }

        // --- Max-latency ceiling check (catches single catastrophic outliers
        // that a p95-based z-score can miss entirely, since p95 by definition
        // discounts the top 5% of requests). Directional: only flags getting
        // SLOWER than baseline. ---
        let max_latency_history: Vec<f64> =
            history.iter().filter_map(|b| b.max_latency_ms).collect();
        if let Some(current_max) = current.max_latency_ms {
            let (mean, stddev) = mean_stddev(&max_latency_history);
            if stddev > 0.0 {
                let z = (current_max - mean) / stddev;
                if let Some(sev) =
                    severity_for_directional(z, cfg.z_score_warning, cfg.z_score_critical)
                {
                    anomalies.push(AnomalyRecord {
                        bucket_ts: current.bucket,
                        service: service.to_string(),
                        anomaly_type: "max_latency_outlier".to_string(),
                        metric_value: current_max,
                        baseline_mean: mean,
                        baseline_stddev: stddev,
                        z_score: z,
                        severity: sev.to_string(),
                    });
                }
            }
        }

        // --- Error rate EWMA detection (bidirectional: both spikes and
        // suspicious drops toward zero are worth flagging) ---
        if current.event_count > 0 {
            let current_error_rate = current.error_count as f64 / current.event_count as f64;
            if let Some((z, sev)) = error_baseline.update_and_check(
                service,
                current_error_rate,
                cfg.ewma_alpha,
                cfg.z_score_warning,
                cfg.z_score_critical,
            ) {
                anomalies.push(AnomalyRecord {
                    bucket_ts: current.bucket,
                    service: service.to_string(),
                    anomaly_type: "error_burst".to_string(),
                    metric_value: current_error_rate,
                    baseline_mean: 0.0,
                    baseline_stddev: 0.0,
                    z_score: z,
                    severity: sev.to_string(),
                });
            }
        }
    }

    anomalies
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn mean_stddev_basic() {
        let (mean, stddev) = mean_stddev(&[10.0, 12.0, 11.0, 13.0, 9.0]);
        assert!((mean - 11.0).abs() < 1e-9);
        assert!(stddev > 0.0);
    }

    #[test]
    fn mean_stddev_single_value_has_zero_stddev() {
        let (mean, stddev) = mean_stddev(&[42.0]);
        assert_eq!(mean, 42.0);
        assert_eq!(stddev, 0.0);
    }

    #[test]
    fn mean_stddev_empty_is_zero() {
        let (mean, stddev) = mean_stddev(&[]);
        assert_eq!(mean, 0.0);
        assert_eq!(stddev, 0.0);
    }

    #[test]
    fn severity_thresholds_bidirectional() {
        assert_eq!(severity_for(1.0, 2.5, 4.0), None);
        assert_eq!(severity_for(2.6, 2.5, 4.0), Some("warning"));
        assert_eq!(severity_for(4.5, 2.5, 4.0), Some("critical"));
        assert_eq!(severity_for(-4.5, 2.5, 4.0), Some("critical"));
    }

    #[test]
    fn severity_thresholds_directional_ignores_negative() {
        // A large negative deviation (e.g. latency lower than baseline) must
        // NOT be flagged by the directional check — it's good news, not an anomaly.
        assert_eq!(severity_for_directional(-4.5, 2.5, 4.0), None);
        assert_eq!(severity_for_directional(-10.0, 2.5, 4.0), None);
        assert_eq!(severity_for_directional(2.6, 2.5, 4.0), Some("warning"));
        assert_eq!(severity_for_directional(4.5, 2.5, 4.0), Some("critical"));
    }

    #[test]
    fn error_rate_baseline_flags_spike_and_adapts() {
        let mut baseline = ErrorRateBaseline::default();
        // Prime the baseline with several normal low-error readings.
        for _ in 0..5 {
            baseline.update_and_check("checkout", 0.02, 0.3, 2.5, 4.0);
        }
        // A sudden spike should be flagged against the now-stable low baseline.
        let result = baseline.update_and_check("checkout", 0.6, 0.3, 2.5, 4.0);
        assert!(result.is_some(), "expected spike to be flagged");
    }
}
