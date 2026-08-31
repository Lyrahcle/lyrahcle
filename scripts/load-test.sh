#!/usr/bin/env bash
set -euo pipefail

if [[ "${LOAD_TEST_ACK:-}" != "1" ]]; then
  echo "Set LOAD_TEST_ACK=1 to confirm this target is authorized for load testing." >&2
  exit 2
fi

command -v hey >/dev/null 2>&1 || { echo "hey is required" >&2; exit 2; }

BASE_URL="${BASE_URL:-http://localhost:8080}"
REQUESTS="${REQUESTS:-1000}"
CONCURRENCY="${CONCURRENCY:-25}"
DURATION="${DURATION:-}"
TOKEN="${TOKEN:-}"
OUTPUT_DIR="${OUTPUT_DIR:-}"

if [[ "$REQUESTS" -gt 100000 || "$CONCURRENCY" -gt 500 ]]; then
  echo "REQUESTS must be <= 100000 and CONCURRENCY <= 500." >&2
  exit 2
fi

if [[ -n "$DURATION" && ! "$DURATION" =~ ^[0-9]+(s|m)$ ]]; then
  echo "DURATION must look like 30s or 2m." >&2
  exit 2
fi

run_case() {
  local name url
  name="$1"
  url="$2"
  shift 2
  local args=(-c "$CONCURRENCY")
  if [[ -n "$DURATION" ]]; then
    args+=(-z "$DURATION")
  else
    args+=(-n "$REQUESTS")
  fi
  if [[ -n "$OUTPUT_DIR" ]]; then
    mkdir -p "$OUTPUT_DIR"
    hey "${args[@]}" "$@" "$url" | tee "$OUTPUT_DIR/$name.txt"
  else
    hey "${args[@]}" "$@" "$url"
  fi
}

run_case health "$BASE_URL/healthz"

if [[ -n "$TOKEN" ]]; then
  auth=(-H "Authorization: Bearer $TOKEN")
  run_case satellites "$BASE_URL/api/v1/satellites/" "${auth[@]}"
  run_case global-search "$BASE_URL/api/v1/search?q=sat" "${auth[@]}"
  run_case incidents "$BASE_URL/api/v1/incidents/?limit=25&offset=0" "${auth[@]}"
fi
