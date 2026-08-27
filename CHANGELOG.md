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
