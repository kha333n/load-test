# Requirements

## What we're measuring

How much latency the app sees when it talks to MySQL and Redis running on a **different node** in the same k3s cluster, under varying load — and how pod scaling (HPA + cross-node spread) affects that.

We are **not** measuring application framework overhead (that's why we're not using Laravel). We are measuring:

1. Steady-state request latency for different I/O shapes (no-I/O, Redis-only, MySQL-only, both).
2. How those latencies change under low / medium / high request rates.
3. How HPA scaling affects latency and error rate.
4. Latency difference when the app pod is co-located with data services vs. on the other node.
5. Connection pool saturation behavior.

## Endpoints

Each endpoint is deliberately minimal so the recorded latency is dominated by I/O, not by application work.

### `GET /test/compute`
- Runs a fixed CPU-bound hash loop (e.g. SHA-256 over 10 KB, 100 iterations).
- Baseline — shows pure request-handling overhead with no I/O.
- Records: `total`.

### `GET /test/cache-hit`
- Reads a key from Redis that is guaranteed to exist (pre-warmed on startup).
- Records: `total`, `redis`.
- Label `outcome=hit`.

### `GET /test/cache-miss`
- Reads a key that does not exist, then writes a ~1 KB value with 60s TTL.
- Records: `total`, `redis` (sum of GET + SET).
- Label `outcome=miss`.

### `GET /test/db-read`
- `SELECT * FROM items WHERE id = ?` with `?` chosen randomly from a seeded range.
- Records: `total`, `mysql`.

### `GET /test/db-write`
- `SELECT` a row, then `UPDATE items SET hit_count = hit_count + 1, updated_at = NOW() WHERE id = ?`.
- Records: `total`, `mysql` (sum of SELECT + UPDATE).

### `GET /test/combined`
- Cache-aside: try Redis; on miss, query MySQL and populate Redis.
- Records: `total`, `redis`, `mysql`. Label `outcome=hit|miss`.

### `GET /healthz`
- Returns 200 if MySQL + Redis reachable. Used for readiness probe.

### `GET /metrics`
- Prometheus scrape endpoint.

## Seed data

On app startup, if `items` table is empty:
- Create `items(id INT PRIMARY KEY, payload VARCHAR(512), hit_count INT, updated_at TIMESTAMP)`.
- Insert 10,000 rows with random payloads.
- Warm Redis with keys `cache:hit:{1..1000}` set to ~1 KB values.

Seeding is idempotent.

## Metrics (Prometheus)

All histograms use buckets: `0.5ms, 1ms, 2ms, 5ms, 10ms, 25ms, 50ms, 100ms, 250ms, 500ms, 1s, 2.5s`.

| Metric | Type | Labels |
|---|---|---|
| `http_request_duration_seconds` | histogram | `endpoint`, `pod`, `node`, `status` |
| `http_requests_total` | counter | `endpoint`, `pod`, `node`, `status` |
| `redis_operation_duration_seconds` | histogram | `endpoint`, `op` (`get`/`set`), `pod`, `node`, `outcome` |
| `mysql_query_duration_seconds` | histogram | `endpoint`, `op` (`select`/`update`), `pod`, `node` |
| `mysql_connections_in_use` | gauge | `pod`, `node` |
| `redis_connections_in_use` | gauge | `pod`, `node` |
| `app_inflight_requests` | gauge | `pod`, `node` |

`pod` and `node` come from env vars populated via Downward API.

## Scaling / replica metrics

From `kube-state-metrics`:
- `kube_deployment_status_replicas{deployment="load-test-app"}`
- `kube_pod_info{namespace="load-test"}` — to map pods to nodes over time
- `kube_horizontalpodautoscaler_status_current_replicas`

These let the dashboard show "how many pods were serving traffic when p95 spiked".

## Load scenarios (k6)

All three drive traffic through the ingress hostname (from outside the cluster). Each scenario cycles through all endpoints weighted roughly as: 10% compute, 20% cache-hit, 10% cache-miss, 20% db-read, 15% db-write, 25% combined.

### Low — `loadgen/low.js`
- Steady 20 RPS for 5 minutes.
- Purpose: establish clean baseline latencies with no resource pressure.

### Medium — `loadgen/medium.js`
- Ramp: 0 → 200 RPS over 2 min, hold 200 RPS for 5 min, ramp down over 1 min.
- Purpose: observe HPA kick-in and pod spread; should stay under saturation.

### High — `loadgen/high.js`
- Ramp: 0 → 800 RPS over 1 min, hold 800 RPS for 3 min, spike to 1200 RPS for 30s, hold 800 RPS for 2 min.
- Purpose: saturation test. Expect HPA to max out at 4 replicas; observe connection-pool queueing and error rate.

Each scenario writes a k6 summary JSON + a Grafana annotation at start/end so the dashboard marks the run visually.

## What the dashboard must show

1. **Request rate** per endpoint over time (stacked area).
2. **Latency** — p50/p95/p99 per endpoint, total vs. redis vs. mysql sub-latencies shown as separate series.
3. **Error rate** — non-2xx per endpoint.
4. **Pod count** for the app deployment over time, with HPA current-replicas overlaid.
5. **Pod-to-node map** — table showing which pods are on which node right now (refreshed every 15s).
6. **Connection pool depth** — `mysql_connections_in_use`, `redis_connections_in_use` per pod.
7. **Co-location comparison panel** — p95 latency for requests served by primary-node pods vs secondary-node pods (filter by `node` label).

## Out of scope (for v1)

- Per-request raw rows (Prometheus aggregates are enough to answer the questions).
- TLS termination measurement.
- Multi-region / multi-DC.
- Application-side caching (in-process LRU) — only Redis.
- Write-heavy workloads beyond the single-row UPDATE. We can add batch/insert scenarios later.
