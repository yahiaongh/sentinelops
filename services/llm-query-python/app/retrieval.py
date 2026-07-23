"""Retrieval layer: pulls grounded context from TimescaleDB for a query.

This is the "R" in RAG — we never let the LLM invent incident data. Every
answer is grounded in rows actually read from anomalies / log_events_1min.
"""

from datetime import datetime, timedelta, timezone

import asyncpg
from pydantic import BaseModel


class AnomalyContext(BaseModel):
    detected_at: datetime
    bucket_ts: datetime
    service: str
    anomaly_type: str
    metric_value: float
    baseline_mean: float
    z_score: float
    severity: str


class ServiceStatsContext(BaseModel):
    service: str
    total_events: int
    total_errors: int
    avg_p95_latency_ms: float | None
    max_latency_ms: float | None


class RetrievedContext(BaseModel):
    window_start: datetime
    window_end: datetime
    anomalies: list[AnomalyContext]
    service_stats: list[ServiceStatsContext]

    def is_empty(self) -> bool:
        return not self.anomalies and not self.service_stats


async def retrieve_context(
    pool: asyncpg.Pool,
    lookback_hours: int,
    service_filter: str | None = None,
) -> RetrievedContext:
    """Fetches recent anomalies and per-service aggregate stats within the
    lookback window. If service_filter is given, scopes both queries to
    that service; otherwise returns a system-wide view.
    """
    window_end = datetime.now(timezone.utc)
    window_start = window_end - timedelta(hours=lookback_hours)

    anomaly_query = """
        SELECT detected_at, bucket_ts, service, anomaly_type,
               metric_value, baseline_mean, z_score, severity
        FROM anomalies
        WHERE detected_at >= $1
          AND ($2::text IS NULL OR service = $2)
        ORDER BY detected_at DESC
        LIMIT 50
    """
    stats_query = """
        SELECT service,
               sum(event_count) AS total_events,
               sum(error_count) AS total_errors,
               avg(p95_latency_ms) AS avg_p95_latency_ms,
               max(max_latency_ms) AS max_latency_ms
        FROM log_events_1min
        WHERE bucket >= $1
          AND ($2::text IS NULL OR service = $2)
        GROUP BY service
        ORDER BY service
    """

    async with pool.acquire() as conn:
        anomaly_rows = await conn.fetch(anomaly_query, window_start, service_filter)
        stats_rows = await conn.fetch(stats_query, window_start, service_filter)

    anomalies = [
        AnomalyContext(
            detected_at=r["detected_at"],
            bucket_ts=r["bucket_ts"],
            service=r["service"],
            anomaly_type=r["anomaly_type"],
            metric_value=r["metric_value"],
            baseline_mean=r["baseline_mean"],
            z_score=r["z_score"],
            severity=r["severity"],
        )
        for r in anomaly_rows
    ]
    service_stats = [
        ServiceStatsContext(
            service=r["service"],
            total_events=r["total_events"] or 0,
            total_errors=r["total_errors"] or 0,
            avg_p95_latency_ms=r["avg_p95_latency_ms"],
            max_latency_ms=r["max_latency_ms"],
        )
        for r in stats_rows
    ]

    return RetrievedContext(
        window_start=window_start,
        window_end=window_end,
        anomalies=anomalies,
        service_stats=service_stats,
    )


def format_context_for_prompt(context: RetrievedContext) -> str:
    """Renders retrieved context as compact, LLM-readable text. Kept terse
    deliberately — every token here costs generation latency on constrained
    hardware, so we favor a dense table-like format over prose.
    """
    lines = [
        f"Time window: {context.window_start.isoformat()} to {context.window_end.isoformat()}",
        "",
    ]

    if context.service_stats:
        lines.append("Service stats:")
        for s in context.service_stats:
            error_rate = (
                (s.total_errors / s.total_events * 100) if s.total_events else 0.0
            )
            lines.append(
                f"- {s.service}: {s.total_events} events, "
                f"{error_rate:.1f}% error rate, "
                f"avg p95 latency {s.avg_p95_latency_ms or 0:.0f}ms, "
                f"max latency {s.max_latency_ms or 0:.0f}ms"
            )
        lines.append("")

    if context.anomalies:
        lines.append("Detected anomalies (most recent first):")
        for a in context.anomalies:
            lines.append(
                f"- [{a.severity.upper()}] {a.service} at {a.bucket_ts.isoformat()}: "
                f"{a.anomaly_type}, value={a.metric_value:.2f}, "
                f"baseline={a.baseline_mean:.2f}, z_score={a.z_score:.2f}"
            )
    else:
        lines.append("No anomalies detected in this window.")

    return "\n".join(lines)
