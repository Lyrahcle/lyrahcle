#!/usr/bin/env bash
# Detects drift between the Terraform/OpenTofu AWS root module and the real
# infrastructure. Read-only: runs `plan -detailed-exitcode -lock=false`
# against the configured remote state and never applies anything.
#
# Requires AWS credentials already in the environment (OIDC-assumed role in
# CI, or `aws sso login` / aws-vault locally) and a real backend.hcl with
# your state bucket/key/region (see backend.hcl.example — never commit the
# real one with account-specific values).
#
# Exit codes (matches `terraform plan -detailed-exitcode`):
#   0 - succeeded, no drift
#   1 - error
#   2 - succeeded, drift detected
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INFRA="$ROOT/deploy/terraform/aws"
BACKEND_CONFIG="${TF_BACKEND_CONFIG:-$INFRA/backend.hcl}"

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

if [[ ! -f "$BACKEND_CONFIG" ]]; then
  echo "ERROR: backend config not found at $BACKEND_CONFIG" >&2
  echo "Copy backend.hcl.example to backend.hcl (or set TF_BACKEND_CONFIG) with your real state bucket/key/region." >&2
  exit 1
fi

echo "==> Verifying AWS identity"
aws sts get-caller-identity

cd "$INFRA"
echo "==> $IAC init (real backend: $BACKEND_CONFIG)"
"$IAC" init -input=false -backend-config="$BACKEND_CONFIG"

echo "==> $IAC plan -detailed-exitcode (read-only, no lock)"
set +e
"$IAC" plan -input=false -lock=false -detailed-exitcode -out=/dev/null "$@"
status=$?
set -e

case "$status" in
  0)
    echo "==> No drift detected."
    ;;
  2)
    echo "==> DRIFT DETECTED: the live AWS infrastructure no longer matches the Terraform state/code." >&2
    echo "Review the plan above, then either apply the intended change or update the code to match reality." >&2
    ;;
  *)
    echo "==> $IAC plan failed (exit $status)." >&2
    ;;
esac

exit "$status"
