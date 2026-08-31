# Deployment

## Production checklist

Before exposing Lyrahcle outside a trusted development environment:

- Set `ENV=production`.
- Use a random `JWT_SECRET` of at least 32 bytes and keep `JWT_ACCESS_TTL` short.
- Set non-default `SEED_ADMIN_EMAIL` and a policy-compliant `SEED_ADMIN_PASSWORD` before first boot. The bootstrap admin is still forced to rotate that password on first login.
- Enable TLS and keep private keys outside the repository.
- Replace the default PostgreSQL credentials and do not expose PostgreSQL publicly.
- Restrict Redis to trusted networks and authentication controls appropriate to your environment; production startup fails when Redis is unavailable because session revocation is security-critical.
- Configure the SDR Agent CA and enroll long-running agents with mTLS. Keep legacy `AGENT_API_KEY` credentials only for controlled migration; do not use human JWTs for unattended agents.
- Configure durable PostgreSQL backups, keep restore evidence, and exercise recovery against an isolated database. Before v1.0, also record production-like DR RPO/RTO.
- Review CI/security scan results and the release SBOM before deploying a release.
- Keep `ORBITAL_AUTO_ESCALATION_ENABLED=false` until Doppler/visibility behavior has been validated with representative physical SDR captures for the deployment.

There is no public self-registration. New users are created by an admin.

## Single-host Docker deployment

Create a production `.env` outside version control, then start the stack:

```bash
docker compose up -d --build
```

For development the frontend is exposed on `5173` and backend on `8080`. Production should normally put a reverse proxy or ingress in front of the application and avoid exposing database/cache ports.


## Kubernetes / Helm

For multi-node and future managed-Kubernetes deployments, use the cloud-neutral
Helm chart documented in [Kubernetes deployment](./kubernetes-deployment.md).
The chart includes probes, rolling updates, optional HPA/PDB/topology spread,
separate least-privilege ServiceAccounts and serialized PostgreSQL migrations
for multi-replica backend startup.

The future AWS mapping is documented separately in
[AWS EKS migration](./aws-eks-migration.md); the base chart deliberately does
not require AWS-specific APIs or controllers.

## TLS

The backend can terminate TLS directly through `TLS_CERT_FILE` and `TLS_KEY_FILE`. A reverse proxy such as Caddy, Traefik, nginx, or a managed ingress can also terminate TLS in front of the application.

`scripts/generate-dev-certs.sh` is for local testing only. Configure `AGENT_CA_CERT` with the trusted development CA instead of disabling certificate verification.

## JWT signing-key rotation

For a rotation without intentionally accepting long-lived tokens:

1. Keep the existing key as `JWT_SECRET_PREVIOUS`.
2. Generate a new random key of at least 32 bytes and set it as `JWT_SECRET`.
3. Deploy the backend. New tokens use only the new key; existing v8.1+ tokens can continue through the previous-key fallback.
4. Wait longer than `JWT_ACCESS_TTL`.
5. Remove `JWT_SECRET_PREVIOUS` and deploy again.

A deployment from a pre-v8.1 build invalidates existing JWTs because older tokens do not contain the stricter issuer/audience/session claims. v8.3 also introduces a per-user authorization version in JWTs, so upgrading from v8.2 or earlier requires one user re-authentication. Plan this explicitly during deployment.

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

For local or single-host observability, run `docker compose --profile observability up -d`. This provisions Grafana on port `3000` with the Lyrahcle overview dashboard and loads Prometheus alert rules. Override the development Grafana administrator password before using the profile outside an isolated local environment.

## Updating

```bash
git pull origin main
docker compose build
docker compose up -d
```

For production, prefer immutable release tags/images rather than deploying an arbitrary `main` commit. Read [RELEASING.md](../RELEASING.md) for the release flow.

## Kubernetes

The repository contains deployment scaffolding, but the Kubernetes/Helm path is not yet the canonical production deployment. Treat those files as evolving until a tested HA deployment is documented.

## WebAuthn and recovery deployment

For security-key authentication, set `WEBAUTHN_RP_ID` to the frontend domain and `WEBAUTHN_ORIGINS` to the exact HTTPS origins users access. WebAuthn should not be deployed behind origin-rewriting proxies unless the browser-visible origin remains stable.

Password recovery remains disabled until `PASSWORD_RECOVERY_ENABLED=true` and the TLS SMTP variables are configured. Use a dedicated SMTP credential with the minimum permissions required for recovery mail. Do not place SMTP passwords in source control; inject them through the deployment secret store.

If `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` is configured in production it must use HTTPS. Treat the collector as security-sensitive infrastructure and keep authentication secrets and reset links out of span attributes.

## SDR agent mTLS

Production SDR agents should use the v9.0 enrollment flow instead of static API keys. Configure a dedicated SDR Agent CA with `AGENT_CA_CERT_FILE` and `AGENT_CA_KEY_FILE`, keep the backend on HTTPS, and mount the CA signing key from a protected secret store. The Go TLS server validates optional client certificates directly; agent endpoints then require the certificate to map to an `active` database identity.

Do not forward client-certificate identity through an untrusted HTTP header. If TLS is terminated before the Go backend, preserve end-to-end mTLS or deploy a separately authenticated proxy integration rather than trusting arbitrary forwarded certificate metadata.

See [`agent-enrollment.md`](./agent-enrollment.md).
