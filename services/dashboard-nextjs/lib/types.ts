export interface Anomaly {
  id: string;
  detected_at: string;
  bucket_ts: string;
  service: string;
  anomaly_type: string;
  metric_value: number;
  baseline_mean: number;
  baseline_stddev: number;
  z_score: number;
  severity: "warning" | "critical";
}

export interface ServiceHealth {
  service: string;
  total_events: number;
  total_errors: number;
  error_rate: number;
  avg_p95_latency_ms: number | null;
  max_latency_ms: number | null;
}

export type JobStatus = "pending" | "running" | "complete" | "failed";

export interface QueryJob {
  job_id: string;
  status: JobStatus;
  query: string;
  created_at: string;
  completed_at: string | null;
  result: string | null;
  error: string | null;
}