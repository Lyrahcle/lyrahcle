#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLUSTER="${LYRAHCLE_KIND_CLUSTER:-lyrahcle}"
NAMESPACE="${LYRAHCLE_K8S_NAMESPACE:-lyrahcle}"
CONTEXT="${LYRAHCLE_K8S_CONTEXT:-kind-${CLUSTER}}"
TARGETS="$ROOT/deploy/resilience/targets.json"
REPORT="/tmp/lyrahcle-k8s-resilience-report.json"
TMP="$(mktemp -d)"
DRAINED_NODE=""
POSTGRES_BLOCKED=0
REDIS_BLOCKED=0
RESULTS=()

need() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'missing required command: %s\n' "$1" >&2
    exit 1
  }
}

for command in kubectl python3; do
  need "$command"
done

kctl=(kubectl --context "$CONTEXT" -n "$NAMESPACE")

restore_faults() {
  if [[ "$POSTGRES_BLOCKED" -eq 1 ]]; then
    "${kctl[@]}" patch service postgres --type=merge -p '{"spec":{"selector":{"app.kubernetes.io/name":"postgres"}}}' >/dev/null 2>&1 || true
  fi
  if [[ "$REDIS_BLOCKED" -eq 1 ]]; then
    "${kctl[@]}" patch service redis --type=merge -p '{"spec":{"selector":{"app.kubernetes.io/name":"redis"}}}' >/dev/null 2>&1 || true
  fi
  if [[ -n "$DRAINED_NODE" ]]; then
    kubectl --context "$CONTEXT" uncordon "$DRAINED_NODE" >/dev/null 2>&1 || true
  fi
  rm -rf "$TMP"
}
trap restore_faults EXIT

read_target() {
  python3 - "$TARGETS" "$1" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as handle:
    data = json.load(handle)
value = data["local_kind"][sys.argv[2]]
print(value)
PY
}

now_ms() {
  python3 - <<'PY'
import time
print(int(time.monotonic() * 1000))
PY
}

elapsed_seconds() {
  python3 - "$1" "$2" <<'PY'
import sys
print(round((int(sys.argv[2]) - int(sys.argv[1])) / 1000, 3))
PY
}

record_result() {
  RESULTS+=("$1|$2|$3|$4")
  printf '%-34s %8ss / %ss  %s\n' "$1" "$2" "$3" "$4"
}

ready_replicas() {
  "${kctl[@]}" get deployment "$1" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || true
}

desired_replicas() {
  "${kctl[@]}" get deployment "$1" -o jsonpath='{.spec.replicas}'
}

wait_deployment_ready() {
  local deployment="$1"
  local timeout="$2"
  local start
  start="$(now_ms)"
  local desired
  desired="$(desired_replicas "$deployment")"
  while true; do
    local ready
    ready="$(ready_replicas "$deployment")"
    ready="${ready:-0}"
    if [[ "$ready" -ge "$desired" ]]; then
      return 0
    fi
    local current
    current="$(now_ms)"
    if (( current - start >= timeout * 1000 )); then
      printf 'timed out waiting for %s: ready=%s desired=%s\n' "$deployment" "$ready" "$desired" >&2
      return 1
    fi
    sleep 1
  done
}

wait_deployment_not_ready() {
  local deployment="$1"
  local timeout="$2"
  local start
  start="$(now_ms)"
  local desired
  desired="$(desired_replicas "$deployment")"
  while true; do
    local ready
    ready="$(ready_replicas "$deployment")"
    ready="${ready:-0}"
    if [[ "$ready" -lt "$desired" ]]; then
      return 0
    fi
    local current
    current="$(now_ms)"
    if (( current - start >= timeout * 1000 )); then
      printf 'timed out waiting for %s readiness to fail\n' "$deployment" >&2
      return 1
    fi
    sleep 1
  done
}

