#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKEND="$ROOT/apps/backend"

echo "==> Backend audit/config security contracts"
(
  cd "$BACKEND"
  go test ./internal/audit ./internal/config -count=1
)

echo
echo "==> Backend audit/config vet"
(
  cd "$BACKEND"
  go vet ./internal/audit ./internal/config
)

echo "backend audit/config contracts: OK"
