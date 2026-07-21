# Milestones

Status and scope for each phase of SentinelOps. Dates are approximate; this is a portfolio project built incrementally, not on a fixed sprint schedule.

## ✅ Milestone 1 — Infrastructure foundation
- Redpanda (Kafka-compatible event bus) with dual internal/external listeners
- TimescaleDB with hypertable schema and continuous aggregates
- Redis (provisioned, not yet consumed by a service)
- Python log producer simulating 4 microservices with realistic anomaly injection
- Docker Compose stack with health checks

## ✅ Milestone 1b — Go ingestion service
- Kafka consumer with batched writes to TimescaleDB (`pgx.Batch`, idempotent via `ON CONFLICT`)
- Self-migrating schema via `go:embed`
- Full Prometheus instrumentation (`/metrics`, `/healthz`)
- Distroless multi-stage Docker build

## ✅ Milestone 1c — CI/CD pipeline
- GitHub Actions: lint (`golangci-lint`, `ruff`), test, Docker build validation
- Fan-in `CI Success` required status check
- Branch protection on `main` (no force-push, no delete, status checks required)
- Adopted PR-based workflow after discovering direct-push conflicts with required status checks

## ✅ Milestone 2 — Rust anomaly-detection engine
- Polls the `log_events_1min` continuous aggregate every 15s
- Z-score detection on p95 and max latency (directional — only flags getting slower)
- EWMA-based error-rate deviation detection (bidirectional — flags spikes and suspicious drops)
- Self-migrating schema (same pattern as the Go service, via `include_str!`)
- Caught and fixed a real directional-severity bug in detection logic, with regression tests
- Extended CI to 7 jobs: added Rust lint/test/fmt and a third Docker image build

## 🚧 Milestone 3 — Python LLM/RAG query service (next)
- FastAPI service exposing natural-language queries over detected anomalies and log context
- RAG pipeline: retrieve relevant log/anomaly context, summarize via LLM
- Incident summarization ("what happened to checkout between 2-3am?")

## 🚧 Milestone 4 — Next.js dashboard
- Real-time anomaly feed
- Natural-language query chat interface
- Service health overview

## 🚧 Milestone 5 — Observability stack
- Prometheus scraping all services (currently exposed but not scraped)
- Grafana dashboards

## 🚧 Milestone 6 — Kubernetes deployment
- Manifests for all services
- Local cluster deployment (k3d/kind) as a production-shaped deploy path

## Known gaps (tracked, not blocking)
- `anomaly-rust`: `detect()` has no end-to-end unit test (only its component math functions are tested)
- `anomaly-rust`: EWMA baseline isn't persisted in `AnomalyRecord` for error-burst anomalies
- `ingestion-go`: `store.go` write path is integration-tested but lacks a unit test with a mocked pool
- No Grafana/Prometheus scrape config yet — metrics are exposed but not yet centrally collected