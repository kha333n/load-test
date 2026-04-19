# Build tasks

Ordered so each step is testable before moving on.

## Phase 1 — Local app skeleton

- [ ] `go mod init github.com/kha333n/load-test/app`
- [ ] Minimal `main.go` with chi, graceful shutdown, `/healthz`, `/metrics`.
- [ ] Prometheus middleware: `http_request_duration_seconds` + `http_requests_total`.
- [ ] Config via env: `MYSQL_DSN`, `REDIS_ADDR`, `POD_NAME`, `NODE_NAME`, `LISTEN_ADDR`.
- [ ] `docker-compose.yml` at repo root with mysql + redis + app for local smoke test.
- [ ] `make app-run` works, `curl /healthz` returns 200 once deps are up.

## Phase 2 — Endpoints

- [ ] `/test/compute` — SHA-256 hash loop, records total.
- [ ] Redis client in `storage/redis.go`, with timed wrapper.
- [ ] `/test/cache-hit` + seed routine that warms `cache:hit:{1..1000}` on startup.
- [ ] `/test/cache-miss` — GET (miss) then SET.
- [ ] MySQL client in `storage/mysql.go`, with timed wrapper + `items` schema migration on startup.
- [ ] Seed 10k rows into `items` on startup if empty.
- [ ] `/test/db-read` — SELECT by random id.
- [ ] `/test/db-write` — SELECT + UPDATE.
- [ ] `/test/combined` — cache-aside over mysql.
- [ ] All endpoints emit sub-latency histograms as specified in [requirements.md](requirements.md).

## Phase 3 — Container + k8s baseline

- [ ] Multi-stage `Dockerfile`, final image `FROM gcr.io/distroless/static-debian12`.
- [ ] Push to `ghcr.io/kha333n/load-test-app:latest`.
- [ ] `k8s/namespace.yaml` for `load-test`.
- [ ] `k8s/mysql.yaml` — Deployment + Service + PVC, `nodeSelector: workload-tier=secondary-1`.
- [ ] `k8s/redis.yaml` — Deployment + Service, same nodeSelector.
- [ ] `k8s/mailpit.yaml` — Deployment + Service (1025, 8025), same nodeSelector.
- [ ] `k8s/app.yaml` — Deployment (replicas 1), Service, Downward-API env for pod/node, resource requests+limits, readiness probe on `/healthz`, `topologySpreadConstraints` over hostname.
- [ ] `k8s/hpa.yaml` — HPA min 1 max 4, CPU target 70%.
- [ ] `k8s/ingress.yaml` — hostname `load-test.<cluster-domain>`.
- [ ] Apply in order, verify `curl https://load-test.../healthz`.

## Phase 4 — Observability

- [ ] `k8s/monitoring/kube-state-metrics.yaml` (upstream manifests).
- [ ] `k8s/monitoring/prometheus.yaml` — Deployment + ConfigMap scrape config (load-test-app + kube-state-metrics), PVC for TSDB.
- [ ] `k8s/monitoring/grafana.yaml` — Deployment + ConfigMap datasource + ConfigMap dashboard provider, reads from `grafana/dashboards/load-test.json`.
- [ ] Ingress for Grafana at `grafana.<cluster-domain>` (or just port-forward via `make dash`).
- [ ] Build `grafana/dashboards/load-test.json` with the panels listed in requirements §"What the dashboard must show".

## Phase 5 — Load generator

- [ ] `loadgen/shared.js` — base scenario, weighted endpoint selection, checks.
- [ ] `loadgen/low.js` — steady 20 RPS × 5min.
- [ ] `loadgen/medium.js` — ramp to 200, hold, ramp down.
- [ ] `loadgen/high.js` — ramp to 800, spike to 1200, hold, ramp down.
- [ ] Script to push Grafana annotations on scenario start/end.
- [ ] Makefile targets `load-low`, `load-medium`, `load-high`.

## Phase 6 — Run the actual experiments

- [ ] Run low, record p50/p95/p99 per endpoint and baseline numbers in a results log.
- [ ] Run medium, watch HPA, record scale-up time and latency response.
- [ ] Run high, record saturation behavior, error rate, queue depth.
- [ ] Repeat high with `podAntiAffinity` forcing 1-per-node at 2 replicas — compare.
- [ ] Capture 3 Grafana snapshots per run; file them in `docs/results/`.

## Phase 7 — Report

- [ ] `docs/results/README.md` with concrete numbers:
  - Cross-node MySQL SELECT RTT (primary → secondary).
  - Cross-node Redis GET RTT.
  - Co-location benefit (pod on secondary vs primary).
  - HPA scale-up latency (from breach of CPU threshold to new pod ready).
  - Throughput ceiling at 4 replicas.
