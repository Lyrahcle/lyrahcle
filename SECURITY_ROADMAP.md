# Lyrahcle — Short-term Security & Hardening Roadmap

Tracked incrementally, one item at a time, before the v1.0.0 release.

- [x] **Authentication & authorization** — users, roles, tokens; protect sensitive
      endpoints like `/rf/ingest` and `/satellites/` (JWT + bcrypt + role hierarchy
      shipped; admin-managed role promotion and per-agent API keys still pending)
- [x] **TLS/HTTPS** across all frontend/backend/sdr-agent communication (opt-in:
      backend serves HTTPS directly when `TLS_CERT_FILE`/`TLS_KEY_FILE` are set,
      nginx has an HTTPS+redirect config, `sdr-agent` supports `https://` backends
      and a dev-only cert-skip-verify flag; `scripts/generate-dev-certs.sh`
      generates a local self-signed cert. Plain HTTP stays the default with a
      startup warning, matching the `JWT_SECRET` pattern, so local dev isn't
      broken. Production should prefer a real CA cert via a reverse proxy.)
- [x] **Strict input validation** on all endpoints, especially AX.25 decoding and RF
      ingestion, to prevent injection and malformed-data issues (`internal/validate`:
      email/password, agent_id, frequency/power/timestamp ranges, NORAD ID, TLE line
      format, AX.25 hex length; 1MB global request body cap)
- [x] **Automated test coverage** (unit, integration, fuzzing) for backend, frontend,
      and decoders before v1.0.0 (backend: unit tests for `auth`, `validate`, `tmtc`,
      `celestrak` — `tmtc` at 100% statement coverage; fuzz tests for the AX.25
      decoder ran for ~1M executions across both entry points with 0 crashes/panics
      found; integration tests for `store` against a real Postgres, skip cleanly
      without `TEST_DATABASE_URL`; frontend: Vitest + Testing Library for components
      and the API client. CI runs all of it, including a Postgres service container.)
- [x] **Rate limiting** and abuse protection on the public API (120 req/min per IP
      globally, 10 req/min per IP on `/auth/register` and `/auth/login` to slow down
      credential stuffing/brute force)
- [x] **SDR agent authentication** (API keys or certificates) to prevent fake RF
      ingestion (`agent_keys` table, `POST`/`DELETE /agent-keys`, SHA-256-hashed
      keys — plaintext shown once at creation, sent as `X-Agent-Key`; the backend
      rejects a reading whose `agent_id` doesn't match the authenticated key's
      registered agent, preventing one agent from spoofing another's identity;
      falls back to the existing operator/admin JWT for backward compatibility;
      unit-tested including the fallback path, integration-tested against real
      Postgres including revocation actually cutting off access)
