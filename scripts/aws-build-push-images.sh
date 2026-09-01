#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INFRA="$ROOT/deploy/terraform/aws"
TAG="${1:-}"
OUTPUT="${2:-/tmp/lyrahcle-aws-images.json}"

if [[ -z "$TAG" ]]; then
  echo "usage: $0 <immutable-tag> [output-json]" >&2
  exit 2
fi

if [[ ! "$TAG" =~ ^[A-Za-z0-9][A-Za-z0-9_.-]{0,127}$ ]]; then
  echo "ERROR: image tag is not OCI-compatible." >&2
  exit 2
fi

for command_name in aws docker python3; do
  if ! command -v "$command_name" >/dev/null 2>&1; then
    echo "ERROR: missing required command: $command_name" >&2
    exit 1
  fi
done

if command -v tofu >/dev/null 2>&1; then
  IAC=tofu
elif command -v terraform >/dev/null 2>&1; then
  IAC=terraform
else
  echo "ERROR: install OpenTofu or Terraform first." >&2
  exit 1
fi

cd "$INFRA"
aws sts get-caller-identity >/dev/null
REGION="$($IAC output -raw aws_region)"
REPOSITORIES="$($IAC output -json ecr_repository_urls)"

readarray -t IMAGE_DATA < <(python3 - "$REPOSITORIES" <<'PY'
import json
import sys
repos = json.loads(sys.argv[1])
for key in ("backend", "frontend", "cve-sync"):
    url = repos[key]
    print(f"{key}|{url}|{url.split('/', 1)[1]}")
PY
)

REGISTRY="$(printf '%s\n' "${IMAGE_DATA[0]}" | cut -d'|' -f2 | cut -d'/' -f1)"
aws ecr get-login-password --region "$REGION" | docker login --username AWS --password-stdin "$REGISTRY" >/dev/null

for row in "${IMAGE_DATA[@]}"; do
  REPOSITORY_NAME="$(printf '%s\n' "$row" | cut -d'|' -f3)"
  if aws ecr describe-images --region "$REGION" --repository-name "$REPOSITORY_NAME" --image-ids "imageTag=$TAG" >/dev/null 2>&1; then
    echo "ERROR: immutable ECR tag already exists: $REPOSITORY_NAME:$TAG" >&2
    exit 1
  fi
done

cd "$ROOT"
docker build -f deploy/docker/Dockerfile.backend -t "$(printf '%s\n' "${IMAGE_DATA[0]}" | cut -d'|' -f2):$TAG" .
docker build -f deploy/docker/Dockerfile.frontend -t "$(printf '%s\n' "${IMAGE_DATA[1]}" | cut -d'|' -f2):$TAG" .
docker build -f deploy/docker/Dockerfile.cve-sync -t "$(printf '%s\n' "${IMAGE_DATA[2]}" | cut -d'|' -f2):$TAG" .

for row in "${IMAGE_DATA[@]}"; do
  REPOSITORY_URL="$(printf '%s\n' "$row" | cut -d'|' -f2)"
  docker push "$REPOSITORY_URL:$TAG"
done

cd "$INFRA"
python3 - "$TAG" "$OUTPUT" "$REGION" "$REPOSITORIES" <<'PY'
import json
import subprocess
import sys
from pathlib import Path

tag, output, region, repos_json = sys.argv[1:]
repos = json.loads(repos_json)
result = {"tag": tag}
for key in ("backend", "frontend", "cve-sync"):
    repository = repos[key]
    repository_name = repository.split("/", 1)[1]
    digest = subprocess.check_output([
        "aws", "ecr", "describe-images", "--region", region,
        "--repository-name", repository_name,
        "--image-ids", f"imageTag={tag}",
        "--query", "imageDetails[0].imageDigest", "--output", "text",
    ], text=True).strip()
    if not digest.startswith("sha256:"):
        raise SystemExit(f"invalid digest returned for {key}: {digest}")
    result[key] = {"repository": repository, "digest": digest}
Path(output).write_text(json.dumps(result, indent=2) + "\n")
print(output)
PY
