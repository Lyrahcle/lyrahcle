#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/certs"
mkdir -p "$OUT_DIR"

echo "Generating self-signed TLS certificate for local development in $OUT_DIR"

openssl req -x509 -newkey rsa:4096 -sha256 -days 365 -nodes \
  -keyout "$OUT_DIR/dev.key" \
  -out "$OUT_DIR/dev.crt" \
  -subj "/CN=localhost" \
  -addext "subjectAltName=DNS:localhost,DNS:backend,DNS:frontend,IP:127.0.0.1"

echo "Done."
echo "  Certificate: $OUT_DIR/dev.crt"
echo "  Private key: $OUT_DIR/dev.key"
echo ""
echo "This certificate is self-signed and for local development ONLY."
echo "In production, use certificates from a trusted CA (e.g. Let's Encrypt)"
echo "or terminate TLS at a reverse proxy (Caddy, Traefik, nginx) in front of Lyrahcle."
echo ""
echo "To use it with the backend, set in docker-compose.yml or your environment:"
echo "  TLS_CERT_FILE=/certs/dev.crt"
echo "  TLS_KEY_FILE=/certs/dev.key"
