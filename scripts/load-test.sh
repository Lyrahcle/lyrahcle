#!/usr/bin/env bash
set -euo pipefail

if [[ "${LOAD_TEST_ACK:-}" != "1" ]]; then
  echo "Set LOAD_TEST_ACK=1 to confirm this target is authorized for load testing." >&2
  exit 2
fi

command -v hey >/dev/null 2>&1 || { echo "hey is required: https://github.com/rakyll/hey" >&2; exit 2; }

BASE_URL="${BASE_URL:-http://localhost:8080}"
REQUESTS="${REQUESTS:-1000}"
CONCURRENCY="${CONCURRENCY:-25}"
TOKEN="${TOKEN:-}"

if [[ "$REQUESTS" -gt 100000 || "$CONCURRENCY" -gt 500 ]]; then
  echo "REQUESTS must be <= 100000 and CONCURRENCY <= 500." >&2
  exit 2
fi

hey -n "$REQUESTS" -c "$CONCURRENCY" "$BASE_URL/healthz"

if [[ -n "$TOKEN" ]]; then
  hey -n "$REQUESTS" -c "$CONCURRENCY" -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/satellites/"
  hey -n "$REQUESTS" -c "$CONCURRENCY" -H "Authorization: Bearer $TOKEN" "$BASE_URL/api/v1/search?q=sat"
fi
