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
- Expanded documentation: `docs/deployment.md`, `docs/configuration.md`, `CONTRIBUTING.md`, and a documentation map in the README linking all guides
- Audit logging: `audit_log` table, `GET /audit-log` (admin-only), records auth events, agent-key lifecycle, and satellite upserts
- App-wide login gate: dashboard and all read endpoints now require authentication; seeded `admin`/`admin` bootstrap account (overridable, warns loudly); `POST /auth/change-password`; Login screen and session handling in the frontend
- CI fix: bumped frontend Node version to 22 (jsdom 30 needs undici APIs from Node 22+)
- CCSDS Space Packet Protocol decoder (`internal/tmtc/ccsds.go`) and `POST /api/v1/tmtc/decode/ccsds`: decodes version, TM/TC packet type, secondary header flag, APID, sequence flags/count, and payload from a hex-encoded packet; rejects packets whose actual length doesn't match the declared data length. Fuzz-tested (~1M executions across both entry points, zero panics), 98%+ package coverage
- PUS decoder (`internal/tmtc/pus.go`) and `POST /api/v1/tmtc/decode/pus`, built on the CCSDS header parser: service type/subtype, message counter, destination ID, payload. Fuzz-tested, zero panics
- Space-related threat intelligence: `space_cves` table, NVD keyword-search client (`internal/threatintel`), daily auto-sync across 8 space-related keywords, `GET /api/v1/threat-intel/space-cves` (viewer+)
- Initial TLE/CVE sync moved to background goroutines so backend startup no longer blocks on external API latency
- Frontend: Decoder tab (AX.25/CCSDS/PUS, paste hex → decoded fields) and Intel tab (space CVEs with severity badges, score, NVD link) — both capabilities now usable from the UI, not just curl
- Frontend redesign: removed the oversized centered hero section (replaced with a slim status strip), added a subtle grain texture, world map now bleeds to its panel's edge, header logo/name is now a clickable link back to the dashboard home
- Saved the user's full extended technical roadmap and business/differentiator goals to persistent memory
- Programs and Assets are now actually usable end-to-end: `POST`/`GET /api/v1/programs/` and `POST /api/v1/assets/` (previously `assets` had no creation path at all — the feature was a dead end)
- `GET /api/v1/agent-keys/` to list issued agent keys (previously create/revoke only, no listing)
- Frontend: "+ New asset" form (with inline program creation), "+ Register satellite" form, and a new Admin tab (agent key issue/revoke, live audit log) — all role-gated, verified viewers don't see them
- README screenshot updated to reflect the redesigned dashboard (slim status strip, 6 tabs including Admin, edge-to-edge map, action buttons)
- Auth UI completed: registration screen and a "change password" flow (header user menu) — previously `POST /auth/register` and `POST /auth/change-password` only existed as raw API calls
- **Architecture decision: `apps/cve-sync` is now the sole writer of `space_cves`.** Removed the backend's duplicate in-process sync (`internal/threatintel`, the `space-cve-sync` cron job, and the startup goroutine) — two independent workers were racing to upsert the same table. `apps/cve-sync` was kept because it has NVD sync incremental tracking, a Redis dedup cache, and Prometheus metrics; the backend now only reads the table via `GET /api/v1/threat-intel/space-cves`. `SPACE_CVE_KEYWORDS` is no longer a backend env var
- Removed `TIER1_IMPLEMENTATION.md` (superseded by `docker-compose.yml`, the migrations, and this changelog)
- Added `.env.example` documenting every environment variable across all services
- Production safety check: the backend now refuses to start (instead of only logging a warning) when `ENV=production` and `JWT_SECRET`, `SEED_ADMIN_PASSWORD`, or TLS are still on insecure defaults
- User role administration: `GET /api/v1/users/`, `PATCH /api/v1/users/{userID}/role` (admin-only, blocks demoting the last remaining admin, audit-logged), plus a "Users & roles" panel in the Admin tab — previously the only way to promote/demote a user was editing the `users` table by hand
- Clarified in the README that `packages/` and `enterprise/` are unimplemented placeholders, not existing structure
- Fixed audit-log IP capture: `clientIP()` previously read `RemoteAddr`, which behind the nginx reverse proxy is always the proxy's own address; it now prefers `X-Real-IP`/`X-Forwarded-For`
- Redesigned Header (user menu with role badge, change password, sign out) and Footer (platform/changelog/contributing links) to the existing graphite/gold visual language
- Real dashboard screenshot regenerated with headless Chrome against the current build (mocked API data, admin session) and saved to `docs/screenshots/dashboard.png`, replacing the stale pre-redesign mockup
- Versioned releases: pushing a `v*.*.*` tag now builds and publishes all four Docker images (backend, frontend, cve-sync, sdr-agent) to GHCR and cuts a GitHub Release with auto-generated notes (`.github/workflows/release.yml`); versioning policy and release steps documented in `RELEASING.md`
- CubeSat Space Protocol (CSP v1) decoder (`internal/tmtc/csp.go`) and `POST /api/v1/tmtc/decode/csp`: decodes priority, source/destination, source/destination port, reserved bits, and the HMAC/XTEA/RDP/CRC flags from a hex-encoded packet's 32-bit header, plus the payload. Fuzz-tested (~750K executions, zero panics). Added as a fourth protocol tab in the dashboard's Decoder alongside AX.25/CCSDS/PUS, verified end-to-end (decode a sample packet, all fields render correctly)
