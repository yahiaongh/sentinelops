# SentinelOps Dashboard

Real-time operations dashboard for the SentinelOps platform, built with Next.js 16 (App Router), TypeScript, and Tailwind CSS.

## What it does

- **Service health grid** — per-service event counts, error rates, and latency percentiles, polled every 15s
- **Anomaly feed** — scrolling log-tail of detected anomalies from the Rust anomaly engine, polled every 10s
- **Incident query terminal** — natural-language questions answered by the LLM/RAG service, submitted as an async job and polled to completion (local Ollama inference can take 30-90s)

All three panels talk to Next.js API routes (`app/api/*`), which query TimescaleDB directly (via `lib/db.ts`) or proxy to the `llm-query-python` service — credentials and internal service URLs never reach the browser.

## Design system

IBM Plex Sans (UI) + IBM Plex Mono (data/metrics/logs), with a dark graphite/teal "SRE command center" palette defined in `tailwind.config.ts`. See the root [README](../../README.md) for the full rationale.

## Local development

```bash
npm install
DATABASE_URL="postgresql://sentinelops:devpassword@localhost:5432/sentinelops" \
LLM_SERVICE_URL="http://localhost:9300" \
npm run dev
```

Requires the rest of the SentinelOps stack running (see [infra/docker-compose.yml](../../infra/docker-compose.yml)) for the API routes to return real data.

## Testing

```bash
npm run lint   # ESLint (flat config, eslint-config-next)
npm test       # Vitest — API route logic
npm run build  # Full production build (Next.js standalone output)
```

## Deployment

Built as a multi-stage Docker image (`Dockerfile`) using Next.js's `standalone` output mode. Runs as the `dashboard` service in [infra/docker-compose.yml](../../infra/docker-compose.yml), exposed on port 3000.