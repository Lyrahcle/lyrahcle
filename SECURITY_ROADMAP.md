# Lyrahcle — Short-term Security & Hardening Roadmap

Tracked incrementally, one item at a time, before the v1.0.0 release.

- [x] **Authentication & authorization** — users, roles, tokens; protect sensitive
      endpoints like `/rf/ingest` and `/satellites/` (JWT + bcrypt + role hierarchy
      shipped; admin-managed role promotion and per-agent API keys still pending)
- [ ] **TLS/HTTPS** across all frontend/backend/sdr-agent communication
- [ ] **Strict input validation** on all endpoints, especially AX.25 decoding and RF
      ingestion, to prevent injection and malformed-data issues
- [ ] **Automated test coverage** (unit, integration, fuzzing) for backend, frontend,
      and decoders before v1.0.0
- [ ] **Rate limiting** and abuse protection on the public API
- [ ] **SDR agent authentication** (API keys or certificates) to prevent fake RF ingestion
- [ ] **Improved documentation** — deployment guides, configuration, endpoint usage,
      contribution guide
- [ ] **Real SDR hardware integration** (RTL-SDR, HackRF, etc.) to validate capture
      beyond the mock source
- [ ] **Security logging & audit trail** in backend and agents
- [ ] **Versioned releases & published packages** (Docker Hub, npm, etc.)

## Status notes

- 2026-08-27: shipped authentication & authorization — JWT (HS256) + bcrypt password
  hashing, `users` table with `admin`/`operator`/`viewer` roles, `POST /auth/register`
  (first user becomes admin) and `POST /auth/login`, `POST /rf/ingest` and
  `POST /satellites/` now require an `operator`-or-higher bearer token, `sdr-agent`
  sends its token via `AGENT_TOKEN`. Next: role management endpoint, per-agent API
  keys (item 6), then TLS (item 2).
