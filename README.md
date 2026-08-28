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

## Documentation map

This README covers the overview, quick start, and API reference. For anything
deeper:

| Guide | Covers |
|---|---|
| [docs/deployment.md](./docs/deployment.md) | Taking Lyrahcle to production: checklist, Docker Compose vs Kubernetes, TLS options, backups, monitoring, rollout |
| [docs/configuration.md](./docs/configuration.md) | Every environment variable, across every service, in one table |
| [CONTRIBUTING.md](./CONTRIBUTING.md) | Local dev setup, coding conventions, testing, PR workflow |
| [SECURITY.md](./SECURITY.md) | Vulnerability disclosure policy |
| [SECURITY_ROADMAP.md](./SECURITY_ROADMAP.md) | Short-term hardening plan and what's already shipped |
| [PLAN.md](./PLAN.md) | Full original development plan and long-term roadmap |

## Status

🚧 Early development — Phase 7 of 9 (TM/TC protocol decoding). See [PLAN.md](./PLAN.md) for
the full roadmap.

- [x] Phase 1 — Branding & repo bootstrap
- [x] Phase 2 — Backend (Go) and frontend (React) base
- [x] Phase 3 — Postgres persistence, migrations, scheduler
- [x] Phase 4 — Branded dashboard
- [x] Phase 5 — SDR edge agent, RF ingest
- [x] Phase 6 — Orbital tracking (SGP4, live Celestrak TLE sync, interactive world map)
- [ ] Phase 7 — TM/TC protocol decoding, ground station & firmware modules (AX.25, CCSDS Space Packet, and PUS decoders shipped; space-CVE threat-intel feed shipped; CubeSat Space Protocol and anomaly detection still pending)
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

The entire app requires a login — both the dashboard (frontend) and every read
or write API endpoint except `/healthz`, `/ping`, `/auth/register`, and
`/auth/login`. A default admin account is seeded automatically on first boot
if no users exist yet:

```
email:    admin   (override via SEED_ADMIN_EMAIL)
password: admin   (override via SEED_ADMIN_PASSWORD)
```

**This is a well-known default and must be changed immediately** — the
backend prints a loud warning banner in its logs on the boot that creates it.
Change it right after your first login:

```bash
curl -X POST localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin", "password": "admin"}'
# -> {"token": "...", "role": "admin"}

curl -X POST localhost:8080/api/v1/auth/change-password \
  -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d '{"current_password": "admin", "new_password": "a-much-stronger-password"}'
```

Additional accounts:

```bash
# First registration ever becomes admin automatically; later ones default to viewer.
curl -X POST localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "you@example.com", "password": "a-strong-password"}'

curl -X POST localhost:8080/api/v1/satellites/ \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"norad_id": 25544, "name": "ISS (ZARYA)", "tle_line1": "...", "tle_line2": "..."}'
```

Roles: `viewer` < `operator` < `admin`. Promoting a user beyond the bootstrap admin
currently requires updating `role` directly in the `users` table (a proper admin
endpoint is planned — see [SECURITY_ROADMAP.md](./SECURITY_ROADMAP.md)).

### SDR agent authentication

`POST /rf/ingest` accepts two kinds of credentials:

- **Agent API key** (recommended) — scoped to a single `agent_id`, long-lived,
  independently revocable, and cannot be used to submit readings under a different
  `agent_id` (the backend rejects any mismatch between the key's registered
  `agent_id` and the one in the request body).
- **Operator/admin bearer token** (legacy/fallback) — the shared human JWT from
  `/auth/login`, kept for backward compatibility. Prefer the agent key below for
  anything long-running.

```bash
# Create a key for an agent (admin token required):
curl -X POST localhost:8080/api/v1/agent-keys \
  -H "Authorization: Bearer <admin-token>" \
  -H "Content-Type: application/json" \
  -d '{"agent_id": "sdr-agent-madrid"}'
# -> {"agent_id": "sdr-agent-madrid", "api_key": "lyr_agent_..."}  (shown once — store it now)

# Revoke it later:
curl -X DELETE localhost:8080/api/v1/agent-keys/sdr-agent-madrid \
  -H "Authorization: Bearer <admin-token>"
```

Set the returned key as `AGENT_API_KEY` for that `sdr-agent` instance
(`docker-compose.yml`); it's sent as an `X-Agent-Key` header, separately from the
`Authorization: Bearer` header used by human sessions. Only the key's SHA-256 hash
is stored — the plaintext key is shown exactly once, at creation time.

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
| `SEED_ADMIN_EMAIL` / `SEED_ADMIN_PASSWORD` | `admin` / `admin` | Bootstrap admin account, created only if no users exist yet. Change immediately after first login |
| `SPACE_CVE_KEYWORDS` | `CCSDS,spacecraft,satellite ground station,SATCOM,CubeSat,space segment,satellite communication,orbital` | Comma-separated NVD keyword searches synced daily into the space-CVE feed |

### API endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/healthz` | — | Liveness check |
| `GET` | `/api/v1/ping` | — | Backend reachability check |
| `GET` | `/api/v1/assets` | viewer+ | List tracked assets |
| `POST` | `/api/v1/auth/register` | — | Create a user (first user ever becomes admin) |
| `POST` | `/api/v1/auth/login` | — | Exchange email + password for a bearer token |
| `POST` | `/api/v1/auth/change-password` | viewer+ | Change your own password |
| `GET` | `/api/v1/audit-log` | admin | List the 200 most recent security-relevant events |
| `POST` | `/api/v1/agent-keys` | admin | Create an API key scoped to one `agent_id` |
| `DELETE` | `/api/v1/agent-keys/{agentID}` | admin | Revoke that agent's key |
| `POST` | `/api/v1/rf/ingest` | agent key or operator+ | Ingest an RF reading (used by `sdr-agent`) |
| `GET` | `/api/v1/rf/readings` | viewer+ | List recent RF readings |
| `GET` | `/api/v1/satellites/` | viewer+ | List tracked satellites with live computed position |
| `POST` | `/api/v1/satellites/` | operator+ | Register a satellite by NORAD ID + TLE |
| `POST` | `/api/v1/tmtc/decode/ax25` | — | Decode a hex-encoded AX.25 UI frame (destination/source callsign+SSID, control, PID, payload) |
| `POST` | `/api/v1/tmtc/decode/ccsds` | — | Decode a hex-encoded CCSDS Space Packet (version, TM/TC type, secondary header flag, APID, sequence flags/count, payload) |
| `POST` | `/api/v1/tmtc/decode/pus` | — | Decode a hex-encoded PUS packet (CCSDS header fields + PUS version, service type/subtype, message counter, destination ID, payload) |
| `GET` | `/api/v1/threat-intel/space-cves` | viewer+ | List space-related CVEs synced from the NVD (auto-updates daily) |

## Testing

```bash
cd apps/backend
go test ./...                 # unit tests
go vet ./...

# Integration tests need a real Postgres; they skip cleanly without it.
export TEST_DATABASE_URL="postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle_test?sslmode=disable"
go test ./internal/store/... -v

# Fuzz the AX.25 decoder (run manually; not part of the default CI run).
go test ./internal/tmtc -fuzz=FuzzDecodeAX25Hex -fuzztime=30s
```

```bash
cd apps/frontend
npm test
```

CI runs backend build+vet+unit+integration tests (with a Postgres service
container) and frontend build+tests on every push and pull request.

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
