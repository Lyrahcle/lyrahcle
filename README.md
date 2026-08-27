<p align="center">
  <img src="./assets/logo-mark.svg" alt="Lyrahcle" width="140" />
</p>

<h1 align="center">Lyrahcle — Constellation Analytics</h1>

<p align="center"><b>Never Lies.</b></p>

<p align="center">
  Lyrahcle is a professional space-cybersecurity platform for assessing, monitoring, and
  defending space infrastructure: satellites, ground stations, RF links, TM/TC protocols,
  embedded firmware, and orbital telemetry.
</p>

<p align="center">
  Lyrahcle follows an <b>Open Core</b> model — an open, auditable core (AGPLv3) combined with
  paid enterprise modules — aiming for transparency, community, and sustainability.
</p>

<p align="center">
  Maintained by <a href="https://github.com/mlko01">mlko01</a> — contact: manu.ciberseguridad@gmail.com
</p>

<p align="center">
  <img src="./docs/screenshots/dashboard.png" alt="Lyrahcle dashboard" width="820" />
  <br />
  <sub><i>Orbital tracking tab with example data.</i></sub>
</p>

## Status

🚧 Early development — Phase 7 of 9 (TM/TC protocol decoding). See [PLAN.md](./PLAN.md) for
the full roadmap.

- [x] Phase 1 — Branding & repo bootstrap
- [x] Phase 2 — Backend (Go) and frontend (React) base
- [x] Phase 3 — Postgres persistence, migrations, scheduler
- [x] Phase 4 — Branded dashboard
- [x] Phase 5 — SDR edge agent, RF ingest
- [x] Phase 6 — Orbital tracking (SGP4, live Celestrak TLE sync, interactive world map)
- [ ] Phase 7 — TM/TC protocol decoding, ground station & firmware modules (started: AX.25 decoder)
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

### Authentication

Write endpoints (`POST /rf/ingest`, `POST /satellites/`) require a bearer token.
Read endpoints stay public for now (dashboard doesn't require login yet).

```bash
# First registration becomes admin automatically; later ones default to viewer.
curl -X POST localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "you@example.com", "password": "a-strong-password"}'

curl -X POST localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "you@example.com", "password": "a-strong-password"}'
# -> {"token": "...", "role": "admin"}

curl -X POST localhost:8080/api/v1/satellites/ \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"norad_id": 25544, "name": "ISS (ZARYA)", "tle_line1": "...", "tle_line2": "..."}'
```

Roles: `viewer` < `operator` < `admin`. Promoting a user beyond the bootstrap admin
currently requires updating `role` directly in the `users` table (a proper admin
endpoint is planned — see [SECURITY_ROADMAP.md](./SECURITY_ROADMAP.md)).

For the `sdr-agent` to publish RF readings, give it a token with at least the
`operator` role via the `AGENT_TOKEN` environment variable (`docker-compose.yml`).
Token-per-agent authentication (API keys, no shared user token) is tracked in the
security roadmap.

### TLS/HTTPS

Plain HTTP is the default for zero-friction local development (the backend logs a
warning on every startup while it is). To enable HTTPS end-to-end:

```bash
./scripts/generate-dev-certs.sh
```

This creates a self-signed certificate in `./certs` (gitignored). Then in
`docker-compose.yml`:

- Uncomment the `volumes` and `TLS_CERT_FILE`/`TLS_KEY_FILE` lines under `backend`
  (and change its port mapping to `8443:8443` with `PORT=8443` if you want the
  backend directly reachable over HTTPS).
- Uncomment the `volumes` lines under `frontend` to swap in
  `deploy/docker/nginx.frontend.tls.conf`, which serves HTTPS on 443 and redirects
  plain HTTP to it.
- If `sdr-agent` talks to an HTTPS backend using this self-signed cert, set
  `AGENT_INSECURE_SKIP_VERIFY=true` for that agent — **local development only**,
  never against a production backend.

In production, prefer certificates from a trusted CA (Let's Encrypt via
[Caddy](https://caddyserver.com/) or [Traefik](https://traefik.io/) in front of
Lyrahcle) over the self-signed dev certificate, and terminate TLS at that proxy or
by setting `TLS_CERT_FILE`/`TLS_KEY_FILE` directly on the backend.

### Environment variables (backend)

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable` | Postgres connection string |
| `REDIS_URL` | `localhost:6379` | Redis connection string |
| `PORT` | `8080` | HTTP listen port |
| `CELESTRAK_GROUPS` | `stations` | Comma-separated [Celestrak groups](https://celestrak.org/NORAD/elements/) to sync TLEs from |
| `JWT_SECRET` | *(insecure dev default, logs a warning)* | Secret used to sign auth tokens — set a strong value in any non-local deployment |
| `TLS_CERT_FILE` / `TLS_KEY_FILE` | *(unset, plain HTTP, logs a warning)* | Paths to a TLS certificate/key to serve HTTPS directly from the backend |

### API endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/healthz` | — | Liveness check |
| `GET` | `/api/v1/ping` | — | Backend reachability check |
| `GET` | `/api/v1/assets` | — | List tracked assets |
| `POST` | `/api/v1/auth/register` | — | Create a user (first user becomes admin) |
| `POST` | `/api/v1/auth/login` | — | Exchange email + password for a bearer token |
| `POST` | `/api/v1/rf/ingest` | operator+ | Ingest an RF reading (used by `sdr-agent`) |
| `GET` | `/api/v1/rf/readings` | — | List recent RF readings |
| `GET` | `/api/v1/satellites/` | — | List tracked satellites with live computed position |
| `POST` | `/api/v1/satellites/` | operator+ | Register a satellite by NORAD ID + TLE |
| `POST` | `/api/v1/tmtc/decode/ax25` | — | Decode a hex-encoded AX.25 UI frame (destination/source callsign+SSID, control, PID, payload) |

## Security

See [SECURITY_ROADMAP.md](./SECURITY_ROADMAP.md) for the short-term hardening plan
(TLS, input validation, test coverage, rate limiting, agent-specific auth, audit
logging, and more) being worked through incrementally before v1.0.0.

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
