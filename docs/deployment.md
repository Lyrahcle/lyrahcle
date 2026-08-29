# Deployment

## Production checklist

Before exposing Lyrahcle outside a trusted development environment:

- Set `ENV=production`.
- Use a strong random `JWT_SECRET`.
- Set non-default `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD` before first boot.
- Enable TLS and keep private keys outside the repository.
- Replace the default PostgreSQL credentials and do not expose PostgreSQL publicly.
- Restrict Redis to trusted networks and authentication controls appropriate to your environment.
- Issue one `AGENT_API_KEY` per SDR agent; do not reuse human JWTs for long-running agents.
- Configure durable PostgreSQL backups and test restoration.
- Review CI/security scan results before deploying a release.

There is no public self-registration. New users are created by an admin.

## Single-host Docker deployment

Create a production `.env` outside version control, then start the stack:

```bash
docker compose up -d --build
```

For development the frontend is exposed on `5173` and backend on `8080`. Production should normally put a reverse proxy or ingress in front of the application and avoid exposing database/cache ports.

## TLS

The backend can terminate TLS directly through `TLS_CERT_FILE` and `TLS_KEY_FILE`. A reverse proxy such as Caddy, Traefik, nginx, or a managed ingress can also terminate TLS in front of the application.

`scripts/generate-dev-certs.sh` is for local testing only. Configure `AGENT_CA_CERT` with the trusted development CA instead of disabling certificate verification.

## Database migrations

Backend startup applies ordered migrations from `apps/backend/migrations` and records them in `schema_migrations`. Migrations are forward-only; take a backup before deploying schema changes.

## Backups

Use the included backup script:

```bash
./scripts/backup-db.sh
BACKUP_DIR=/mnt/backups RETENTION_DAYS=30 ./scripts/backup-db.sh
```

Backup files are created with restrictive permissions and the script does not print `DATABASE_URL`. A backup is not complete until restoration has been tested in a separate database.

## Monitoring

| Endpoint | Purpose |
|---|---|
| `GET /healthz` | Process liveness |
| `GET /readyz` | Database/cache readiness without dependency details |
| `GET /api/v1/ping` | API reachability |
| `GET /metrics` | Backend Prometheus metrics |
| CVE worker `:9091/metrics` | CVE synchronization metrics |

Every backend response includes `X-Request-ID`. Preserve that value in reverse-proxy/application logs so operator reports can be correlated with backend activity.

## Updating

```bash
git pull origin main
docker compose build
docker compose up -d
```

For production, prefer immutable release tags/images rather than deploying an arbitrary `main` commit. Read [RELEASING.md](../RELEASING.md) for the release flow.

## Kubernetes

The repository contains deployment scaffolding, but the Kubernetes/Helm path is not yet the canonical production deployment. Treat those files as evolving until a tested HA deployment is documented.
