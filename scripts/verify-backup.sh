#!/usr/bin/env bash
set -euo pipefail

BACKUP_FILE="${1:?backup file is required}"

if [[ ! -f "$BACKUP_FILE" ]]; then
  echo "backup file not found: $BACKUP_FILE" >&2
  exit 1
fi

if [[ ! -s "$BACKUP_FILE" ]]; then
  echo "backup file is empty: $BACKUP_FILE" >&2
  exit 1
fi

gzip -t "$BACKUP_FILE"

HEADER="$(gzip -cd "$BACKUP_FILE" | sed -n '1,8p')"
if ! grep -q '^-- PostgreSQL database dump' <<<"$HEADER"; then
  echo "backup does not look like a PostgreSQL dump" >&2
  exit 1
fi

SHA256="$(sha256sum "$BACKUP_FILE" | awk '{print $1}')"
SIZE="$(stat -c '%s' "$BACKUP_FILE")"
printf 'backup_ok sha256=%s bytes=%s file=%s\n' "$SHA256" "$SIZE" "$BACKUP_FILE"
