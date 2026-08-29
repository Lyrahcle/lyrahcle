#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

WITH_SDR=false
if [[ "${1:-}" == "--with-sdr" ]]; then
  WITH_SDR=true
fi

COMPOSE_PROFILE_ARGS=(--profile observability)
if [[ "$WITH_SDR" == true ]]; then
  COMPOSE_PROFILE_ARGS+=(--profile sdr)
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
wait_for grafana "curl -fsS http://localhost:3000/api/health" || FAILURES=$((FAILURES + 1))

run_auth_e2e() {
  local bootstrap_password new_password login_response token must_change_password status relogin_response sessions setup_response secret provider totp_code totp_login_response totp_token

  bootstrap_password="${SEED_ADMIN_PASSWORD:-admin}"
  new_password="${SMOKE_ADMIN_NEW_PASSWORD:-Lyrahcle-Smoke-2026!}"
  [[ "$bootstrap_password" != "$new_password" ]] || new_password="Lyrahcle-Smoke-2026!x"

  totp_now() {
    python3 - "$1" <<'PYCODE'
import base64, hashlib, hmac, struct, sys, time
secret = sys.argv[1].strip().upper()
padding = '=' * ((8 - len(secret) % 8) % 8)
key = base64.b32decode(secret + padding)
counter = int(time.time()) // 30
digest = hmac.new(key, struct.pack('>Q', counter), hashlib.sha1).digest()
offset = digest[-1] & 0x0f
value = struct.unpack('>I', digest[offset:offset + 4])[0] & 0x7fffffff
print(f'{value % 1000000:06d}')
PYCODE
  }

  login_response="$(curl -fsS -H 'Content-Type: application/json' -d "{\"identifier\":\"admin\",\"password\":\"$bootstrap_password\"}" http://localhost:8080/api/v1/auth/login)" || return 1
  token="$(printf '%s' "$login_response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')" || return 1
  must_change_password="$(printf '%s' "$login_response" | python3 -c 'import json,sys; print(str(json.load(sys.stdin).get("must_change_password", False)).lower())')" || return 1
  [[ -n "$token" && "$must_change_password" == "true" ]] || return 1

  status="$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $token" http://localhost:8080/api/v1/auth/sessions/)" || return 1
  [[ "$status" == "403" ]] || return 1
  status="$(curl -sS -o /dev/null -w '%{http_code}' -X POST -H "Authorization: Bearer $token" -H 'Content-Type: application/json' -d "{\"current_password\":\"$bootstrap_password\",\"new_password\":\"$new_password\"}" http://localhost:8080/api/v1/auth/change-password)" || return 1
  [[ "$status" == "204" ]] || return 1
  status="$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $token" http://localhost:8080/api/v1/auth/me)" || return 1
  [[ "$status" == "401" ]] || return 1

  relogin_response="$(curl -fsS -H 'Content-Type: application/json' -d "{\"identifier\":\"admin\",\"password\":\"$new_password\"}" http://localhost:8080/api/v1/auth/login)" || return 1
  token="$(printf '%s' "$relogin_response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')" || return 1
  [[ -n "$token" ]] || return 1

  sessions="$(curl -fsS -H "Authorization: Bearer $token" http://localhost:8080/api/v1/auth/sessions/)" || return 1
  printf '%s' "$sessions" | python3 -c 'import json,sys; data=json.load(sys.stdin); assert any(item.get("current") for item in data)' >/dev/null || return 1
  curl -fsS -H "Authorization: Bearer $token" http://localhost:8080/api/v1/auth/me >/dev/null || return 1

  curl -fsS -X POST -H "Authorization: Bearer $token" http://localhost:8080/api/v1/auth/logout >/dev/null || return 1
  status="$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $token" http://localhost:8080/api/v1/auth/me)" || return 1
  [[ "$status" == "401" ]] || return 1

  relogin_response="$(curl -fsS -H 'Content-Type: application/json' -d "{\"identifier\":\"admin\",\"password\":\"$new_password\"}" http://localhost:8080/api/v1/auth/login)" || return 1
  token="$(printf '%s' "$relogin_response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')" || return 1

  setup_response="$(curl -fsS -X POST -H "Authorization: Bearer $token" http://localhost:8080/api/v1/auth/totp/setup)" || return 1
  secret="$(printf '%s' "$setup_response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["secret"])')" || return 1
  provider="$(printf '%s' "$setup_response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["provider"])')" || return 1
  [[ "$provider" == "google_authenticator" && -n "$secret" ]] || return 1
  totp_code="$(totp_now "$secret")" || return 1
  status="$(curl -sS -o /dev/null -w '%{http_code}' -X POST -H "Authorization: Bearer $token" -H 'Content-Type: application/json' -d "{\"code\":\"$totp_code\"}" http://localhost:8080/api/v1/auth/totp/enable)" || return 1
  [[ "$status" == "204" ]] || return 1

  status="$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $token" http://localhost:8080/api/v1/auth/me)" || return 1
  [[ "$status" == "401" ]] || return 1
  status="$(curl -sS -o /tmp/lyrahcle-login-no-totp.json -w '%{http_code}' -H 'Content-Type: application/json' -d "{\"identifier\":\"admin\",\"password\":\"$new_password\"}" http://localhost:8080/api/v1/auth/login)" || return 1
  [[ "$status" == "401" ]] || return 1
  python3 -c 'import json; assert json.load(open("/tmp/lyrahcle-login-no-totp.json")).get("error") == "totp_required"' || return 1

  totp_code="$(totp_now "$secret")" || return 1
  totp_login_response="$(curl -fsS -H 'Content-Type: application/json' -d "{\"identifier\":\"admin\",\"password\":\"$new_password\",\"totp_code\":\"$totp_code\"}" http://localhost:8080/api/v1/auth/login)" || return 1
  totp_token="$(printf '%s' "$totp_login_response" | python3 -c 'import json,sys; print(json.load(sys.stdin)["token"])')" || return 1
  [[ -n "$totp_token" ]] || return 1

  status="$(curl -sS -o /dev/null -w '%{http_code}' -X POST -H "Authorization: Bearer $totp_token" -H 'Content-Type: application/json' -d "{\"password\":\"$new_password\"}" http://localhost:8080/api/v1/auth/totp/disable)" || return 1
  [[ "$status" == "204" ]] || return 1
  status="$(curl -sS -o /dev/null -w '%{http_code}' -H "Authorization: Bearer $totp_token" http://localhost:8080/api/v1/auth/me)" || return 1
  [[ "$status" == "401" ]] || return 1

  relogin_response="$(curl -fsS -H 'Content-Type: application/json' -d "{\"identifier\":\"admin\",\"password\":\"$new_password\"}" http://localhost:8080/api/v1/auth/login)" || return 1
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
