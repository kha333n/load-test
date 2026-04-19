# load-test — K8s multi-node latency benchmark

A Go HTTP service deployed on k3s that exposes endpoints hitting MySQL, Redis, and combinations of both. External load generator (k6) drives traffic. Prometheus + Grafana record and visualize latency, throughput, pod count, and scaling events.

**Goal:** measure how much latency to expect when app pods are on one node and MySQL/Redis are on another — and how HPA + pod spreading across nodes affects throughput under low/medium/high load.

## Architecture

- **Target app:** Go 1.23, `chi` router, `database/sql` + `go-sql-driver/mysql`, `go-redis/v9`, `prometheus/client_golang`.
- **Data services:** MySQL 8, Redis 7, Mailpit — all pinned to the secondary node (`workload-tier=secondary-1`) via `nodeSelector`.
- **App deployment:** starts on primary (`workload-tier=primary`), HPA scales up to 4 replicas, `topologySpreadConstraints` spreads pods across both nodes.
- **Observability:** Prometheus scrapes the app's `/metrics`; kube-state-metrics exposes replica counts; Grafana renders dashboards.
- **Load generator:** k6 scripts (low / medium / high) run from outside the cluster, hitting the ingress.

## Why Go (not Laravel)

This is a measurement tool, not a product. Go gives the cleanest signal for infra-level latency because it adds almost no framework overhead. If we later want realistic Laravel numbers, we clone the same endpoint set into a Laravel app and compare — but we don't start there.

## Directory conventions

```
load-test/
├── app/              # Go service under test
│   ├── main.go
│   ├── handlers/     # one file per endpoint family
│   ├── middleware/   # request-id, per-request timing, pod/node labels
│   ├── metrics/      # Prometheus histograms + counters
│   └── storage/      # mysql + redis clients
├── loadgen/          # k6 scripts: low.js, medium.js, high.js, scenarios.js
├── k8s/              # manifests: mysql, redis, mailpit, app, hpa, monitoring
│   └── monitoring/   # prometheus, grafana, kube-state-metrics
├── grafana/          # dashboard JSON (provisioned)
└── docs/
    ├── requirements.md
    ├── architecture.md
    └── tasks.md
```

## Key rules

- **No framework bloat in the target app** — every added dependency should be justified. The whole point is measuring infra, not our code.
- **Every endpoint reports its own sub-latencies** — total is measured by middleware; per-I/O latency (redis, mysql) is captured inline and recorded as separate Prometheus histograms.
- **Labels on every metric:** `endpoint`, `pod`, `node`, `outcome` (e.g. `hit`/`miss`/`ok`/`err`). `pod` and `node` come from the Downward API via env vars (`POD_NAME`, `NODE_NAME`).
- **Data services pinned to secondary node** so we're always measuring the cross-node case when the app pod lands on primary.
- **No stored procedures, no ORMs** — raw SQL via `database/sql`. Keep the DB path as thin as possible.
- **k6 scripts define scenarios, not loops** — use k6 `scenarios` with `ramping-vus` / `constant-arrival-rate` so load shape is reproducible.

## Endpoints (v1)

| Path | Does | Records |
|---|---|---|
| `GET /test/compute` | CPU-bound hash loop, no I/O | total |
| `GET /test/cache-hit` | Redis GET on pre-warmed key | total, redis |
| `GET /test/cache-miss` | Redis GET (miss) then SET | total, redis |
| `GET /test/db-read` | `SELECT` a random row | total, mysql |
| `GET /test/db-write` | `SELECT` + `UPDATE` | total, mysql |
| `GET /test/combined` | Redis cache-aside over MySQL | total, redis, mysql |
| `GET /healthz` | readiness | — |
| `GET /metrics` | Prometheus | — |

## Commands

- `make app-run` — run Go app locally against docker-compose MySQL+Redis
- `make app-build` — build static Linux binary
- `make docker` — build + push image to ghcr
- `make k8s-apply` — apply all manifests in dependency order
- `make k8s-destroy` — tear everything in the `load-test` namespace
- `make load-low` / `load-medium` / `load-high` — run k6 scenarios against the ingress
- `make dash` — port-forward Grafana to `localhost:3000`

## Reference

- [docs/requirements.md](docs/requirements.md) — endpoint spec, metrics spec, load scenarios
- [docs/architecture.md](docs/architecture.md) — k8s layout, scaling scenarios, how we attribute latency
- [docs/tasks.md](docs/tasks.md) — build order checklist

## Cluster context

Running on k3s (`kubectl` context: `default`). Two nodes:
- `v2202511264567410841` — control-plane, label `workload-tier=primary`
- `vm` — worker, label `workload-tier=secondary-1`
Both share `topology.devforce141/dc=netcup-nue`.

Namespace: `load-test` (create it before applying manifests).
