#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

python3 -m unittest discover -s scripts -p 'test_check_*.py'

for check in scripts/check-*.py; do
  python3 "$check" .
done
