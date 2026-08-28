# Changelog

All notable changes to this project are documented in this file.
Format based on [Keep a Changelog](https://keepachangelog.com/), using
[Conventional Commits](https://www.conventionalcommits.org/).

## [Unreleased]

### Added
- Monorepo scaffold (`apps/`, `enterprise/`, `packages/`, `deploy/`, `scripts/`, `docs/`) and project branding
- Backend (Go + chi): REST API, Postgres persistence, cron-based scheduler
- Frontend (React + Vite + Tailwind): mission-console dashboard, dark graphite/gold identity
- Orbital tracking: SGP4 propagation (`go-satellite`), automatic TLE sync from Celestrak, interactive world map with live satellite ground-track pins
- RF signal intelligence: edge `sdr-agent` (mock capture), spectrum readings pipeline, per-agent z-score anomaly detection
- TM/TC protocol decoding: AX.25, CCSDS Space Packet, PUS, and CubeSat Space Protocol (CSP v1), fuzz-tested
- Space-related threat intelligence: NVD-sourced CVE feed via `apps/cve-sync` (sole writer of `space_cves`, Redis dedup cache, Prometheus metrics)
- Authentication & authorization: JWT (HS256) + bcrypt, `admin`/`operator`/`viewer` roles, per-agent API keys, admin-managed role changes
- Audit logging (`/audit-log`, admin-only) and an app-wide login gate
- Dashboard: asset inventory, satellite registration, protocol decoder, threat intel, and admin panels (agent keys, audit log, users & roles), all role-gated
- API pagination (`?limit=`/`?offset=`) on list endpoints
- Versioned releases: tagging `v*.*.*` builds and publishes all four Docker images to GHCR and cuts a GitHub Release — see [RELEASING.md](./RELEASING.md)
- Backend observability (`GET /metrics`, Prometheus) and automated database backups (`scripts/backup-db.sh`)
- React error boundary for frontend crash isolation

### Changed
- `apps/cve-sync` is now the sole writer of `space_cves`; the backend only reads it, removing a duplicate in-process sync that raced on the same table
- Dashboard redesign: slim status strip, tab navigation, edge-to-edge world map, redesigned Header (role-colored account menu) and Footer (data-driven links)
- README trimmed to overview + quick start + documentation links; full API/auth walkthrough moved to `docs/api-reference.md`

### Fixed
- `clientIP()` now reads `X-Real-IP`/`X-Forwarded-For` instead of the reverse proxy's own address
- Docker Compose startup race: `postgres`/`redis` now have real healthchecks, dependent services wait on `service_healthy`, and every long-running service restarts automatically
- Test isolation bug where `audit_log`/`space_cves` weren't reset between test runs
- Frontend pagination not refetching immediately on Next/Prev

### Security
- Backend and frontend security response headers (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy`, `Content-Security-Policy`, `Strict-Transport-Security`)
- Dependency and container image vulnerability scanning (`govulncheck`, `npm audit`, Trivy, Dependabot) in CI, fixing a SQL injection in the Postgres driver and other real findings on first run
- Graceful shutdown on `SIGINT`/`SIGTERM`, draining in-flight requests
- Versioned, idempotent database migrations (`schema_migrations` tracking table)
- Production safety check: the backend refuses to start with insecure defaults when `ENV=production`
