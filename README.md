<p align="center">
  <img src="./assets/logo-mark.svg" alt="Lyrahcle" width="140" />
</p>

<h1 align="center">Lyrahcle — Constellation Analytics</h1>

<p align="center"><b>Never Lies.</b></p>

<p align="center">
  A professional space-cybersecurity platform for assessing, monitoring, and defending
  space infrastructure: satellites, ground stations, RF links, TM/TC protocols, embedded
  firmware, and orbital telemetry.
</p>

<p align="center">
  <b>Open Core</b> — an open, auditable core (AGPLv3) plus paid enterprise modules.
</p>

<p align="center">
  Maintained by <a href="https://github.com/mlko01">mlko01</a> — manu.ciberseguridad@gmail.com
</p>

<p align="center">
  <img src="./docs/screenshots/dashboard.png" alt="Lyrahcle dashboard" width="820" />
  <br />
  <sub><i>Full dashboard view — header, status strip, orbital tracking, and footer (mock data).</i></sub>
</p>

## Documentation

| Guide | Covers |
|---|---|
| [docs/api-reference.md](./docs/api-reference.md) | Every endpoint, authentication, agent keys |
| [docs/deployment.md](./docs/deployment.md) | Production checklist, Docker Compose vs Kubernetes, TLS, backups |
| [docs/configuration.md](./docs/configuration.md) | Every environment variable, across every service |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | Local dev setup, coding conventions, testing, PR workflow |
| [SECURITY.md](./SECURITY.md) · [SECURITY_ROADMAP.md](./SECURITY_ROADMAP.md) | Vulnerability disclosure · hardening plan & status |
| [RELEASING.md](./RELEASING.md) | Versioning policy, cutting a release |
| [PLAN.md](./PLAN.md) | Full development plan and long-term roadmap |

## Status

🚧 Phase 8 of 9 — test coverage, hardening, v1.0.0 release. Phases 1–7 shipped
(branding, backend/frontend base, Postgres persistence, branded dashboard, SDR
edge agent, SGP4 orbital tracking, TM/TC decoding + RF anomaly detection). See
[PLAN.md](./PLAN.md) for the full roadmap.

## Features

- **Asset inventory** — domains, IPs, satellites, ground stations, firmware, by program.
- **Orbital tracking** — TLEs auto-synced from [Celestrak](https://celestrak.org); live
  lat/lon/altitude via SGP4, plotted on an interactive world map.
- **RF signal intelligence** — edge `sdr-agent` streams spectrum readings to the backend.
- **RF anomaly detection** — every reading is checked against that agent's baseline
  (z-score, 3σ threshold); outliers are flagged automatically.
- **TM/TC protocol decoding** — paste a hex frame and get parsed fields for AX.25, CCSDS
  Space Packet, PUS, or CubeSat Space Protocol (CSP v1).
- **Mission-console dashboard** — React + Tailwind, dark graphite/gold identity, live
  subsystem status, 8s auto-refresh.

## Architecture

```
┌─────────────┐      HTTP        ┌─────────────┐      SQL      ┌────────────┐
│  frontend    │ ───────────────▶ │   backend    │ ─────────────▶ │  postgres   │
│  React/Vite  │ ◀─────────────── │  Go / chi    │ ◀───────────── │             │
└─────────────┘                  └──────┬───────┘                └──────┬─────┘
                                          │  ▲                            ▲
                        POST /rf/ingest   │  │  TLE sync (cron)           │ writes space_cves
                                          ▼  │                            │
                                   ┌─────────────┐        ┌───────────────┐   ┌──────────┐
                                   │  sdr-agent   │        │   Celestrak    │   │ cve-sync  │
                                   │  (edge, Go)  │        │   (external)   │   │ (worker)  │
                                   └─────────────┘        └───────────────┘   └────┬─────┘
                                                                                     ▼
                                                                              NVD API + Redis
```

`apps/cve-sync` is the **only** writer of `space_cves` (Redis dedup cache,
Prometheus metrics); the backend only reads it. See
[CHANGELOG.md](./CHANGELOG.md) for details.

## Repository layout

```
lyrahcle/
├── apps/
│   ├── backend/       # Go API: chi router, Postgres store, SGP4 orbit, Celestrak sync, scheduler
│   ├── frontend/       # React + Vite + Tailwind dashboard
│   ├── sdr-agent/      # Edge RF capture agent (mock capture, real SDR pending hardware)
│   └── cve-sync/       # NVD sync worker — source of truth for the space-CVE feed
├── enterprise/         # Placeholder for proprietary modules — not implemented yet
├── packages/           # Placeholder for shared MIT-licensed libraries — not implemented yet
├── deploy/             # Dockerfiles, Helm charts, Kubernetes manifests
├── scripts/            # Dev / CI utilities
└── docs/               # Technical & user documentation
```

## Getting started

Requires Docker and Docker Compose.

```bash
docker compose up --build
```

- Frontend: [http://localhost:5173](http://localhost:5173)
- Backend health check: `curl localhost:8080/healthz`
- Default login: `admin` / `admin` — **change it immediately**, see
  [docs/api-reference.md](./docs/api-reference.md#authentication)

To also run the SDR edge agent (mock capture, no hardware required):

```bash
docker compose --profile sdr up --build
```

For TLS, environment variables, agent keys, and the full endpoint list, see
[docs/api-reference.md](./docs/api-reference.md),
[docs/configuration.md](./docs/configuration.md), and
[docs/deployment.md](./docs/deployment.md).

## Testing

```bash
cd apps/backend
go test ./...                                    # unit tests
go vet ./...
export TEST_DATABASE_URL="postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle_test?sslmode=disable"
go test ./internal/store/... -v                  # needs real Postgres; skips cleanly without it
go test ./cmd/server/... -v                      # HTTP-level tests against the real router
go install golang.org/x/vuln/cmd/govulncheck@latest && govulncheck ./...
```

```bash
cd apps/frontend
npm test
npm audit --audit-level=high
```

```bash
./scripts/smoke-test.sh            # full docker-compose stack + health checks
./scripts/smoke-test.sh --with-sdr # also starts sdr-agent
```

CI runs all of the above (build+vet+unit+integration+HTTP tests, vuln
scanning, frontend build+tests+audit, and the full-stack smoke test) on every
push and PR. Dependabot keeps dependencies current across Go/npm/Docker/Actions.

## Security

Every response carries a locked-down set of security headers
(`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`,
`Permissions-Policy`, `Content-Security-Policy`, `Strict-Transport-Security`
with TLS), on top of JWT auth, bcrypt hashing, per-agent API keys, rate
limiting, strict input validation, and audit logging. See
[SECURITY_ROADMAP.md](./SECURITY_ROADMAP.md) for the full hardening plan and
what's already shipped.

## Licensing

| Component | License |
|---|---|
| Core (backend, frontend, API, basic scanning/monitoring/SDR) | AGPLv3 |
| Enterprise modules (advanced RF, proprietary decoders, SIEM integrations) | Commercial |
| Shared libraries (`rf-lib`, `tle-lib`, `ui-components`) | MIT |
| Documentation | CC BY-SA 4.0 |

See [PLAN.md](./PLAN.md) for the development plan and [SECURITY.md](./SECURITY.md)
for the vulnerability disclosure policy. Enterprise modules require a license key
(not yet implemented).
