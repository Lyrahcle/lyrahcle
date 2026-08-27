# Lyrahcle — Constellation Analytics

**Never Lies.**

Lyrahcle is a professional space-cybersecurity platform for assessing, monitoring, and
defending space infrastructure: satellites, ground stations, RF links, TM/TC protocols,
embedded firmware, and orbital telemetry.

Lyrahcle follows an **Open Core** model — an open, auditable core (AGPLv3) combined with
paid enterprise modules — aiming for transparency, community, and sustainability.

<img src="./assets/logo-mark.svg" alt="Lyrahcle" width="140" />

Maintained by [mlko01](https://github.com/mlko01) — contact: manu.ciberseguridad@gmail.com

## Status

🚧 Early development — Phase 6 of 9 (orbital tracking). See [PLAN.md](./PLAN.md) for the
full roadmap.

- [x] Phase 1 — Branding & repo bootstrap
- [x] Phase 2 — Backend (Go) and frontend (React) base
- [x] Phase 3 — Postgres persistence, migrations, scheduler
- [x] Phase 4 — Branded dashboard
- [x] Phase 5 — SDR edge agent, RF ingest
- [x] Phase 6 — Orbital tracking (SGP4, live Celestrak TLE sync, interactive world map)
- [ ] Phase 7 — TM/TC protocol decoding, ground station & firmware modules
- [ ] Phase 8 — Test coverage, hardening, v1.0.0 release
- [ ] Phase 9 — Enterprise modules (advanced RF, proprietary decoders, SIEM integrations)

## Features

- **Asset inventory** — tracked domains, IPs, satellites, ground stations, and firmware,
  grouped by program.
- **Orbital tracking** — TLEs synced automatically from [Celestrak](https://celestrak.org)
  on startup and every 6h; live position (lat/lon/altitude) computed via SGP4 propagation
  and plotted on an interactive world map with hover detail.
- **RF signal intelligence** — an edge `sdr-agent` streams spectrum readings to the
  backend; the dashboard shows live signal-strength meters per reading.
- **Mission-console dashboard** — React + Tailwind, dark purple/gold identity, tab
  navigation, live subsystem status, 8s auto-refresh.

## Architecture

```
┌─────────────┐      HTTP        ┌─────────────┐      SQL      ┌────────────┐
│  frontend    │ ───────────────▶ │   backend    │ ─────────────▶ │  postgres   │
│  React/Vite  │ ◀─────────────── │  Go / chi    │ ◀───────────── │             │
└─────────────┘                  └──────┬───────┘                └────────────┘
                                          │  ▲
                        POST /rf/ingest   │  │  TLE sync (cron)
                                          ▼  │
                                   ┌─────────────┐        ┌───────────────┐
                                   │  sdr-agent   │        │   Celestrak    │
                                   │  (edge, Go)  │        │   (external)   │
                                   └─────────────┘        └───────────────┘
```

## Repository layout

```
lyrahcle/
├── apps/
│   ├── backend/       # Go API: chi router, Postgres store, SGP4 orbit, Celestrak sync, scheduler
│   ├── frontend/       # React + Vite + Tailwind dashboard
│   ├── sdr-agent/      # Edge RF capture agent (mock capture, real SDR pending hardware)
│   ├── cve-sync/       # NVD sync worker (placeholder)
│   └── docs/           # Generated API docs
├── enterprise/         # Proprietary modules (not published in the public repo)
├── packages/           # Shared MIT-licensed libraries (planned)
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

To also run the SDR edge agent (mock capture, no hardware required):

```bash
docker compose --profile sdr up --build
```

### Environment variables (backend)

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable` | Postgres connection string |
| `REDIS_URL` | `localhost:6379` | Redis connection string |
| `PORT` | `8080` | HTTP listen port |
| `CELESTRAK_GROUPS` | `stations` | Comma-separated [Celestrak groups](https://celestrak.org/NORAD/elements/) to sync TLEs from |

### API endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/healthz` | Liveness check |
| `GET` | `/api/v1/ping` | Backend reachability check |
| `GET` | `/api/v1/assets` | List tracked assets |
| `POST` | `/api/v1/rf/ingest` | Ingest an RF reading (used by `sdr-agent`) |
| `GET` | `/api/v1/rf/readings` | List recent RF readings |
| `GET` | `/api/v1/satellites/` | List tracked satellites with live computed position |
| `POST` | `/api/v1/satellites/` | Register a satellite by NORAD ID + TLE |

## Licensing

| Component | License |
|---|---|
| Core (backend, frontend, API, basic scanning/monitoring/SDR) | AGPLv3 |
| Enterprise modules (advanced RF, proprietary protocol decoding, SIEM integrations, executive reporting) | Commercial |
| Shared libraries (`rf-lib`, `tle-lib`, `ui-components`) | MIT |
| Documentation | CC BY-SA 4.0 |

See [PLAN.md](./PLAN.md) for the full development plan and [SECURITY.md](./SECURITY.md)
for the vulnerability disclosure policy.

(Enterprise modules: `docker compose -f docker-compose.enterprise.yml up`, requires a license key — not yet implemented.)
