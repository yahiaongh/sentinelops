FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    librdkafka-dev \
    && rm -rf /var/lib/apt/lists/*

COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY producer.py .

ENV KAFKA_BROKERS=redpanda:9092 \
    LOG_TOPIC=service-logs \
    EMIT_INTERVAL_SECONDS=0.2 \
    ANOMALY_PROBABILITY=0.02

CMD ["python", "producer.py"]