#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INFRA="$ROOT/deploy/terraform/aws"

if command -v tofu >/dev/null 2>&1; then
  IAC=tofu
elif command -v terraform >/dev/null 2>&1; then
  IAC=terraform
else
  echo "ERROR: install OpenTofu or Terraform first." >&2
  exit 1
fi

if ! command -v aws >/dev/null 2>&1; then
  echo "ERROR: aws CLI is required to verify the active identity." >&2
  exit 1
fi

aws sts get-caller-identity >/dev/null

cd "$INFRA"
"$IAC" fmt -check -recursive
"$IAC" init -backend=false
"$IAC" validate
"$IAC" plan "$@"
