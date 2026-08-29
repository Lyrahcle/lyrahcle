#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="${1:-$ROOT_DIR/artifacts/sbom}"
GOMOD_VERSION="${CYCLONEDX_GOMOD_VERSION:-v1.10.0}"
NPM_VERSION="${CYCLONEDX_NPM_VERSION:-6.0.1}"

mkdir -p "$OUT_DIR"
OUT_DIR="$(cd "$OUT_DIR" && pwd)"

go install "github.com/CycloneDX/cyclonedx-gomod/cmd/cyclonedx-gomod@${GOMOD_VERSION}"

for module in backend cve-sync sdr-agent; do
  cyclonedx-gomod mod -json -output-version 1.6 -output "$OUT_DIR/${module}.cdx.json" "$ROOT_DIR/apps/$module"
done

(
  cd "$ROOT_DIR/apps/frontend"
  npm ci
  npx --yes "@cyclonedx/cyclonedx-npm@${NPM_VERSION}" --output-format JSON --output-file "$OUT_DIR/frontend.cdx.json"
)

(
  cd "$OUT_DIR"
  sha256sum ./*.cdx.json > SHA256SUMS
)
