#!/usr/bin/env bash
set -euo pipefail
CLUSTER="${LYRAHCLE_KIND_CLUSTER:-lyrahcle}"
if ! command -v kind >/dev/null 2>&1; then
  printf 'missing required command: kind\n' >&2
  exit 1
fi
if kind get clusters 2>/dev/null | grep -Fxq "$CLUSTER"; then
  kind delete cluster --name "$CLUSTER"
else
  printf 'kind cluster %s does not exist\n' "$CLUSTER"
fi
