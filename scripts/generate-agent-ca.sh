#!/usr/bin/env bash
set -euo pipefail

OUT_DIR="${1:-./certs/agents}"
mkdir -p "$OUT_DIR"
umask 077

openssl ecparam -name prime256v1 -genkey -noout -out "$OUT_DIR/agent-ca.key"
openssl req -x509 -new -sha256 -key "$OUT_DIR/agent-ca.key" -days 3650 -subj "/O=Lyrahcle/CN=Lyrahcle SDR Agent CA" -addext "basicConstraints=critical,CA:TRUE" -addext "keyUsage=critical,keyCertSign,cRLSign" -out "$OUT_DIR/agent-ca.crt"
chmod 600 "$OUT_DIR/agent-ca.key"
chmod 644 "$OUT_DIR/agent-ca.crt"
printf '%s\n' "$OUT_DIR/agent-ca.crt" "$OUT_DIR/agent-ca.key"
