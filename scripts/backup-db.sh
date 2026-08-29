#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BACKUP_DIR="${BACKUP_DIR:-$ROOT_DIR/backups}"
RETENTION_DAYS="${RETENTION_DAYS:-14}"
TIMESTAMP="$(date -u +%Y%m%dT%H%M%SZ)"

DATABASE_URL="${DATABASE_URL:-postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable}"

mkdir -p "$BACKUP_DIR"

OUT_FILE="$BACKUP_DIR/lyrahcle-$TIMESTAMP.sql.gz"

echo "==> Backing up $DATABASE_URL to $OUT_FILE"

if command -v docker >/dev/null 2>&1 && docker compose ps postgres >/dev/null 2>&1; then
  docker compose exec -T postgres pg_dump -U lyrahcle lyrahcle | gzip > "$OUT_FILE"
else
  pg_dump "$DATABASE_URL" | gzip > "$OUT_FILE"
fi

SIZE="$(du -h "$OUT_FILE" | cut -f1)"
echo "==> Wrote $OUT_FILE ($SIZE)"

echo "==> Pruning backups older than $RETENTION_DAYS days"
find "$BACKUP_DIR" -name 'lyrahcle-*.sql.gz' -mtime +"$RETENTION_DAYS" -print -delete

echo "==> Done. $(find "$BACKUP_DIR" -name 'lyrahcle-*.sql.gz' | wc -l) backup(s) retained in $BACKUP_DIR"
