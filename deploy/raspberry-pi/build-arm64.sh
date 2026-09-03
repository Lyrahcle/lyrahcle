#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUTPUT="${ROOT}/deploy/raspberry-pi/build/sdr-agent-linux-arm64"
mkdir -p "$(dirname "${OUTPUT}")"
cd "${ROOT}/apps/sdr-agent"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o "${OUTPUT}" ./cmd/agent
sha256sum "${OUTPUT}" > "${OUTPUT}.sha256"
echo "Built ${OUTPUT}"

