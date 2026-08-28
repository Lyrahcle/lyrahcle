# Security & Hardening Roadmap

Tracked incrementally before the v1.0.0 release. Details on each shipped item
are in [CHANGELOG.md](./CHANGELOG.md); this page tracks status only.

- [x] Authentication & authorization — JWT + bcrypt, `admin`/`operator`/`viewer`
      role hierarchy, admin-managed role changes
- [x] TLS/HTTPS across frontend, backend, and sdr-agent
- [x] Strict input validation on all endpoints
- [x] Automated test coverage — unit, integration, fuzzing, HTTP-level route
      tests, all run in CI against a real Postgres service container
- [x] Rate limiting on the public API, tighter limits on auth endpoints
- [x] SDR agent authentication via per-agent, revocable API keys
- [x] Security response headers on the backend API and frontend nginx
- [x] Security logging & audit trail, admin-only `/audit-log`
- [x] App-wide login gate — no public reads
- [x] Versioned releases published to GHCR — see [RELEASING.md](./RELEASING.md)
- [x] Dependency vulnerability scanning — `govulncheck`, `npm audit`, Dependabot
- [x] Container image vulnerability scanning (Trivy) in the release workflow
- [x] Graceful shutdown on `SIGINT`/`SIGTERM`
- [x] Versioned, idempotent database migrations
- [x] Frontend crash isolation via a React error boundary
- [x] Backend observability — Prometheus metrics at `/metrics`
- [x] Automated database backups (`scripts/backup-db.sh`)
- [x] API pagination on list endpoints
- [x] IQ capture & forensic playback — record raw IQ to disk, replay a saved capture as historical RF readings
- [ ] Real SDR hardware integration (RTL-SDR, HackRF) to validate capture
      beyond the mock source
