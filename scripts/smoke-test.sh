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
  local attempts=60
  local delay=3

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
wait_for backend "curl -fsS http://localhost:8080/readyz" || FAILURES=$((FAILURES + 1))
wait_for backend-api "curl -fsS http://localhost:8080/api/v1/ping" || FAILURES=$((FAILURES + 1))
wait_for frontend "curl -fsS http://localhost:5173/" || FAILURES=$((FAILURES + 1))
wait_for cve-sync "curl -fsS http://localhost:9091/metrics" || FAILURES=$((FAILURES + 1))
wait_for prometheus "curl -fsS http://localhost:9090/-/healthy" || FAILURES=$((FAILURES + 1))

run_auth_e2e() {
  local login_response token status relogin_response
  login_response="$(curl -fsS -H 'Content-Type: application/json' -d '{"identifier":"admin","password":"admin"}' http://localhost:8080/api/v1/auth/login)" || return 1
  token="$(printf '%s' "$login_response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')" || return 1
  [[ -n "$token" ]] || return 1
  curl -fsS -H "Authorization: Bearer $token" http://localhost:8080/api/v1/auth/me >/dev/null || return 1
  curl -fsS -X POST -H "Authorization: Bearer $token" http://localhost:8080/api/v1/auth/logout >/dev/null || return 1
  status="$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $token" http://localhost:8080/api/v1/auth/me)" || return 1
  [[ "$status" == "401" ]] || return 1
  relogin_response="$(curl -fsS -H 'Content-Type: application/json' -d '{"identifier":"admin","password":"admin"}' http://localhost:8080/api/v1/auth/login)" || return 1
  printf '%s' "$relogin_response" | python3 -c 'import json,sys; data=json.load(sys.stdin); assert data.get("token")' >/dev/null || return 1
}

echo -n "==> Running authentication E2E"
if run_auth_e2e; then
  echo " ok"
else
  echo " FAILED"
  FAILURES=$((FAILURES + 1))
fi

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
