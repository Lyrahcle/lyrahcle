#!/usr/bin/env bash
set -euo pipefail

CLUSTER="${LYRAHCLE_KIND_CLUSTER:-lyrahcle}"
NAMESPACE="${LYRAHCLE_K8S_NAMESPACE:-lyrahcle}"
CONTEXT="${LYRAHCLE_K8S_CONTEXT:-kind-${CLUSTER}}"
FRONTEND_PORT="${LYRAHCLE_K8S_FRONTEND_PORT:-18080}"
BACKEND_PORT="${LYRAHCLE_K8S_BACKEND_PORT:-18081}"
CVE_PORT="${LYRAHCLE_K8S_CVE_PORT:-19091}"
TMP="$(mktemp -d)"
PIDS=()

cleanup() {
  for pid in "${PIDS[@]:-}"; do
    kill "$pid" >/dev/null 2>&1 || true
    wait "$pid" >/dev/null 2>&1 || true
  done
  rm -rf "$TMP"
}
trap cleanup EXIT

for command in kubectl curl; do
  command -v "$command" >/dev/null 2>&1 || {
    printf 'missing required command: %s\n' "$command" >&2
    exit 1
  }
done

kctl=(kubectl --context "$CONTEXT" -n "$NAMESPACE")

"${kctl[@]}" rollout status deployment/lyrahcle-backend --timeout=120s
"${kctl[@]}" rollout status deployment/lyrahcle-frontend --timeout=120s
"${kctl[@]}" rollout status deployment/lyrahcle-cve-sync --timeout=120s

backend_ready="$("${kctl[@]}" get deployment lyrahcle-backend -o jsonpath='{.status.readyReplicas}')"
frontend_ready="$("${kctl[@]}" get deployment lyrahcle-frontend -o jsonpath='{.status.readyReplicas}')"
if [[ "${backend_ready:-0}" -lt 2 || "${frontend_ready:-0}" -lt 2 ]]; then
  printf 'expected at least two ready backend/frontend replicas; got backend=%s frontend=%s\n' \
    "${backend_ready:-0}" "${frontend_ready:-0}" >&2
  exit 1
fi

"${kctl[@]}" port-forward svc/lyrahcle-frontend "$FRONTEND_PORT:80" >"$TMP/frontend.log" 2>&1 & PIDS+=("$!")
"${kctl[@]}" port-forward svc/lyrahcle-backend "$BACKEND_PORT:8080" >"$TMP/backend.log" 2>&1 & PIDS+=("$!")
"${kctl[@]}" port-forward svc/lyrahcle-cve-sync "$CVE_PORT:9091" >"$TMP/cve.log" 2>&1 & PIDS+=("$!")

wait_http() {
  local url="$1"
  local attempts=0
  until curl --fail --silent --show-error --max-time 2 "$url" >/dev/null 2>&1; do
    attempts=$((attempts + 1))
    if [[ "$attempts" -ge 40 ]]; then
      printf 'timed out waiting for %s\n' "$url" >&2
      cat "$TMP"/*.log >&2 || true
      return 1
    fi
    sleep 0.5
  done
}

wait_http "http://127.0.0.1:${FRONTEND_PORT}/healthz"
wait_http "http://127.0.0.1:${BACKEND_PORT}/readyz"
wait_http "http://127.0.0.1:${BACKEND_PORT}/metrics"
wait_http "http://127.0.0.1:${CVE_PORT}/metrics"

curl --fail --silent "http://127.0.0.1:${BACKEND_PORT}/readyz" | grep -q '"status":"ready"'
curl --fail --silent "http://127.0.0.1:${BACKEND_PORT}/metrics" | grep -q '^lyrahcle_http_requests_total'
curl --fail --silent "http://127.0.0.1:${CVE_PORT}/metrics" | grep -q '^lyrahcle_cve_sync_'

printf 'kubernetes smoke test: OK (backend=%s frontend=%s)\n' "$backend_ready" "$frontend_ready"
