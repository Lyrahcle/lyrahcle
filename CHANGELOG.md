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
