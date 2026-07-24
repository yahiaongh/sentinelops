# Kubernetes Deployment

Local Kubernetes deployment of SentinelOps's core pipeline (log producer → Redpanda → Go ingestion → TimescaleDB → Rust anomaly engine), using [k3d](https://k3d.io/) and `kubectl`'s built-in Kustomize support.

## What's deployed here

- `timescaledb`, `redpanda` — infrastructure
- `log-producer`, `ingestion-go`, `anomaly-rust` — the three services that make up the core detection pipeline

## Known limitation

`llm-query-python` and `dashboard-nextjs` are **not yet deployed to Kubernetes**. Both depend on Ollama, which runs natively on the host machine in the Docker Compose setup (via `host.docker.internal` + a systemd `OLLAMA_HOST=0.0.0.0` override — see the root README). That networking path doesn't translate directly into a k3d cluster, which is network-isolated from the host differently than a single Docker container is. Deploying these two services would need either：
- a proper Ollama Kubernetes deployment (running Ollama itself as a pod, which needs significant CPU/RAM and ideally GPU passthrough), or
- a documented cluster-to-host networking bridge specific to k3d.

This is left as a deliberate scope boundary for this project, not an oversight — noted here rather than silently skipped.

## Quickstart

```bash
# 1. Create the cluster
k3d cluster create sentinelops --agents 1 --wait

# 2. Create the namespace and secret
kubectl apply -f base/namespace.yaml
kubectl create secret generic sentinelops-secrets \
  --namespace sentinelops \
  --from-literal=POSTGRES_USER=sentinelops \
  --from-literal=POSTGRES_PASSWORD=devpassword \
  --from-literal=POSTGRES_DB=sentinelops \
  --from-literal=DATABASE_URL=postgres://sentinelops:devpassword@timescaledb:5432/sentinelops

# 3. Deploy infrastructure and apply the schema
kubectl apply -f base/timescaledb.yaml
kubectl apply -f base/redpanda.yaml
kubectl -n sentinelops exec -i deploy/timescaledb -- psql -U sentinelops -d sentinelops < ../sql/001_init_hypertable.sql
kubectl -n sentinelops exec -i deploy/timescaledb -- psql -U sentinelops -d sentinelops < ../sql/002_anomalies_table.sql

# 4. Build and import service images (no registry needed for local dev)
docker build -t sentinelops/log-producer:local ../../services/log-producer
docker build -t sentinelops/ingestion-go:local ../../services/ingestion-go
docker build -t sentinelops/anomaly-rust:local ../../services/anomaly-rust
k3d image import sentinelops/log-producer:local sentinelops/ingestion-go:local sentinelops/anomaly-rust:local -c sentinelops

# 5. Deploy the services
kubectl apply -f base/log-producer.yaml -f base/ingestion-go.yaml -f base/anomaly-rust.yaml

# 6. Verify
kubectl -n sentinelops get pods
kubectl -n sentinelops exec deploy/timescaledb -- psql -U sentinelops -d sentinelops -c "SELECT count(*) FROM log_events;"
```

## Teardown

```bash
k3d cluster delete sentinelops
```