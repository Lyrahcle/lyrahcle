# API reference

Base URL: `http://localhost:8080` (or wherever `PORT`/`TLS_CERT_FILE` point it).
All routes below are under `/api/v1` unless noted otherwise.

**Interactive docs**: the backend serves a full OpenAPI 3.0 spec at
[`/openapi.yaml`](http://localhost:8080/openapi.yaml) and a browsable Swagger UI
at [`/docs`](http://localhost:8080/docs) — both embedded in the binary, no
extra setup required. The source spec lives at
[`apps/backend/cmd/server/openapi.yaml`](../apps/backend/cmd/server/openapi.yaml).

## Authentication

The entire app requires a login — both the dashboard and every endpoint
except `/healthz`, `/ping`, `/auth/register`, and `/auth/login`. A default
admin account is seeded automatically on first boot if no users exist yet:

```
email:    admin   (override via SEED_ADMIN_EMAIL)
password: admin   (override via SEED_ADMIN_PASSWORD)
```

**This is a well-known default and must be changed immediately** — the
backend prints a loud warning banner in its logs on the boot that creates it.

```bash
curl -X POST localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "admin", "password": "admin"}'
# -> {"token": "...", "role": "admin"}

curl -X POST localhost:8080/api/v1/auth/change-password \
  -H "Authorization: Bearer <token>" -H "Content-Type: application/json" \
  -d '{"current_password": "admin", "new_password": "a-much-stronger-password"}'
```

Roles: `viewer` < `operator` < `admin`. Admins can promote or demote any
user's role from the Admin tab in the dashboard, or via
`PATCH /api/v1/users/{userID}/role`.

## Agent authentication (sdr-agent)

Two ways for `sdr-agent` to authenticate against `POST /rf/ingest`:

- **Per-agent API key** (recommended) — scoped to one `agent_id`, revocable
  independently, doesn't expire. Only the SHA-256 hash is stored; the
  plaintext key is shown exactly once, at creation time.
- **Operator/admin bearer token** (legacy/fallback) — the shared human JWT
  from `/auth/login`, kept for backward compatibility. Prefer the agent key
  for anything long-running.

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
(`docker-compose.yml`); it's sent as an `X-Agent-Key` header, separately from
the `Authorization: Bearer` header used by human sessions.

## Endpoints

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/healthz` | — | Liveness check |
| `GET` | `/metrics` | — | Prometheus metrics (request count/latency by method, route, status) |
| `GET` | `/api/v1/ping` | — | Backend reachability check |
| `GET` | `/api/v1/assets` | viewer+ | List tracked assets (paginated) |
| `POST` | `/api/v1/auth/register` | — | Create a user (first user ever becomes admin) |
| `POST` | `/api/v1/auth/login` | — | Exchange email + password for a bearer token |
| `POST` | `/api/v1/auth/change-password` | viewer+ | Change your own password (also revokes the token used to make this request — see Redis-backed features below) |
| `POST` | `/api/v1/auth/logout` | viewer+ | Revoke the current token immediately, before its natural expiry (requires Redis — a no-op if unavailable) |
| `GET` | `/api/v1/users/` | admin | List all users and their roles |
| `PATCH` | `/api/v1/users/{userID}/role` | admin | Promote or demote a user's role (blocks demoting the last remaining admin) |
| `GET` | `/api/v1/audit-log` | admin | List security-relevant events (paginated) |
| `POST` | `/api/v1/agent-keys` | admin | Create an API key scoped to one `agent_id` |
| `DELETE` | `/api/v1/agent-keys/{agentID}` | admin | Revoke that agent's key |
| `POST` | `/api/v1/rf/ingest` | agent key or operator+ | Ingest an RF reading (used by `sdr-agent`) |
| `GET` | `/api/v1/rf/readings` | viewer+ | List recent RF readings |
| `GET` | `/api/v1/rf/anomalies` | viewer+ | List recent RF anomalies (statistical outliers in power or frequency vs. that agent's baseline) |
| `GET` | `/api/v1/satellites/` | viewer+ | List tracked satellites with live computed position |
| `POST` | `/api/v1/satellites/` | operator+ | Register a satellite by NORAD ID + TLE |
| `POST` | `/api/v1/tmtc/decode/ax25` | viewer+ | Decode a hex-encoded AX.25 UI frame (destination/source callsign+SSID, control, PID, payload) |
| `POST` | `/api/v1/tmtc/decode/ccsds` | viewer+ | Decode a hex-encoded CCSDS Space Packet (version, TM/TC type, secondary header flag, APID, sequence flags/count, payload) |
| `POST` | `/api/v1/tmtc/decode/pus` | viewer+ | Decode a hex-encoded PUS packet (CCSDS header fields + PUS version, service type/subtype, message counter, destination ID, payload) |
| `POST` | `/api/v1/tmtc/decode/csp` | viewer+ | Decode a hex-encoded CubeSat Space Protocol (CSP v1) packet (priority, source/destination, source/destination port, HMAC/XTEA/RDP/CRC flags, payload) |
| `GET` | `/api/v1/threat-intel/space-cves` | viewer+ | List space-related CVEs synced from the NVD (auto-updates daily, paginated) |
| `GET` | `/api/v1/asset-alerts/` | viewer+ | List CVE-to-asset alerts, raised automatically when a synced CVE is relevant to a tracked asset's kind (paginated) |
| `POST` | `/api/v1/asset-alerts/{alertID}/acknowledge` | operator+ | Mark an alert as reviewed |

`/assets`, `/audit-log`, `/threat-intel/space-cves`, and `/asset-alerts` accept `?limit=` (default
50, max 200) and `?offset=` query params, and respond with
`{"items": [...], "total": N, "limit": L, "offset": O}` instead of a bare
array.

Every endpoint above also has a matching panel or form in the dashboard —
none of them require `curl` for day-to-day use.

## Redis-backed features

Three things are backed by Redis when `REDIS_URL` is reachable at startup —
all optional, all degrade gracefully (a warning is logged, the feature is
simply disabled) if Redis isn't available:

- **Rate limiting** persists across restarts and is shared correctly
  across multiple backend replicas, instead of each process keeping its
  own in-memory counter.
- **Token revocation**: `POST /auth/logout` and `POST /auth/change-password`
  both invalidate the JWT used to make the request immediately, rather than
  leaving it valid until its natural expiry (up to 24h later).
- **Response caching**: `GET /satellites/` (5s TTL) and
  `GET /threat-intel/space-cves` (30s TTL) are cached, since both are
  identical for every caller and the satellites list in particular is
  polled by every open dashboard every few seconds.