wait_service_endpoint() {
  local service="$1"
  local expect="$2"
  local timeout="$3"
  local start
  start="$(now_ms)"
  while true; do
    local addresses
    addresses="$("${kctl[@]}" get endpoints "$service" -o jsonpath='{.subsets[*].addresses[*].ip}' 2>/dev/null || true)"
    if [[ "$expect" == "present" && -n "$addresses" ]]; then
      return 0
    fi
    if [[ "$expect" == "absent" && -z "$addresses" ]]; then
      return 0
    fi
    local current
    current="$(now_ms)"
    if (( current - start >= timeout * 1000 )); then
      printf 'timed out waiting for %s endpoint state %s\n' "$service" "$expect" >&2
      return 1
    fi
    sleep 1
  done
}

restart_total() {
  "${kctl[@]}" get pods -l app.kubernetes.io/component=backend -o json | python3 -c 'import json,sys; d=json.load(sys.stdin); print(sum(c.get("restartCount",0) for p in d.get("items",[]) for c in p.get("status",{}).get("containerStatuses",[])))'
}

measure_pod_recovery() {
  local label="$1"
  local deployment="$2"
  local selector="$3"
  local target_key="$4"
  local target
  target="$(read_target "$target_key")"
  local pod
  pod="$("${kctl[@]}" get pods -l "$selector" --field-selector=status.phase=Running -o jsonpath='{.items[0].metadata.name}')"
  local uid
  uid="$("${kctl[@]}" get pod "$pod" -o jsonpath='{.metadata.uid}')"
  local start
  start="$(now_ms)"
  "${kctl[@]}" delete pod "$pod" --wait=false >/dev/null
  wait_deployment_ready "$deployment" "$target"
  while "${kctl[@]}" get pods -l "$selector" -o jsonpath='{range .items[*]}{.metadata.uid}{"\n"}{end}' | grep -Fxq "$uid"; do
    sleep 1
  done
  local end
  end="$(now_ms)"
  local elapsed
  elapsed="$(elapsed_seconds "$start" "$end")"
  python3 - "$elapsed" "$target" <<'PY'
import sys
raise SystemExit(0 if float(sys.argv[1]) <= float(sys.argv[2]) else 1)
PY
  record_result "$label" "$elapsed" "$target" "PASS"
}

for deployment in lyrahcle-backend lyrahcle-frontend lyrahcle-cve-sync; do
  "${kctl[@]}" rollout status "deployment/$deployment" --timeout=120s >/dev/null
done

backend_ready="$(ready_replicas lyrahcle-backend)"
frontend_ready="$(ready_replicas lyrahcle-frontend)"
if [[ "${backend_ready:-0}" -lt 2 || "${frontend_ready:-0}" -lt 2 ]]; then
  printf 'resilience drill requires at least two ready backend and frontend replicas\n' >&2
  exit 1
fi

for pdb in lyrahcle-backend lyrahcle-frontend; do
  min_available="$("${kctl[@]}" get pdb "$pdb" -o jsonpath='{.spec.minAvailable}')"
  disruptions="$("${kctl[@]}" get pdb "$pdb" -o jsonpath='{.status.disruptionsAllowed}')"
  if [[ "$min_available" -lt 1 || "$disruptions" -lt 1 ]]; then
    printf 'PDB %s is not ready for one voluntary disruption\n' "$pdb" >&2
    exit 1
  fi
done

measure_pod_recovery "backend pod recovery" "lyrahcle-backend" "app.kubernetes.io/component=backend" "backend_pod_rto_seconds"
measure_pod_recovery "frontend pod recovery" "lyrahcle-frontend" "app.kubernetes.io/component=frontend" "frontend_pod_rto_seconds"
measure_pod_recovery "CVE sync pod recovery" "lyrahcle-cve-sync" "app.kubernetes.io/component=cve-sync" "cve_sync_pod_rto_seconds"

