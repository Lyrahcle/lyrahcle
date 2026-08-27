# Deployment guide

This covers taking Lyrahcle from local `docker compose up` to a real
deployment. It assumes familiarity with Docker; it does not assume a specific
cloud provider.

## Minimum viable production checklist

Before exposing Lyrahcle to anything other than `localhost`:

- [ ] Set a strong `JWT_SECRET` (`openssl rand -hex 32`), not the insecure dev
      default. The backend logs a warning on every startup while it isn't set.
- [ ] Enable TLS — either terminate it at a reverse proxy (recommended, see
      below) or set `TLS_CERT_FILE`/`TLS_KEY_FILE` directly on the backend.
      Plain HTTP logs a warning on every startup while it's active.
- [ ] Change the Postgres password from the `docker-compose.yml` default
      (`lyrahcle`/`lyrahcle`) and don't expose port `5432` publicly.
- [ ] Set `SEED_ADMIN_EMAIL`/`SEED_ADMIN_PASSWORD` to something other than the
      `admin`/`admin` default **before the first boot** — that default is
      created automatically if no users exist yet, and it's well-known. If you
      already booted with the default, log in and call
      `POST /auth/change-password` immediately.
- [ ] Register your first user via `POST /auth/register` immediately after
      first boot — it becomes admin automatically, and only the first one does.
- [ ] Issue a dedicated `AGENT_API_KEY` per `sdr-agent` (see
      [README](../README.md#sdr-agent-authentication)) instead of reusing a
      human's `AGENT_TOKEN`.
- [ ] Review [SECURITY_ROADMAP.md](../SECURITY_ROADMAP.md) for what's shipped
      and what's still planned — some hardening (e.g. audit logging) isn't in
      yet as of this writing.

## Option A: single host with Docker Compose

The simplest path — one VM or server running the same `docker-compose.yml`
used in development, with production values.

1. Provision a host (any VPS with Docker installed works).
2. Copy the repo (or just `docker-compose.yml`, `deploy/`, and `apps/`) to it.
3. Create a `.env` file next to `docker-compose.yml` (Compose loads it
   automatically) with production secrets:
   ```
   JWT_SECRET=<output of: openssl rand -hex 32>
   ```
4. Generate or obtain real TLS certificates (see [TLS](#tls) below) rather
   than the self-signed dev script.
5. `docker compose up -d --build`
6. Point DNS at the host and confirm `https://your-domain/healthz` responds.

This is a fine starting point for a single-team or evaluation deployment. It
doesn't give you zero-downtime deploys or horizontal scaling — for that, see
Option B.

## Option B: Kubernetes

Lyrahcle's monorepo already separates `deploy/k8s/` and `deploy/helm/` for
this (see [PLAN.md](../PLAN.md)), but manifests/charts aren't populated yet —
this is tracked as future work. Until then, the Dockerfiles under
`deploy/docker/` are what you'd wrap in your own Deployment/Service manifests:
`backend` needs `DATABASE_URL` pointed at a managed or in-cluster Postgres and
a `Secret` for `JWT_SECRET`; `frontend` is a stateless nginx container that
needs the backend's internal service address for its `/api` proxy (edit
`nginx.frontend.conf` or mount an override, matching what Compose does with
volumes today).

## TLS

Three ways to get HTTPS, roughly in order of how much most production setups
should prefer them:

1. **Reverse proxy with automatic certificates** — put
   [Caddy](https://caddyserver.com/) or [Traefik](https://traefik.io/) in
   front of the `frontend` and `backend` containers. Both can obtain and renew
   Let's Encrypt certificates automatically; you only manage the proxy config,
   not certificate files. This is the recommended default for most
   deployments.
2. **Certificates from your existing CA/ACME setup** — if your organization
   already issues certificates another way, set `TLS_CERT_FILE`/
   `TLS_KEY_FILE` on the backend directly and swap in
   `nginx.frontend.tls.conf` for the frontend (see the commented-out example
   in `docker-compose.yml`).
3. **Self-signed, local/dev only** — `scripts/generate-dev-certs.sh`. Never
   use this for anything reachable outside your own machine; browsers and
   `sdr-agent` (unless `AGENT_INSECURE_SKIP_VERIFY=true`, itself dev-only)
   will correctly refuse to trust it.

## Database

- Migrations run automatically on backend startup (`db.Migrate`, applying
  everything under `apps/backend/migrations/` in order). There's no separate
  migration step to remember.
- Back up with standard `pg_dump`/`pg_restore`; nothing Lyrahcle-specific is
  required. There's no built-in backup scheduling yet.
- The `pgdata` named volume in `docker-compose.yml` is where data lives for
  the single-host setup — make sure your host's volume backups actually
  include it.

## Monitoring the deployment

- `GET /healthz` — liveness check, no auth required. Point your
  orchestrator's health check or an uptime monitor at it.
- `GET /api/v1/ping` — confirms the API layer (not just the process) is
  responding.
- Backend logs (`docker compose logs backend` or your platform's log
  aggregation) include startup warnings for every insecure default that's
  still active (`JWT_SECRET`, TLS) — treat those as an actionable checklist,
  not noise.

## Rolling out an update

```bash
git pull
docker compose build
docker compose up -d
```

Migrations apply automatically on the new backend container's startup, before
it starts serving traffic. There is currently no automated rollback path for a
migration — see [SECURITY_ROADMAP.md](../SECURITY_ROADMAP.md) item 10
(versioned releases) for where this is headed.
