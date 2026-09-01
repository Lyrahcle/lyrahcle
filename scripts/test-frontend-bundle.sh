#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
FRONTEND_DIR="$ROOT_DIR/apps/frontend"

if [[ ! -d "$FRONTEND_DIR" ]]; then
  echo "frontend directory not found: $FRONTEND_DIR" >&2
  exit 2
fi

echo "==> Frontend unit tests"
(
  cd "$FRONTEND_DIR"
  npm test -- --run
)

echo
echo "==> Frontend production build"
(
  cd "$FRONTEND_DIR"
  npm run build
)

echo
echo "==> Frontend bundle regression budget"
python3 "$ROOT_DIR/scripts/check-frontend-bundle.py" --require-build "$FRONTEND_DIR/dist"