backend_nodes="$("${kctl[@]}" get pods -l app.kubernetes.io/component=backend -o jsonpath='{range .items[*]}{.spec.nodeName}{"\n"}{end}' | sort -u | sed '/^$/d' | wc -l)"
frontend_nodes="$("${kctl[@]}" get pods -l app.kubernetes.io/component=frontend -o jsonpath='{range .items[*]}{.spec.nodeName}{"\n"}{end}' | sort -u | sed '/^$/d' | wc -l)"
if [[ "$backend_nodes" -lt 2 || "$frontend_nodes" -lt 2 ]]; then
  printf 'backend/frontend replicas are not spread across at least two nodes\n' >&2
  exit 1
fi

DRAINED_NODE="$("${kctl[@]}" get pods -l app.kubernetes.io/component=backend -o jsonpath='{.items[0].spec.nodeName}')"
node_target="$(read_target node_drain_application_rto_seconds)"
node_start="$(now_ms)"
kubectl --context "$CONTEXT" drain "$DRAINED_NODE" --ignore-daemonsets --delete-emptydir-data --force --pod-selector=app.kubernetes.io/instance=lyrahcle --timeout="${node_target}s" >/dev/null
wait_deployment_ready lyrahcle-backend "$node_target"
wait_deployment_ready lyrahcle-frontend "$node_target"
wait_deployment_ready lyrahcle-cve-sync "$node_target"
node_end="$(now_ms)"
node_elapsed="$(elapsed_seconds "$node_start" "$node_end")"
python3 - "$node_elapsed" "$node_target" <<'PY'
import sys
raise SystemExit(0 if float(sys.argv[1]) <= float(sys.argv[2]) else 1)
PY
record_result "worker drain recovery" "$node_elapsed" "$node_target" "PASS"
kubectl --context "$CONTEXT" uncordon "$DRAINED_NODE" >/dev/null
DRAINED_NODE=""

redis_detection="$(read_target redis_outage_detection_seconds)"
redis_target="$(read_target redis_recovery_rto_seconds)"
redis_restart_before="$(restart_total)"
redis_outage_start="$(now_ms)"
"${kctl[@]}" patch service redis --type=merge -p '{"spec":{"selector":{"app.kubernetes.io/name":"resilience-blocked-redis"}}}' >/dev/null
REDIS_BLOCKED=1
wait_service_endpoint redis absent 20
redis_pod="$("${kctl[@]}" get pods -l app.kubernetes.io/name=redis -o jsonpath='{.items[0].metadata.name}')"
"${kctl[@]}" exec "$redis_pod" -- redis-cli CLIENT KILL TYPE normal >/dev/null
wait_deployment_not_ready lyrahcle-backend "$redis_detection"
redis_detected="$(now_ms)"
redis_detection_elapsed="$(elapsed_seconds "$redis_outage_start" "$redis_detected")"
python3 - "$redis_detection_elapsed" "$redis_detection" <<'PY'
import sys
raise SystemExit(0 if float(sys.argv[1]) <= float(sys.argv[2]) else 1)
PY
record_result "Redis outage detection" "$redis_detection_elapsed" "$redis_detection" "PASS"

redis_restore_start="$(now_ms)"
"${kctl[@]}" patch service redis --type=merge -p '{"spec":{"selector":{"app.kubernetes.io/name":"redis"}}}' >/dev/null
REDIS_BLOCKED=0
wait_service_endpoint redis present "$redis_target"
wait_deployment_ready lyrahcle-backend "$redis_target"
redis_restore_end="$(now_ms)"
redis_elapsed="$(elapsed_seconds "$redis_restore_start" "$redis_restore_end")"
redis_restart_after="$(restart_total)"
if [[ "$redis_restart_after" -gt "$redis_restart_before" ]]; then
  printf 'backend containers restarted during Redis connectivity fault: before=%s after=%s\n' "$redis_restart_before" "$redis_restart_after" >&2
  exit 1
