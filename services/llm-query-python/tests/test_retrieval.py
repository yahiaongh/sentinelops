from datetime import UTC, datetime

from app.retrieval import (
    AnomalyContext,
    RetrievedContext,
    ServiceStatsContext,
    format_context_for_prompt,
)


def make_window():
    return (
        datetime(2026, 7, 22, 6, 0, tzinfo=UTC),
        datetime(2026, 7, 22, 12, 0, tzinfo=UTC),
    )


def test_format_context_empty_window():
    start, end = make_window()
    context = RetrievedContext(window_start=start, window_end=end, anomalies=[], service_stats=[])
    text = format_context_for_prompt(context)
    assert "No anomalies detected" in text
    assert context.is_empty()


def test_format_context_includes_service_stats_and_error_rate():
    start, end = make_window()
    context = RetrievedContext(
        window_start=start,
        window_end=end,
        anomalies=[],
        service_stats=[
            ServiceStatsContext(
                service="checkout",
                total_events=200,
                total_errors=20,
                avg_p95_latency_ms=150.0,
                max_latency_ms=300.0,
            )
        ],
    )
    text = format_context_for_prompt(context)
    assert "checkout" in text
    # 20/200 = 10.0% error rate — verifies the computed field, not just passthrough
    assert "10.0% error rate" in text
    assert not context.is_empty()


def test_format_context_includes_anomalies_with_severity():
    start, end = make_window()
    context = RetrievedContext(
        window_start=start,
        window_end=end,
        anomalies=[
            AnomalyContext(
                detected_at=end,
                bucket_ts=end,
                service="payments",
                anomaly_type="error_burst",
                metric_value=0.5,
                baseline_mean=0.12,
                z_score=3.29,
                severity="warning",
            )
        ],
        service_stats=[],
    )
    text = format_context_for_prompt(context)
    assert "[WARNING]" in text
    assert "payments" in text
    assert "error_burst" in text
    assert "z_score=3.29" in text


def test_format_context_handles_zero_events_without_division_error():
    start, end = make_window()
    context = RetrievedContext(
        window_start=start,
        window_end=end,
        anomalies=[],
        service_stats=[
            ServiceStatsContext(
                service="idle-service",
                total_events=0,
                total_errors=0,
                avg_p95_latency_ms=None,
                max_latency_ms=None,
            )
        ],
    )
    # Must not raise ZeroDivisionError
    text = format_context_for_prompt(context)
    assert "0.0% error rate" in text
