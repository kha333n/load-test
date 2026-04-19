#!/usr/bin/env bash
# Push a Grafana annotation marking the start/end of a load scenario.
# Usage: annotate.sh <text> [tags-comma-separated]
set -euo pipefail

GRAFANA_URL="${GRAFANA_URL:-http://localhost:3000}"
GRAFANA_USER="${GRAFANA_USER:-admin}"
GRAFANA_PASS="${GRAFANA_PASS:-admin}"

text="${1:?text required}"
tags="${2:-loadtest}"
tags_json=$(printf '"%s"' "${tags//,/\",\"}")

curl -fsS -u "${GRAFANA_USER}:${GRAFANA_PASS}" \
  -H 'Content-Type: application/json' \
  "${GRAFANA_URL}/api/annotations" \
  -d "{\"text\":\"${text}\",\"tags\":[${tags_json}]}" >/dev/null
echo "annotated: ${text} [${tags}]"
