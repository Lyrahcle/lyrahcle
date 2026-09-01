#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

log_file="$(mktemp)"
trap 'rm -f "$log_file"' EXIT

echo "==> Building Lyrahcle local images"
if docker compose "$@" build 2>&1 | tee "$log_file"; then
  echo
  echo "==> Starting Lyrahcle"
  exec docker compose "$@" up -d
fi

if ! grep -Eqi 'registry-1\.docker\.io|failed to resolve source metadata|lookup .*docker\.io|i/o timeout|temporary failure in name resolution|no such host' "$log_file"; then
  echo >&2
  echo "ERROR: Docker build failed for a reason other than registry/DNS resolution." >&2
  exit 1
fi

echo >&2
echo "WARNING: Docker Hub registry/DNS resolution failed." >&2
echo "Trying the last successfully built local Lyrahcle images without contacting a registry..." >&2

# Migrate the default Compose image names used by older Lyrahcle checkouts so a
# workstation that has already built the stack can recover without Docker Hub.
for mapping in \
  "lyrahcle-main-backend:latest lyrahcle/backend:local" \
  "lyrahcle-main-frontend:latest lyrahcle/frontend:local" \
  "lyrahcle-main-cve-sync:latest lyrahcle/cve-sync:local" \
  "lyrahcle-main-sdr-agent:latest lyrahcle/sdr-agent:local"
do
  set -- $mapping
  if ! docker image inspect "$2" >/dev/null 2>&1 && docker image inspect "$1" >/dev/null 2>&1; then
    docker tag "$1" "$2"
    echo "  reused cached legacy image: $1 -> $2" >&2
  fi
done

missing=0
while IFS= read -r image
do
  [ -n "$image" ] || continue
  if ! docker image inspect "$image" >/dev/null 2>&1; then
    echo "  missing local image: $image" >&2
    missing=1
  fi
done < <(docker compose "$@" config --images | sort -u)

if [ "$missing" -ne 0 ]; then
  cat >&2 <<'MSG'

ERROR: Registry DNS is unavailable and at least one required image is not cached locally.
This cannot be repaired by application code because Docker must resolve/pull the missing base image.

Check from the host:
  getent hosts registry-1.docker.io
  docker info

If this is WSL and name resolution is also failing outside Docker, repair WSL/host DNS first and rerun this script.
No Lyrahcle source or database change is required.
MSG
  exit 1
fi

echo
echo "==> Starting from cached images (no build, no registry pull)"
exec docker compose "$@" up -d --no-build --pull never
