#!/usr/bin/env bash
set -euo pipefail
umask 077

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_DATABASE="lyrahcle_test"
TARGET_DATABASE="lyrahcle_dr_$(date -u +%Y%m%d%H%M%S)_$$"
TMP="$(mktemp -d)"
BACKUP="$TMP/lyrahcle-test.sql.gz"
SOURCE_SNAPSHOT="$TMP/source.tsv"
TARGET_SNAPSHOT="$TMP/target.tsv"
REPORT="/tmp/lyrahcle-dr-restore-report.json"
TARGET_CREATED=0

cleanup() {
  if [[ "$TARGET_CREATED" -eq 1 ]]; then
    docker compose exec -T postgres dropdb --force -U lyrahcle "$TARGET_DATABASE" >/dev/null 2>&1 || true
  fi
  rm -rf "$TMP"
}
trap cleanup EXIT

for command in docker gzip python3; do
  command -v "$command" >/dev/null 2>&1 || {
    printf 'missing required command: %s\n' "$command" >&2
    exit 1
  }
done

cd "$ROOT"
docker compose up -d postgres >/dev/null
if ! docker compose exec -T postgres psql -At -U lyrahcle -d postgres -c "SELECT 1 FROM pg_database WHERE datname='$SOURCE_DATABASE'" | grep -Fxq 1; then
  printf 'source test database does not exist: %s\n' "$SOURCE_DATABASE" >&2
  exit 1
fi

snapshot_database() {
  local database="$1"
  docker compose exec -T postgres psql -Aqt -F $'\t' -v ON_ERROR_STOP=1 -U lyrahcle -d "$database" <<'SQL'
CREATE TEMP TABLE resilience_counts(name text primary key, row_count bigint not null);
DO $$
DECLARE
  r record;
BEGIN
  FOR r IN
    SELECT schemaname, tablename
    FROM pg_tables
    WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
    ORDER BY schemaname, tablename
  LOOP
    EXECUTE format('INSERT INTO resilience_counts(name, row_count) SELECT %L, count(*) FROM %I.%I', r.schemaname || '.' || r.tablename, r.schemaname, r.tablename);
  END LOOP;
END
$$;
SELECT name, row_count FROM resilience_counts ORDER BY name;
SQL
}

start_backup="$(python3 - <<'PY'
import time
print(time.monotonic())
PY
)"
docker compose exec -T postgres pg_dump -U lyrahcle "$SOURCE_DATABASE" | gzip > "$BACKUP"
"$ROOT/scripts/verify-backup.sh" "$BACKUP" >/dev/null
end_backup="$(python3 - <<'PY'
import time
print(time.monotonic())
PY
)"

snapshot_database "$SOURCE_DATABASE" > "$SOURCE_SNAPSHOT"

docker compose exec -T postgres createdb -U lyrahcle "$TARGET_DATABASE"
TARGET_CREATED=1
start_restore="$(python3 - <<'PY'
import time
print(time.monotonic())
PY
)"
gzip -cd "$BACKUP" | docker compose exec -T postgres psql -q -v ON_ERROR_STOP=1 -U lyrahcle "$TARGET_DATABASE"
end_restore="$(python3 - <<'PY'
import time
print(time.monotonic())
PY
)"
snapshot_database "$TARGET_DATABASE" > "$TARGET_SNAPSHOT"

if ! cmp -s "$SOURCE_SNAPSHOT" "$TARGET_SNAPSHOT"; then
  printf 'restored database table inventory or row counts differ from the backup source\n' >&2
  diff -u "$SOURCE_SNAPSHOT" "$TARGET_SNAPSHOT" >&2 || true
  exit 1
fi

migration_source="$(docker compose exec -T postgres psql -At -U lyrahcle -d "$SOURCE_DATABASE" -c 'SELECT count(*) FROM schema_migrations')"
migration_target="$(docker compose exec -T postgres psql -At -U lyrahcle -d "$TARGET_DATABASE" -c 'SELECT count(*) FROM schema_migrations')"
if [[ "$migration_source" != "$migration_target" ]]; then
  printf 'schema_migrations mismatch: source=%s restored=%s\n' "$migration_source" "$migration_target" >&2
  exit 1
fi

table_count="$(wc -l < "$SOURCE_SNAPSHOT")"
python3 - "$start_backup" "$end_backup" "$start_restore" "$end_restore" "$table_count" "$migration_source" "$REPORT" <<'PY'
import json, sys
from datetime import datetime, timezone
backup_seconds = round(float(sys.argv[2]) - float(sys.argv[1]), 3)
restore_seconds = round(float(sys.argv[4]) - float(sys.argv[3]), 3)
report = {
    "generated_at": datetime.now(timezone.utc).isoformat(),
    "source_database": "lyrahcle_test",
    "target_database_scope": "temporary database removed after the drill",
    "backup_seconds": backup_seconds,
    "restore_seconds": restore_seconds,
    "tables_verified": int(sys.argv[5]),
    "migrations_verified": int(sys.argv[6]),
    "rpo_scope": "All rows present at the backup boundary were verified after restore. Production RPO depends on backup/PITR cadence.",
    "status": "PASS",
}
with open(sys.argv[7], "w", encoding="utf-8") as handle:
    json.dump(report, handle, indent=2)
    handle.write("\n")
PY

printf 'local DR restore test: OK (tables=%s migrations=%s)\n' "$table_count" "$migration_source"
printf 'report: %s\n' "$REPORT"
