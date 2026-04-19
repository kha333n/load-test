# Architecture

## Cluster topology

Two k3s nodes in the `netcup-nue` DC:

| Node | Role | Label we select on |
|---|---|---|
| `v2202511264567410841` | control-plane | `workload-tier=primary` |
| `vm` | worker | `workload-tier=secondary-1` |

k3s control-plane has no `NoSchedule` taint, so workloads can land on either node. We use `nodeSelector` (data services) and `topologySpreadConstraints` (app) to control placement.

## Component placement

```
┌────────────────────────── primary node ──────────────────────────┐
│                                                                  │
│   app pod (replica 1)  ──┐                                       │
│                          │                                       │
│   prometheus             │                                       │
│   grafana                │                                       │
│                          │                                       │
└──────────────────────────┼───────────────────────────────────────┘
                           │ cross-node traffic (this is what we measure)
┌──────────────────────────┼────────── secondary node ─────────────┐
│                          ▼                                       │
│   mysql  ◄──────────  app pod (replica 2, scaled)                │
│   redis                                                          │
│   mailpit                                                        │
│   kube-state-metrics                                             │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

Data services pinned to secondary so **every request from the primary-node app pod crosses the node boundary** — that's the thing we care about.

## App deployment scaling profile

- `replicas: 1` initial.
- HPA: `minReplicas: 1`, `maxReplicas: 4`, target `averageUtilization: 70%` on CPU.
- `topologySpreadConstraints`: `maxSkew: 1` over `kubernetes.io/hostname` — so when HPA scales up, pods spread across nodes instead of piling on one.
- `PodAntiAffinity` (preferred, not required) against itself on same node — secondary preference, backstop for small replica counts.

This yields three observable states we care about:
- **State A:** 1 pod on primary, MySQL+Redis on secondary → always cross-node.
- **State B:** 2 pods (one per node) → 50% cross-node, 50% local.
- **State C:** 3–4 pods spread → mix with one node hosting more pods.

The Grafana `node` label lets us slice latency by which state each request was served under.

## How latency is attributed

Each request flows through this middleware chain:

```
incoming → request-id → timing-start → handler → timing-end → metrics-record → response
```

- `timing-start` captures `time.Now()` on entry, before any handler work.
- `timing-end` records the delta as `http_request_duration_seconds`.
- Handlers that do I/O wrap each call (`redis.Get`, `db.QueryRow`, etc.) with their own timer and emit the sub-latency histogram **before** returning.
- A request that uses both Redis and MySQL records: one `http_request_duration` sample (total), one `redis_operation_duration` sample per Redis call, one `mysql_query_duration` sample per query. They share the same `endpoint` label.

This way, for any high-latency outlier in `http_request_duration`, we can look at the same timestamp bucket in the sub-latency histograms and see which I/O path caused it.

## Data services — k8s specifics

- **MySQL:** single pod, PVC for data, `nodeSelector: workload-tier=secondary-1`. Exposed as ClusterIP `mysql.load-test.svc`.
- **Redis:** single pod, ephemeral (no PVC — it's a benchmark target, data is regenerated on startup). Same nodeSelector. Exposed as `redis.load-test.svc`.
- **Mailpit:** single pod, ephemeral, same nodeSelector. Exposed as `mailpit.load-test.svc:1025` (SMTP) and `:8025` (web). App sends mails asynchronously via a goroutine queue so SMTP latency doesn't bleed into HTTP latency.

## Observability stack

- **Prometheus:** scrapes `load-test-app:8080/metrics` every 5s (high resolution because load tests are short). Scrapes kube-state-metrics every 15s.
- **kube-state-metrics:** standard deployment from upstream manifests.
- **Grafana:** provisioned with one dashboard JSON at `grafana/dashboards/load-test.json`, Prometheus datasource auto-configured, anonymous-viewer enabled for simplicity.

## Ingress

One ingress host `load-test.kha333n.cluster` (or `.localhost` via port-forward during early dev). k6 targets that hostname. No TLS in v1.

## Load generator placement

k6 runs **outside** the cluster (from dev machine over VPN, or from a separate pod on the primary node that we drain before the test). It must not compete for CPU with the app pods; the cleanest setup is dev-machine over the VPN.

## Scale-event visualization

Grafana reads:
- `kube_horizontalpodautoscaler_status_current_replicas`
- `kube_deployment_status_replicas_ready`
- `kube_pod_info` joined with `node` label

Rendered as:
- A step-line showing replica count over time.
- A table panel refreshing every 15s with `pod_name → node_name`.
- A timeseries of p95 latency per `node` label, so scale-up moments line up visually with latency effects.

## Connection pooling

- MySQL: `SetMaxOpenConns(20)`, `SetMaxIdleConns(10)`, `SetConnMaxLifetime(5m)`.
- Redis: `PoolSize: 20`, `MinIdleConns: 5`.

Per pod. With 4 pods at max, that's 80 MySQL conns and 80 Redis conns — well within defaults. If we want to stress pool behavior later, we lower these numbers.

## Failure modes to watch

- **MySQL connection exhaustion:** visible as p99 spike on db-* endpoints plus `mysql_connections_in_use` hitting 20.
- **Redis pool saturation:** same pattern on cache-* endpoints.
- **HPA lag:** if medium load arrives and HPA hasn't scaled yet, p95 on compute endpoint (CPU-bound) spikes first. Useful canary for when to raise `minReplicas`.
- **Cross-node network blips:** p99 on db-write much worse than p99 on db-read suggests cross-node tail latency rather than DB slowness (an UPDATE is one extra RTT over a SELECT).
