#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

WITH_SDR=false
if [[ "${1:-}" == "--with-sdr" ]]; then
  WITH_SDR=true
fi

COMPOSE_PROFILE_ARGS=()
if [[ "$WITH_SDR" == true ]]; then
  COMPOSE_PROFILE_ARGS=(--profile sdr)
fi

echo "==> Building and starting the full stack (this can take a few minutes on first run)"
docker compose "${COMPOSE_PROFILE_ARGS[@]}" up --build -d

cleanup() {
  echo "==> Tearing down"
  docker compose "${COMPOSE_PROFILE_ARGS[@]}" down
}
trap cleanup EXIT

wait_for() {
  local name="$1"
  local check_cmd="$2"
  local attempts=30
  local delay=2

  echo -n "==> Waiting for $name"
  for ((i = 1; i <= attempts; i++)); do
    if eval "$check_cmd" >/dev/null 2>&1; then
      echo " ok"
      return 0
    fi
    echo -n "."
    sleep "$delay"
  done
  echo " FAILED"
  echo "    $name did not become healthy after $((attempts * delay))s"
  echo "    --- logs ---"
  docker compose logs --tail=50 "$name" || true
  return 1
}

FAILURES=0

wait_for postgres "docker compose exec -T postgres pg_isready -U lyrahcle" || FAILURES=$((FAILURES + 1))
wait_for redis "docker compose exec -T redis redis-cli ping | grep -q PONG" || FAILURES=$((FAILURES + 1))
wait_for backend "curl -fsS http://localhost:8080/healthz" || FAILURES=$((FAILURES + 1))
wait_for backend-api "curl -fsS http://localhost:8080/api/v1/ping" || FAILURES=$((FAILURES + 1))
wait_for frontend "curl -fsS http://localhost:5173/" || FAILURES=$((FAILURES + 1))
wait_for cve-sync "curl -fsS http://localhost:9091/metrics" || FAILURES=$((FAILURES + 1))
wait_for prometheus "curl -fsS http://localhost:9090/-/healthy" || FAILURES=$((FAILURES + 1))

if [[ "$WITH_SDR" == true ]]; then
  wait_for sdr-agent "docker compose ps sdr-agent | grep -q Up" || FAILURES=$((FAILURES + 1))
fi

echo ""
if [[ "$FAILURES" -eq 0 ]]; then
  echo "==> All services healthy."
  exit 0
else
  echo "==> $FAILURES service(s) failed their health check. See logs above."
  exit 1
fi