- [x] **Improved documentation** — deployment guides, configuration, endpoint usage,
      contribution guide (`docs/deployment.md`: production checklist, Compose vs
      Kubernetes, TLS options, backups, monitoring, rollout; `docs/configuration.md`:
      every env var across every service in one table; `CONTRIBUTING.md`: local
      setup, coding conventions, testing, security-sensitive-change guidelines, PR
      workflow; README gained a documentation map linking all of it. Endpoint usage
      was already covered by the README's API table and per-feature curl examples.)
- [ ] **Real SDR hardware integration** (RTL-SDR, HackRF, etc.) to validate capture
      beyond the mock source
- [ ] **Security logging & audit trail** in backend and agents
- [ ] **Versioned releases & published packages** (Docker Hub, npm, etc.)

## Status notes

- 2026-08-27: shipped authentication & authorization — JWT (HS256) + bcrypt password
  hashing, `users` table with `admin`/`operator`/`viewer` roles, `POST /auth/register`
  (first user becomes admin) and `POST /auth/login`, `POST /rf/ingest` and
  `POST /satellites/` now require an `operator`-or-higher bearer token, `sdr-agent`
  sends its token via `AGENT_TOKEN`.
- 2026-08-27: shipped strict input validation (`internal/validate`) across
  `/auth/register`, `/auth/login`, `/rf/ingest`, `POST /satellites/`, and
  `/tmtc/decode/ax25` — bounded ranges for frequency/power/timestamps, NORAD ID and
  TLE line format checks, AX.25 hex length cap, tightened email regex (rejected a
  `<script>...</script>@x.com` payload that the first-pass regex let through), and a
  1MB global request body limit.
- 2026-08-27: shipped rate limiting (`go-chi/httprate`) — 120 req/min per IP across
  the whole API, 10 req/min per IP on `/auth/register` and `/auth/login`; verified
  with a standalone test (5-req limit: first 5 requests 200, next 3 return 429).
- 2026-08-27: shipped opt-in TLS/HTTPS — backend serves HTTPS natively when
  `TLS_CERT_FILE`/`TLS_KEY_FILE` are set (verified with a real self-signed cert:
  `curl -k` succeeds, plain `curl` correctly fails on the untrusted cert), an nginx
  HTTPS+redirect config for the frontend, `sdr-agent` supports `https://` backend
  URLs plus a dev-only `AGENT_INSECURE_SKIP_VERIFY` flag, and
  `scripts/generate-dev-certs.sh` to generate a local cert. Defaults to plain HTTP
  with a startup warning so existing local setups keep working.
- 2026-08-27: shipped automated tests. Backend: unit tests for `internal/auth`
  (password hashing, JWT round-trip, expiry, wrong-secret rejection, role ranking),
  `internal/validate` (every validator, valid/invalid/malicious cases), `internal/tmtc`
  (AX.25 round-trip, edge cases — 100% coverage), `internal/celestrak` (TLE parsing).
  Fuzz tests for both AX.25 decode entry points ran ~1M executions combined with zero
  panics. Integration tests for `internal/store` against a real Postgres (installed
  locally to verify, not just mocked) covering upsert idempotency and unique-email
  handling; they skip cleanly when `TEST_DATABASE_URL` is unset. Frontend: Vitest +
  Testing Library covering `SignalBar` (tone thresholds, width clamping), `KindBadge`,
  and the `lib/api.js` fetch wrappers (15 tests, all passing). CI now actually runs
  build+test+vet for both apps against a real Postgres service container, replacing
  the previous placeholder steps that silently skipped when files didn't exist yet.
- 2026-08-27: shipped SDR agent authentication — `agent_keys` table (SHA-256 hash
  only, never plaintext), `POST /agent-keys` (admin-only, returns the plaintext key
  once) and `DELETE /agent-keys/{agentID}`, a combined middleware that accepts
  `X-Agent-Key` first and falls back to the existing operator/admin JWT, and a
  server-side check rejecting any RF reading whose `agent_id` doesn't match the
  authenticated key's registered agent (blocks one agent from spoofing another's
  identity even with a valid key). `sdr-agent` now sends `AGENT_API_KEY` via
  `X-Agent-Key` when set, falling back to the old `AGENT_TOKEN` bearer flow for
  compatibility. Unit tests cover the middleware fallback logic; integration tests
  against real Postgres cover create/lookup/duplicate/revoke, including confirming
  a revoked key stops authenticating immediately.
- 2026-08-27: shipped expanded documentation — `docs/deployment.md` (production
  checklist, Docker Compose vs Kubernetes, TLS options ranked by recommendation,
  database/backup notes, monitoring, rollout steps), `docs/configuration.md` (every
  env var across backend/frontend/sdr-agent/cve-sync/compose in one reference),
  `CONTRIBUTING.md` (local setup for both Docker and bare-metal dev, coding
  conventions including the no-comments/English-only rules, testing, a
  security-sensitive-change checklist, PR workflow, commit style), and a
  documentation map at the top of the README linking all of it. Cross-checked every
  factual claim against the actual code (e.g. confirmed `REDIS_URL` is loaded but
  unused by any code path, and that `deploy/k8s`/`deploy/helm` are placeholder
  directories) rather than assuming.
  Next: audit logging (item 9), then versioned releases (item 10).
