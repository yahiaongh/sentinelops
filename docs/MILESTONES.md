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

## ✅ Milestone 3 — Python LLM/RAG query service
- FastAPI service exposing natural-language queries over detected anomalies and service stats
- Retrieval grounded in TimescaleDB (`anomalies`, `log_events_1min`) — verified the model's numbers match real rows exactly, not hallucinated
- Async job queue (`202 Accepted` + poll) instead of a blocking request, designed around measured local-hardware inference latency (~2.8 tokens/sec on CPU-only Ollama)
- Diagnosed and fixed a real Ollama networking issue (host-only binding blocking container access) via a systemd override

## ✅ Milestone 4 — Next.js dashboard
- Real-time anomaly feed (scrolling log-tail style), service health grid, and an LLM query terminal
- Custom design system (IBM Plex Sans/Mono, SRE command-center palette) — not a default template look
- Server-side API routes keep database credentials and the LLM service URL out of the browser
- Mid-build security upgrade to Next.js 16 + ESLint 9 to close several known CVEs before writing any app code
- Extended CI to 9 jobs

## ✅ Milestone 5 — Observability stack
- Prometheus scraping ingestion-go, anomaly-rust, and llm-query-python on 15s intervals
- Grafana with provisioned datasource and dashboard (committed JSON, not clicked together — reproducible from a fresh `docker compose up`)
- 9-panel dashboard covering ingestion throughput/errors, batch performance, anomaly detection rate and severity breakdown, and LLM query latency/failure rate

## ✅ Milestone 6 — Kubernetes deployment
- k3d local cluster (1 server + 1 agent node), kubectl's built-in Kustomize for manifest organization
- Core pipeline deployed and verified end-to-end inside the cluster: log-producer → Redpanda → ingestion-go → TimescaleDB → anomaly-rust, with real multi-node pod scheduling
- Kubernetes Secrets for credentials (never committed with real values)
- Known, documented scope boundary: llm-query-python and dashboard-nextjs are not yet deployed to K8s, since both depend on a host-native Ollama instance reachable via `host.docker.internal` in Compose — that networking path doesn't translate directly into a k3d cluster. Noted explicitly rather than silently skipped; see infra/k8s/README.md.

## Known gaps (tracked, not blocking)
- No automated tests for the Next.js dashboard (API routes or components) — only manual/visual verification so far
- `services/dashboard-nextjs/README.md` is still `create-next-app`'s default boilerplate, not project-specific docs

## Recently closed
- Log producer was assigning `level` independently of `status_code` (a 200 response could randomly be logged as ERROR) — root cause of inflated error-rate metrics. Fixed by deriving level from status_code.
- A follow-up fix (`ANOMALY_PROBABILITY` 0.02 → 0.0015) never actually took effect due to a copy/paste error that applied it to the wrong service's environment block. Corrected and verified live: error rates dropped from ~30-50% to ~0-15%, matching expected burst-driven anomaly frequency.
- `anomaly-rust`: added `detect()` end-to-end tests, including a regression guard for the directional-severity bug
- `anomaly-rust`: EWMA baseline now persisted in `AnomalyRecord` for error-burst anomalies
- `ingestion-go`: added mocked-pool unit tests for `store.go`'s write path (`pgxmock`), alongside existing integration tests