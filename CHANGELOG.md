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
- Header search bar (UI, not yet wired to a backend search endpoint)
- OpenAPI 3.0 spec covering all 22 endpoints, embedded in the backend binary and served at `/openapi.yaml`, with a browsable Swagger UI at `/docs`
- `sdr-agent` IQ forensics: periodic raw IQ capture to disk (`IQ_CAPTURE_DIR`) and one-shot replay of a saved capture as historical RF readings (`REPLAY_FILE`), reconstructing the original timeline so replayed anomalies are flagged the same way a live signal would be — see [docs/forensics.md](./docs/forensics.md)
- Anomaly detection upgraded from a plain mean/stddev z-score to two complementary statistical baselines: a robust modified z-score (median + MAD) for spot outliers, resistant to a prior anomaly already in the window skewing the baseline; and an EWMA control-chart trend baseline that catches a sustained drift too gradual to trip a single-point check
- Dashboard light/dark theme (all colors now CSS custom properties, toggleable and following the system preference by default) and a bilingual English/Spanish interface, both configurable from a new Settings tab alongside account/password management
- Users & roles panel gained a search-by-email filter and a match count
- Accessibility pass across the dashboard: fixed unassociated form labels (3 forms), added `scope="col"` to every data table, `role="alert"` on form errors, a proper focus-trapped `role="dialog"` for the change-password modal (Tab cycling, Escape-to-close, focus restored on close), keyboard-focusable world-map markers, live-region status/loading announcements, and a dynamic `<html lang>`. Also found and fixed two WCAG AA contrast failures in the new light/dark theme's dimmest text color (was 2.8–3.6:1, now 4.7–5.0:1 in both themes). 12 new tests, including an automated axe-core audit
- Redis integration, all optional and gracefully degrading if Redis isn't reachable: rate limiting is now backed by Redis (survives restarts, correct across multiple replicas, replacing the previous in-memory counter); a new `POST /auth/logout` and password changes both revoke the JWT used to make the request immediately instead of leaving it valid until its natural expiry; `GET /satellites/` and `GET /threat-intel/space-cves` are now response-cached (5s and 30s TTL respectively)
- CVE-to-asset alerting: `cve-sync` now publishes each newly synced CVE to a Redis queue; a background worker in the backend matches it against tracked assets by kind (satellite/ground_station/firmware CVE keywords → matching asset kinds) and raises an alert an operator can acknowledge — `GET /asset-alerts/`, `POST /asset-alerts/{id}/acknowledge`, and a new "Asset Alerts" panel in the dashboard's Intel tab

### Changed
- `apps/cve-sync` is now the sole writer of `space_cves`; the backend only reads it, removing a duplicate in-process sync that raced on the same table
- Dashboard redesign: slim status strip, tab navigation, edge-to-edge world map, redesigned Header (role-colored account menu) and Footer (data-driven links)
- README trimmed to overview + quick start + documentation links; full API/auth walkthrough moved to `docs/api-reference.md`

### Fixed
- `clientIP()` now reads `X-Real-IP`/`X-Forwarded-For` instead of the reverse proxy's own address
- Docker Compose startup race: `postgres`/`redis` now have real healthchecks, dependent services wait on `service_healthy`, and every long-running service restarts automatically
- Test isolation bug where `audit_log`/`space_cves` weren't reset between test runs
- Frontend pagination not refetching immediately on Next/Prev
- `cve-sync` had never successfully written a single CVE to the real database: `UpsertCVEs` referenced columns (`id`, `has_score`, `references`) that don't exist in the migrated `space_cves` schema (`cve_id`, no `has_score`, `references_urls`) — `references` is also a reserved SQL keyword, so every write failed with a syntax error. The sync loop only logs the error and continues, so it looked healthy in logs the whole time. Fixed and covered by regression tests against the real schema

### Security
- Backend and frontend security response headers (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`, `Permissions-Policy`, `Content-Security-Policy`, `Strict-Transport-Security`)
- Dependency and container image vulnerability scanning (`govulncheck`, `npm audit`, Trivy, Dependabot) in CI, fixing a SQL injection in the Postgres driver and other real findings on first run
- Graceful shutdown on `SIGINT`/`SIGTERM`, draining in-flight requests
- Versioned, idempotent database migrations (`schema_migrations` tracking table)
- Production safety check: the backend refuses to start with insecure defaults when `ENV=production`
- The four `POST /tmtc/decode/*` endpoints didn't require authentication, contradicting the documented "every endpoint except healthz/ping/register/login requires login" policy — the frontend already sent its auth token to them regardless, so this closes the gap with no frontend change needed. Covered by a new regression test
