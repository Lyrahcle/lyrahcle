#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FULL=0
DB_URL="${TEST_DATABASE_URL:-}"
REDIS_ADDR="${TEST_REDIS_ADDR:-}"
if [[ "${1:-}" == "--full" ]]; then
  FULL=1
fi

section() { printf '\n\033[1;34m==> %s\033[0m\n' "$1"; }

require_tcp_endpoint() {
  local label="$1"
  local host="$2"
  local port="$3"
  if ! python3 - "$host" "$port" <<'PY'
import socket, sys
host, port = sys.argv[1], int(sys.argv[2])
try:
    with socket.create_connection((host, port), timeout=2):
        pass
except OSError as exc:
    print(f"{exc}", file=sys.stderr)
    raise SystemExit(1)
PY
  then
    printf '\nERROR: %s is not reachable at %s:%s.\n' "$label" "$host" "$port" >&2
    printf 'Start the local test service or unset its TEST_* environment variable before running the gate.\n' >&2
    return 1
  fi
}

preflight_services() {
  if [[ -n "$DB_URL" ]]; then
    local db_endpoint
    db_endpoint="$(python3 - "$DB_URL" <<'PY'
import sys
from urllib.parse import urlparse
u = urlparse(sys.argv[1])
if not u.hostname:
    raise SystemExit("TEST_DATABASE_URL has no hostname")
print(f"{u.hostname}\t{u.port or 5432}")
PY
)"
    local db_host db_port
    IFS=$'\t' read -r db_host db_port <<< "$db_endpoint"
    require_tcp_endpoint "PostgreSQL test service" "$db_host" "$db_port"
  fi

  if [[ -n "$REDIS_ADDR" ]]; then
    local redis_host redis_port
    if [[ "$REDIS_ADDR" == \[*\]:* ]]; then
      redis_host="${REDIS_ADDR#\[}"
      redis_host="${redis_host%%\]*}"
      redis_port="${REDIS_ADDR##*:}"
    else
      redis_host="${REDIS_ADDR%:*}"
      redis_port="${REDIS_ADDR##*:}"
    fi
    if [[ -z "$redis_host" || "$redis_host" == "$REDIS_ADDR" || -z "$redis_port" ]]; then
      printf 'ERROR: TEST_REDIS_ADDR must be host:port (got %q).\n' "$REDIS_ADDR" >&2
      return 1
    fi
    require_tcp_endpoint "Redis test service" "$redis_host" "$redis_port"
  fi
}

section "Local service preflight"
preflight_services
printf 'Configured integration services are reachable.\n'

section "Security and parity contracts"
cd "$ROOT"
python3 -m unittest \
  scripts/test_check_command_execution.py \
  scripts/test_check_frontend_xss.py \
  scripts/test_check_sql_construction.py \
  scripts/test_check_go_toolchain.py \
  scripts/test_check_asset_program_parity.py \
  scripts/test_check_platform_organization_parity.py \
  scripts/test_check_satellite_ui_parity.py \
  scripts/test_check_correlation_ui_parity.py \
  scripts/test_check_http_error_ui.py \
  scripts/test_check_keyprovider_kid.py \
  scripts/test_check_signing_key_lifecycle.py \
  scripts/test_check_signing_key_automation.py \
  scripts/test_check_tenant_rate_limiting.py \
  scripts/test_check_background_job_idempotency.py \
  scripts/test_check_agent_hardware_ui.py \
  scripts/test_check_route_authorization.py \
  scripts/test_check_csrf_cookie_auth.py
  python3 -m unittest scripts/test_check_script_permissions.py
python3 scripts/check-command-execution.py .
python3 scripts/check-frontend-xss.py .
python3 scripts/check-sql-construction.py .
python3 scripts/check-go-toolchain.py .
python3 scripts/check-asset-program-parity.py .
python3 scripts/check-platform-organization-parity.py .
python3 scripts/check-satellite-ui-parity.py .
python3 scripts/check-correlation-ui-parity.py .
python3 scripts/check-http-error-ui.py .
python3 scripts/check-keyprovider-kid.py .
python3 scripts/check-signing-key-lifecycle.py .
python3 scripts/check-signing-key-automation.py .
python3 scripts/check-tenant-rate-limiting.py .
python3 scripts/check-background-job-idempotency.py .
python3 scripts/check-agent-hardware-ui.py .
python3 scripts/check-route-authorization.py .
python3 scripts/check-csrf-cookie-auth.py .
python3 scripts/check-script-permissions.py .

section "Backend tests"
cd "$ROOT/apps/backend"
# Unit tests must not become integration tests merely because the caller has
# TEST_DATABASE_URL / TEST_REDIS_ADDR exported in their shell.
TEST_DATABASE_URL= TEST_REDIS_ADDR= go test ./...

if [[ -n "$REDIS_ADDR" ]]; then
  section "Backend Redis integration tests"
  TEST_DATABASE_URL= TEST_REDIS_ADDR="$REDIS_ADDR" go test \
    ./internal/ratelimit/... \
    ./internal/joblock/... \
    ./internal/alerts/... \
    ./internal/respcache/... \
    ./internal/sessionstore/... \
    ./internal/tokenblocklist/... -v
