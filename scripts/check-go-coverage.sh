#!/usr/bin/env bash
set -euo pipefail

PROFILE="${1:?coverage profile is required}"
THRESHOLD="${2:?coverage threshold is required}"

COVERAGE="$(go tool cover -func="$PROFILE" | awk '/^total:/ {gsub(/%/, "", $3); print $3}')"
if [[ -z "$COVERAGE" ]]; then
  echo "coverage total not found" >&2
  exit 1
fi

awk -v coverage="$COVERAGE" -v threshold="$THRESHOLD" 'BEGIN { if (coverage + 0 < threshold + 0) exit 1 }' || {
  echo "coverage ${COVERAGE}% is below required ${THRESHOLD}%" >&2
  exit 1
}

echo "coverage ${COVERAGE}% meets required ${THRESHOLD}%"
