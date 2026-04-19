SHELL := /bin/bash

# ---- config ----
IMAGE         ?= ghcr.io/kha333n/load-test-app
TAG           ?= latest
NS            ?= load-test
PRIMARY_IP    ?= 159.195.58.193
# Public URL (via Caddy). Override with TARGET_URL=http://$(PRIMARY_IP):30080 to bypass Caddy.
TARGET_URL    ?= https://loadtest.kha333n.com
GRAFANA_URL   ?= http://localhost:3000
K6            ?= docker run --rm -i --network host -e TARGET=$(TARGET_URL) -v $$PWD/loadgen:/scripts grafana/k6:latest run

# ---- local app ----
.PHONY: app-run app-build docker docker-push

app-run: ## Bring up app + mysql + redis via docker-compose
	docker compose up --build

app-build:
	cd app && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o bin/load-test .

docker:
	docker build -t $(IMAGE):$(TAG) ./app

docker-push: docker
	docker push $(IMAGE):$(TAG)

# ---- k8s ----
.PHONY: k8s-apply k8s-destroy k8s-status dash dash-cm

k8s-apply: dash-cm ## Apply manifests in dependency order
	kubectl apply -f k8s/namespace.yaml
	kubectl apply -f k8s/mysql.yaml
	kubectl apply -f k8s/redis.yaml
	kubectl apply -f k8s/mailpit.yaml
	kubectl apply -f k8s/monitoring/kube-state-metrics.yaml
	kubectl apply -f k8s/monitoring/prometheus.yaml
	kubectl apply -f k8s/monitoring/grafana.yaml
	kubectl apply -f k8s/app.yaml
	kubectl apply -f k8s/hpa.yaml
	@echo "applied. wait for pods:"
	kubectl -n $(NS) get pods -w

k8s-destroy: ## Tear down everything in the namespace
	kubectl delete ns $(NS) --ignore-not-found

k8s-status:
	@kubectl -n $(NS) get pods -o wide
	@echo "---"
	@kubectl -n $(NS) get hpa
	@echo "---"
	@kubectl -n $(NS) get svc

# Build/refresh the dashboards ConfigMap from the source JSON.
dash-cm:
	kubectl create namespace $(NS) --dry-run=client -o yaml | kubectl apply -f -
	kubectl -n $(NS) create configmap grafana-dashboards \
	  --from-file=load-test.json=grafana/dashboards/load-test.json \
	  --dry-run=client -o yaml | kubectl apply -f -

dash: ## Port-forward Grafana to localhost:3000
	kubectl -n $(NS) port-forward svc/grafana 3000:3000

# ---- load scenarios ----
.PHONY: load-low load-medium load-high

load-low:
	./loadgen/annotate.sh "low.js start" "loadtest,low" || true
	$(K6) /scripts/low.js
	./loadgen/annotate.sh "low.js end"   "loadtest,low" || true

load-medium:
	./loadgen/annotate.sh "medium.js start" "loadtest,medium" || true
	$(K6) /scripts/medium.js
	./loadgen/annotate.sh "medium.js end"   "loadtest,medium" || true

load-high:
	./loadgen/annotate.sh "high.js start" "loadtest,high" || true
	$(K6) /scripts/high.js
	./loadgen/annotate.sh "high.js end"   "loadtest,high" || true

# ---- meta ----
.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  %-15s %s\n", $$1, $$2}'
