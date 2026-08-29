# Backup and restore

Lyrahcle database backups are PostgreSQL logical dumps compressed with gzip. Backup files should be treated as sensitive security data because they can contain users, incidents, audit records, asset metadata and authentication state.

## Create and verify a backup

From the repository root:

```bash
./scripts/backup-db.sh
```

The script writes to `backups/` by default, applies restrictive file permissions, verifies gzip integrity, checks that the file is a PostgreSQL dump and prints its SHA-256 digest.

Set `BACKUP_DIR` to store backups on a dedicated encrypted volume and `RETENTION_DAYS` to change local retention.

## Verify an existing backup

```bash
./scripts/verify-backup.sh backups/lyrahcle-YYYYMMDDTHHMMSSZ.sql.gz
```

Verification checks container integrity and dump format but does not replace a restore drill.

## Restore

Restores require an explicit acknowledgement:

```bash
RESTORE_ACK=1 ./scripts/restore-db.sh backups/lyrahcle-YYYYMMDDTHHMMSSZ.sql.gz
```

Remote database restores are refused by default. Set `ALLOW_REMOTE_RESTORE=1` only after independently verifying the destination and maintenance window.

For a Docker Compose restore into a non-production drill database:

```bash
docker compose exec -T postgres createdb -U lyrahcle lyrahcle_restore_test
RESTORE_ACK=1 RESTORE_DATABASE=lyrahcle_restore_test ./scripts/restore-db.sh backups/example.sql.gz
```

CI performs the same backup/restore pattern against a temporary database during the full-stack smoke test.

## Production recommendations

Use encrypted storage, off-host replication and separate credentials for backup automation. Monitor backup age and size, regularly restore into an isolated environment, and record the tested recovery point and recovery time. Never publish database dumps as CI artifacts.
