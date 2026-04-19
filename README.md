# load-test

K8s multi-node latency benchmark. A minimal Go HTTP service deployed on the `kha333n` k3s cluster, hit by k6 from outside, with Prometheus + Grafana recording every request.

Answers: **"when the app pod is on one node and MySQL/Redis are on another, what latency should we expect — and how does HPA scaling change that under load?"**

## Quick start

```bash
# local smoke test (no k8s)
make app-run                 # docker compose: app + mysql + redis on :8080

# deploy to the cluster (image is built & pushed by CI on push to main)
make k8s-apply               # creates ns, dashboards CM, applies in dep order

# expose loadtest.kha333n.com
# 1. Add the snippet at caddy/loadtest.kha333n.com.caddy to your Caddy config.
# 2. DNS: A loadtest.kha333n.com → 159.195.58.193
# 3. Caddy reverse-proxies to NodePort 30080.

# run a load scenario (defaults to https://loadtest.kha333n.com)
make load-low                # steady baseline
make load-medium             # HPA kick-in
make load-high               # saturation

# open Grafana
make dash                    # port-forwards localhost:3000
# (or NodePort: http://159.195.58.193:30030, anon viewer enabled)
```

## Image build

`.github/workflows/build-app-image.yml` builds `ghcr.io/kha333n/load-test-app` on every push to `main` and on `v*.*.*` tags. After the first push, make the package public (GitHub → Profile → Packages → load-test-app → Settings → Change visibility → Public) so the cluster can pull without an `imagePullSecret`.

## Docs

- [CLAUDE.md](CLAUDE.md) — tech stack + project rules
- [docs/requirements.md](docs/requirements.md) — endpoints + metrics spec + load scenarios
- [docs/architecture.md](docs/architecture.md) — k8s layout + latency attribution
- [docs/tasks.md](docs/tasks.md) — build order

## Why Go

This is a measurement tool. Go has negligible framework overhead, so the recorded latency is dominated by network + DB/Redis RTT — exactly the signal we want. A Laravel variant can come later if we want realistic app-framework numbers to compare.
