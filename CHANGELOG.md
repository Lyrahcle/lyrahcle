# Changelog

All notable changes to this project will be documented in this file.
Format based on [Keep a Changelog](https://keepachangelog.com/), using
[Conventional Commits](https://www.conventionalcommits.org/).

## [Unreleased]
### Added
- Initial monorepo scaffold (apps/, enterprise/, packages/, deploy/, scripts/, docs/)
- Project branding: name, logo, slogan "Lyrahcle – Never Lies"
- Backend base (Go + chi): `/healthz` and `/api/v1/ping` endpoints, config loader
- Frontend base (React + Vite + Tailwind): landing page pinging backend
- Dockerfiles for backend, frontend, cve-sync
- Vite dev proxy `/api` → backend
- Postgres schema (programs, assets, scans, findings) + minimal SQL migration runner
- Asset store (upsert/list) and `GET /api/v1/assets`
- Cron-based scheduler (robfig/cron) wired for future recon sweeps
- Branded dashboard (Header, StatCard, AssetsTable) using logo and purple/gold palette
- SDR edge agent (`apps/sdr-agent`) with mock RF capture and HTTP uplink to backend
- `rf_readings` table, `POST /api/v1/rf/ingest`, `GET /api/v1/rf/readings`
- Dashboard RF spectrum panel (RfPanel)
- `sdr-agent` compose service under the `sdr` profile (`docker compose --profile sdr up`)
- Dashboard redesign: mission-console look (deep purple/gold palette, Space Grotesk/Inter/JetBrains Mono type system, starfield background, live UTC clock, signal-strength meters, kind badges)
- Custom vector logo (LogoMark, SVG constellation glyph) replacing the raster placeholder
- Orbital tracking: `satellites` table, SGP4 propagation via go-satellite (`internal/orbit`), `GET/POST /api/v1/satellites`, dashboard satellites panel
- Dashboard v2: hero section with animated orbit rings and live subsystem status, tab navigation (Orbital/Assets/Signal), dotted world map with live satellite ground-track pins, 8s auto-refresh polling, loading skeletons
- Automatic TLE sync from Celestrak (`internal/celestrak`), runs on startup and every 6h via scheduler, configurable via `CELESTRAK_GROUPS`
- Interactive world map: hover tooltip per satellite (name, NORAD ID, lat/lon, altitude), pulsing marker
- Expanded README: architecture diagram, env vars, API reference, roadmap checklist
- Responsive/accessibility polish: horizontal scroll for data tables on narrow screens (no more cramped/wrapped columns), visible focus rings on tabs, `motion-safe` guards on all pulse/spin animations, tighter mobile spacing, meta description and theme-color
- Dashboard screenshot added to README
- Palette change: neutral graphite/black background (previously deep purple) with gold as the sole accent; centered README header, logo, and screenshot
- Fixed world map marker stacking: hovered satellite tooltip now renders above other nearby markers instead of being covered
- Scaled up UI: larger hero, logo, tabs, table text and padding across the dashboard
- Full footer: brand blurb, project/contact links, license and roadmap-phase line
- Open Graph/Twitter meta tags, apple-touch-icon, and a keyboard-accessible skip-to-content link
- Fixed clock placement: moved from the hero status row to a sticky header (always visible, no longer misaligned with status pills)
- Phase 7 started: AX.25 frame decoder (`internal/tmtc`) and `POST /api/v1/tmtc/decode/ax25` endpoint — decodes destination/source callsign+SSID, control, PID, and payload from a hex-encoded frame
- README screenshot upgraded to a browser-window mockup at 2x resolution
- Authentication & authorization: `users` table, JWT (HS256) via `golang-jwt`, bcrypt password hashing, `admin`/`operator`/`viewer` role hierarchy, `POST /auth/register` + `POST /auth/login`; `POST /rf/ingest` and `POST /satellites/` now require an operator-or-higher bearer token; `sdr-agent` sends its token via `AGENT_TOKEN`
- Added `SECURITY_ROADMAP.md` tracking the short-term hardening plan (TLS, input validation, tests, rate limiting, agent API keys, audit logging, releases) incrementally before v1.0.0
- Strict input validation (`internal/validate`) across `/auth/register`, `/auth/login`, `/rf/ingest`, `POST /satellites/`, and `/tmtc/decode/ax25`: bounded frequency/power/timestamp ranges, NORAD ID and TLE line format checks, AX.25 hex length cap, hardened email regex, 1MB global request body limit
- Rate limiting (`go-chi/httprate`): 120 req/min per IP globally, 10 req/min per IP on `/auth/register` and `/auth/login`
- Clock now shows the browser's local time and UTC offset instead of a fixed UTC
- Opt-in TLS/HTTPS: backend can serve HTTPS directly (`TLS_CERT_FILE`/`TLS_KEY_FILE`), nginx gained an HTTPS+redirect config, `sdr-agent` supports `https://` backends with a dev-only cert-skip-verify flag, and `scripts/generate-dev-certs.sh` generates a local self-signed cert; plain HTTP remains the default with a startup warning
- Automated tests: backend unit tests (`auth`, `validate`, `tmtc` at 100% coverage, `celestrak`), AX.25 fuzz tests (~1M executions, zero panics), `store` integration tests against real Postgres, frontend Vitest/Testing Library tests (`SignalBar`, `KindBadge`, `lib/api.js`); CI now runs build+test+vet for both apps with a Postgres service container instead of the previous placeholder steps
- SDR agent authentication: `agent_keys` table with SHA-256-hashed keys, `POST`/`DELETE /agent-keys` (admin-only), `X-Agent-Key` header support with fallback to the existing operator JWT, server-side rejection of RF readings whose `agent_id` doesn't match the authenticated key (blocks agent spoofing); `sdr-agent` sends `AGENT_API_KEY` when set
