# SentinelOps

**AI-Powered Distributed Log Intelligence & Anomaly Detection Platform**

SentinelOps ingests real-time logs and metrics from distributed microservices, detects anomalies using statistical and ML-based models, and exposes a natural-language query interface (LLM/RAG) for fast incident triage.

## Architecture

```mermaid
flowchart LR
    A[Microservice Producers] -->|events| B[(Kafka)]
    B --> C[Go Ingestion Service]
    C --> D[(TimescaleDB)]
    C --> E[Rust Anomaly Engine]
    E --> D
    D --> F[Python LLM/RAG Service]
    F --> G[Next.js Dashboard]
    C --> H[Redis Cache]
    F --> H
    subgraph Observability
      I[Prometheus] --> J[Grafana]
    end
    C -.metrics.-> I
    E -.metrics.-> I
    F -.metrics.-> I
```

## Status
🚧 Early development — see [milestones](docs/MILESTONES.md).

## Local Development
See `infra/docker-compose.yml` (coming in Milestone 1).

## License
MIT — see [LICENSE](LICENSE).