else
  printf 'TEST_REDIS_ADDR is not set; Redis-backed integration tests will be skipped.\n'
fi

if [[ -n "$DB_URL" ]]; then
  section "Backend PostgreSQL integration tests"
  TEST_DATABASE_URL="$DB_URL" TEST_REDIS_ADDR="$REDIS_ADDR" go test ./internal/store/... -v
  TEST_DATABASE_URL="$DB_URL" TEST_REDIS_ADDR="$REDIS_ADDR" go test ./cmd/server/... -v
else
  printf 'TEST_DATABASE_URL is not set; DB-backed integration tests will be skipped by the test harness.\n'
fi

section "CVE sync tests"
cd "$ROOT/apps/cve-sync"
go test ./...

section "SDR agent tests"
cd "$ROOT/apps/sdr-agent"
go test ./...

section "Frontend tests and production build"
cd "$ROOT/apps/frontend"
npm test
npm run build

if [[ "$FULL" -eq 1 ]]; then
  section "Full Go race/vet gate"
  cd "$ROOT/apps/backend"
  TEST_DATABASE_URL= TEST_REDIS_ADDR= go test -race ./...
  go vet ./...
  cd "$ROOT/apps/cve-sync"
  go test -race ./...
  go vet ./...
  cd "$ROOT/apps/sdr-agent"
  go test -race ./...
  go vet ./...
fi

if command -v docker >/dev/null 2>&1; then
  section "Docker Compose configuration"
  cd "$ROOT"
  docker compose --profile observability config -q
fi

section "Critical backend security coverage"
(
  cd "$ROOT/apps/backend"
  go test ./internal/keyprovider/... -coverprofile=coverage-keyprovider.out
  ../../scripts/check-go-coverage.sh coverage-keyprovider.out 80
  rm -f coverage-keyprovider.out

  go test ./internal/auth/... -coverprofile=coverage-auth.out
  ../../scripts/check-go-coverage.sh coverage-auth.out 80
  rm -f coverage-auth.out
)

section "Agent trust-boundary coverage"
(
  cd "$ROOT/apps/backend"

  go test ./internal/agentidentity/... -coverprofile=coverage-agentidentity.out
  ../../scripts/check-go-coverage.sh coverage-agentidentity.out 80
  rm -f coverage-agentidentity.out

  go test ./internal/agentconfig/... -coverprofile=coverage-agentconfig.out
  ../../scripts/check-go-coverage.sh coverage-agentconfig.out 80
  rm -f coverage-agentconfig.out
)

if [[ -n "$REDIS_ADDR" ]]; then
  section "Session and token revocation coverage"
  (
    cd "$ROOT/apps/backend"
    TEST_REDIS_ADDR="$REDIS_ADDR" go test ./internal/sessionstore/... -coverprofile=coverage-sessionstore.out
    ../../scripts/check-go-coverage.sh coverage-sessionstore.out 80
    rm -f coverage-sessionstore.out

    TEST_REDIS_ADDR="$REDIS_ADDR" go test ./internal/tokenblocklist/... -coverprofile=coverage-tokenblocklist.out
    ../../scripts/check-go-coverage.sh coverage-tokenblocklist.out 80
    rm -f coverage-tokenblocklist.out
  )
else
  printf 'TEST_REDIS_ADDR is not set; session/token coverage gates requiring Redis are skipped.\n'
fi

section "Tenant scope and rate-limit coverage"
(
  cd "$ROOT/apps/backend"
  go test ./internal/dbscope/... -coverprofile=coverage-dbscope.out
  ../../scripts/check-go-coverage.sh coverage-dbscope.out 80
  rm -f coverage-dbscope.out

  if [[ -n "$REDIS_ADDR" ]]; then
    TEST_REDIS_ADDR="$REDIS_ADDR" go test ./internal/ratelimit/... -coverprofile=coverage-ratelimit.out
    ../../scripts/check-go-coverage.sh coverage-ratelimit.out 80
    rm -f coverage-ratelimit.out
  else
    printf 'TEST_REDIS_ADDR is not set; distributed rate-limit coverage is skipped.\n'
  fi
)

section "Scheduler and singleton-job HA coverage"
(
  cd "$ROOT/apps/backend"
  go test ./internal/scheduler/... -coverprofile=coverage-scheduler.out
  ../../scripts/check-go-coverage.sh coverage-scheduler.out 80
  rm -f coverage-scheduler.out

  if [[ -n "$REDIS_ADDR" ]]; then
    TEST_REDIS_ADDR="$REDIS_ADDR" go test ./internal/joblock/... -coverprofile=coverage-joblock.out
    ../../scripts/check-go-coverage.sh coverage-joblock.out 80
    rm -f coverage-joblock.out
  else
    printf 'TEST_REDIS_ADDR is not set; distributed job-lock coverage is skipped.\n'
  fi
)

section "Local v1 test gate passed"
