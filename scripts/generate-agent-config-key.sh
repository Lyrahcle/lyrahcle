#!/usr/bin/env bash
set -euo pipefail

OUT_DIR=${1:-certs/agent-config}
mkdir -p "$OUT_DIR"
umask 077
openssl genpkey -algorithm ED25519 -out "$OUT_DIR/signing.key"
openssl pkey -in "$OUT_DIR/signing.key" -pubout -out "$OUT_DIR/verify.pub"
chmod 600 "$OUT_DIR/signing.key"
chmod 644 "$OUT_DIR/verify.pub"
printf '%s\n' "$OUT_DIR/signing.key" "$OUT_DIR/verify.pub"
