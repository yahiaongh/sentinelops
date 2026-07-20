"""
SentinelOps Log Producer Simulator

Simulates structured JSON logs from multiple fake microservices and streams
them to Redpanda/Kafka. Periodically injects realistic anomalies (latency
spikes, error bursts, auth failure storms) so downstream anomaly-detection
services have real signal to detect.
"""

import json
import logging
import os
import random
import signal
import sys
import time
import uuid
from dataclasses import asdict, dataclass
from datetime import datetime, timezone

from confluent_kafka import Producer
from dotenv import load_dotenv
from faker import Faker

load_dotenv(override=True)

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(name)s: %(message)s",
)
logger = logging.getLogger("log-producer")

KAFKA_BROKERS = os.getenv("KAFKA_BROKERS", "localhost:9092")
TOPIC = os.getenv("LOG_TOPIC", "service-logs")
EMIT_INTERVAL_SECONDS = float(os.getenv("EMIT_INTERVAL_SECONDS", "0.2"))
ANOMALY_PROBABILITY = float(os.getenv("ANOMALY_PROBABILITY", "0.02"))

SERVICES = ["checkout", "auth", "payments", "inventory"]
ENDPOINTS = {
    "checkout": ["/cart/add", "/cart/checkout", "/cart/remove"],
    "auth": ["/login", "/refresh-token", "/logout"],
    "payments": ["/charge", "/refund", "/payment-status"],
    "inventory": ["/stock/check", "/stock/reserve", "/stock/release"],
}
LOG_LEVELS = ["INFO", "INFO", "INFO", "WARN", "ERROR"]

fake = Faker()


@dataclass
class LogEvent:
    event_id: str
    timestamp: str
    service: str
    endpoint: str
    level: str
    status_code: int
    latency_ms: float
    message: str
    trace_id: str
    anomaly_injected: bool


class AnomalyState:
    """Tracks whether we're currently mid-anomaly-burst for a given service."""

    def __init__(self):
        self.active_bursts: dict[str, int] = {}

    def maybe_start_burst(self, service: str) -> None:
        if service not in self.active_bursts and random.random() < ANOMALY_PROBABILITY:
            burst_len = random.randint(20, 60)
            self.active_bursts[service] = burst_len
            logger.warning(f"[ANOMALY INJECTED] Starting burst on '{service}' for {burst_len} events")

    def in_burst(self, service: str) -> bool:
        return service in self.active_bursts

    def consume(self, service: str) -> None:
        if service in self.active_bursts:
            self.active_bursts[service] -= 1
            if self.active_bursts[service] <= 0:
                del self.active_bursts[service]


def build_event(service: str, anomaly_state: AnomalyState) -> LogEvent:
    anomaly_state.maybe_start_burst(service)
    in_burst = anomaly_state.in_burst(service)

    endpoint = random.choice(ENDPOINTS[service])
    trace_id = str(uuid.uuid4())

    if in_burst:
        anomaly_state.consume(service)
        burst_type = random.choice(["latency", "errors", "auth_failures"])
        if burst_type == "latency":
            latency_ms = round(random.uniform(1500, 6000), 2)
            status_code = 200
            level = "WARN"
            message = f"Slow response on {endpoint}"
        elif burst_type == "errors":
            latency_ms = round(random.uniform(50, 400), 2)
            status_code = random.choice([500, 502, 503])
            level = "ERROR"
            message = f"Internal error handling {endpoint}"
        else:
            latency_ms = round(random.uniform(30, 200), 2)
            status_code = 401
            level = "ERROR"
            message = "Authentication failed: invalid or expired token"
        return LogEvent(
            event_id=str(uuid.uuid4()),
            timestamp=datetime.now(timezone.utc).isoformat(),
            service=service,
            endpoint=endpoint,
            level=level,
            status_code=status_code,
            latency_ms=latency_ms,
            message=message,
            trace_id=trace_id,
            anomaly_injected=True,
        )

    # Normal baseline traffic
    latency_ms = round(max(5, random.gauss(120, 40)), 2)
    status_code = 200 if random.random() > 0.03 else random.choice([400, 404])
    level = random.choice(LOG_LEVELS)
    message = f"{service} handled {endpoint} for user {fake.uuid4()[:8]}"

    return LogEvent(
        event_id=str(uuid.uuid4()),
        timestamp=datetime.now(timezone.utc).isoformat(),
        service=service,
        endpoint=endpoint,
        level=level,
        status_code=status_code,
        latency_ms=latency_ms,
        message=message,
        trace_id=trace_id,
        anomaly_injected=False,
    )


def delivery_report(err, msg):
    if err is not None:
        logger.error(f"Delivery failed for record {msg.key()}: {err}")


def main() -> None:
    producer = Producer({
        "bootstrap.servers": KAFKA_BROKERS,
        "client.id": "sentinelops-log-producer",
    })

    anomaly_states = {service: AnomalyState() for service in SERVICES}
    running = True

    def handle_shutdown(signum, frame):
        nonlocal running
        logger.info("Shutdown signal received, flushing producer...")
        running = False

    signal.signal(signal.SIGINT, handle_shutdown)
    signal.signal(signal.SIGTERM, handle_shutdown)

    logger.info(f"Starting log producer -> topic '{TOPIC}' on brokers '{KAFKA_BROKERS}'")

    events_sent = 0
    while running:
        service = random.choice(SERVICES)
        event = build_event(service, anomaly_states[service])

        producer.produce(
            TOPIC,
            key=service.encode("utf-8"),
            value=json.dumps(asdict(event)).encode("utf-8"),
            callback=delivery_report,
        )
        producer.poll(0)
        events_sent += 1

        if events_sent % 100 == 0:
            logger.info(f"Emitted {events_sent} events so far")

        time.sleep(EMIT_INTERVAL_SECONDS)

    producer.flush(10)
    logger.info(f"Producer stopped. Total events emitted: {events_sent}")
    sys.exit(0)


if __name__ == "__main__":
    main()