#!/usr/bin/env bash
set -euo pipefail
umask 077

if [[ "${RESTORE_ACK:-}" != "1" ]]; then
  echo "Set RESTORE_ACK=1 to confirm this restore is intentional." >&2
  exit 2
fi

BACKUP_FILE="${1:?backup file is required}"
DATABASE_URL="${DATABASE_URL:-postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable}"
ALLOW_REMOTE_RESTORE="${ALLOW_REMOTE_RESTORE:-0}"
RESTORE_DATABASE="${RESTORE_DATABASE:-lyrahcle}"

if [[ ! "$RESTORE_DATABASE" =~ ^[A-Za-z0-9_]+$ ]]; then
  echo "invalid RESTORE_DATABASE" >&2
  exit 2
fi

"$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/verify-backup.sh" "$BACKUP_FILE"

if [[ "$ALLOW_REMOTE_RESTORE" != "1" && ! "$DATABASE_URL" =~ @(localhost|127\.0\.0\.1): ]]; then
  echo "remote restore refused; set ALLOW_REMOTE_RESTORE=1 only after verifying the target" >&2
  exit 2
fi

if command -v docker >/dev/null 2>&1 && docker compose ps postgres >/dev/null 2>&1; then
  gzip -cd "$BACKUP_FILE" | docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -U lyrahcle "$RESTORE_DATABASE"
else
  gzip -cd "$BACKUP_FILE" | psql -v ON_ERROR_STOP=1 "$DATABASE_URL"
fi

echo "restore completed"