fi
python3 - "$redis_elapsed" "$redis_target" <<'PY'
import sys
raise SystemExit(0 if float(sys.argv[1]) <= float(sys.argv[2]) else 1)
PY
record_result "Redis service recovery" "$redis_elapsed" "$redis_target" "PASS"

postgres_detection="$(read_target postgres_outage_detection_seconds)"
postgres_recovery="$(read_target postgres_recovery_rto_seconds)"
restart_before="$(restart_total)"
postgres_outage_start="$(now_ms)"
"${kctl[@]}" patch service postgres --type=merge -p '{"spec":{"selector":{"app.kubernetes.io/name":"resilience-blocked-postgres"}}}' >/dev/null
POSTGRES_BLOCKED=1
wait_service_endpoint postgres absent 20
postgres_pod="$("${kctl[@]}" get pods -l app.kubernetes.io/name=postgres -o jsonpath='{.items[0].metadata.name}')"
"${kctl[@]}" exec "$postgres_pod" -- sh -ec 'psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$POSTGRES_DB" -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = current_database() AND pid <> pg_backend_pid();" >/dev/null'
wait_deployment_not_ready lyrahcle-backend "$postgres_detection"
postgres_detected="$(now_ms)"
postgres_detection_elapsed="$(elapsed_seconds "$postgres_outage_start" "$postgres_detected")"
python3 - "$postgres_detection_elapsed" "$postgres_detection" <<'PY'
import sys
raise SystemExit(0 if float(sys.argv[1]) <= float(sys.argv[2]) else 1)
PY
record_result "PostgreSQL outage detection" "$postgres_detection_elapsed" "$postgres_detection" "PASS"

postgres_restore_start="$(now_ms)"
"${kctl[@]}" patch service postgres --type=merge -p '{"spec":{"selector":{"app.kubernetes.io/name":"postgres"}}}' >/dev/null
POSTGRES_BLOCKED=0
wait_service_endpoint postgres present "$postgres_recovery"
wait_deployment_ready lyrahcle-backend "$postgres_recovery"
postgres_restore_end="$(now_ms)"
postgres_recovery_elapsed="$(elapsed_seconds "$postgres_restore_start" "$postgres_restore_end")"
restart_after="$(restart_total)"
if [[ "$restart_after" -gt "$restart_before" ]]; then
  printf 'backend containers restarted during PostgreSQL connectivity fault: before=%s after=%s\n' "$restart_before" "$restart_after" >&2
  exit 1
fi
python3 - "$postgres_recovery_elapsed" "$postgres_recovery" <<'PY'
import sys
raise SystemExit(0 if float(sys.argv[1]) <= float(sys.argv[2]) else 1)
PY
record_result "PostgreSQL service recovery" "$postgres_recovery_elapsed" "$postgres_recovery" "PASS"

printf '%s\n' "${RESULTS[@]}" > "$TMP/results.txt"
python3 - "$TARGETS" "$TMP/results.txt" "$REPORT" <<'PY'
import json, sys
from datetime import datetime, timezone
with open(sys.argv[1], encoding="utf-8") as handle:
    targets = json.load(handle)
results = []
with open(sys.argv[2], encoding="utf-8") as handle:
    for line in handle:
        name, actual, target, status = line.rstrip("\n").split("|", 3)
        results.append({"name": name, "actual_seconds": float(actual), "target_seconds": float(target), "status": status})
report = {
    "generated_at": datetime.now(timezone.utc).isoformat(),
    "environment": "local-kind",
    "status": "PASS",
    "rpo_scope": "No persistent storage was destroyed. PostgreSQL and Redis faults isolate Service connectivity and terminate existing client connections only.",
    "targets_status": targets["status"],
    "results": results,
}
with open(sys.argv[3], "w", encoding="utf-8") as handle:
    json.dump(report, handle, indent=2)
    handle.write("\n")
PY

printf '\nkubernetes resilience test: OK\n'
printf 'report: %s\n' "$REPORT"
