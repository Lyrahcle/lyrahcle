# API Reference

Base URL: `http://localhost:8080`. The backend also serves Swagger UI at `/docs` and the OpenAPI document at `/openapi.yaml`.

## Authentication

Public self-registration is disabled. On an empty database the backend creates one bootstrap admin from `SEED_ADMIN_EMAIL` and `SEED_ADMIN_PASSWORD`.

Development defaults:

```text
identifier: admin
password:   admin
```

Change them before production. An existing admin password is not reset by changing the seed environment variables.

Login accepts an email or username:

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"identifier":"admin","password":"admin"}'
```

Use the returned token as `Authorization: Bearer <token>`. Access tokens default to a 30-minute lifetime, carry fixed Lyrahcle issuer/audience claims, and are invalidated when the user changes password/TOTP state or an administrator changes the user role.

Roles are `viewer`, `auditor`, `analyst`, `operator`, and `admin`. `auditor` adds audit-log access to read-only operations; `analyst` adds incident-investigation actions; `operator` adds operational mutations; `admin` manages identities and security configuration. Admins create users through `POST /api/v1/users/` or the dashboard.

## Google Authenticator

The dashboard exposes one 2FA setup flow: Google Authenticator using standard TOTP.

1. `POST /api/v1/auth/totp/setup` returns a Google Authenticator setup key and provisioning URI.
2. Add the key in Google Authenticator as a time-based key.
3. `POST /api/v1/auth/totp/enable` with `{"code":"123456"}` verifies the first code and enables 2FA.

When enabled, login without `totp_code` returns `401` with `{"error":"totp_required"}`. Enabling or disabling 2FA revokes all existing sessions and requires a fresh login. TOTP is an interoperable standard, so the server validates the code rather than the brand of client that generated it.

## Agent authentication

Long-running SDR agents should use individual API keys created by an admin:

```bash
curl -X POST http://localhost:8080/api/v1/agent-keys/ \
  -H 'Authorization: Bearer <admin-token>' \
  -H 'Content-Type: application/json' \
  -d '{"agent_id":"sdr-agent-madrid"}'
```

The plaintext key is returned once and is then stored only as a hash. Agents send it as `X-Agent-Key`.

`POST /api/v1/agents/me/device` requires agent-key identity. The backend derives the agent from the credential; the request body cannot select a different agent.

## Request IDs

Every backend response includes `X-Request-ID`. A safe incoming `X-Request-ID` may be preserved; unsafe values are replaced. Include this ID when correlating frontend errors, reverse-proxy logs, backend logs, and operator reports.

## Endpoints

| Method | Path | Minimum access | Purpose |
|---|---|---|---|
| `GET` | `/healthz` | Public | Liveness |
| `GET` | `/readyz` | Public | Dependency readiness without dependency details |
| `GET` | `/metrics` | Public | Prometheus metrics |
| `GET` | `/openapi.yaml` | Public | OpenAPI document |
| `GET` | `/docs` | Public | Swagger UI |
| `GET` | `/api/v1/ping` | Public | API reachability |
| `GET` | `/api/v1/system-info` | Viewer | Runtime/system information |
| `POST` | `/api/v1/auth/login` | Public | Login; rate limited independently from normal API traffic |
| `GET` | `/api/v1/auth/me` | Viewer | Current account and Google Authenticator state |
| `GET` | `/api/v1/auth/sessions/` | Viewer | List active sessions owned by the current user |
| `DELETE` | `/api/v1/auth/sessions/{sessionID}` | Viewer | Revoke one owned session |
| `DELETE` | `/api/v1/auth/sessions/` | Viewer | Revoke all sessions for the current user |
| `POST` | `/api/v1/auth/logout` | Viewer | Revoke the current JWT |
| `POST` | `/api/v1/auth/change-password` | Viewer | Change password and invalidate all older sessions for the user |
| `POST` | `/api/v1/auth/totp/setup` | Viewer | Generate pending Google Authenticator setup key |
| `POST` | `/api/v1/auth/totp/enable` | Viewer | Verify and enable Google Authenticator |
| `POST` | `/api/v1/auth/totp/disable` | Viewer | Disable Google Authenticator after password verification |
| `GET` | `/api/v1/users/` | Admin | List users |
| `POST` | `/api/v1/users/` | Admin | Create user with role |
| `PATCH` | `/api/v1/users/{userID}/role` | Admin | Change role; last active admin cannot be demoted |
| `PATCH` | `/api/v1/users/{userID}/status` | Admin | Enable or disable an account |
| `DELETE` | `/api/v1/users/{userID}` | Admin | Privacy-preserving soft delete |
| `GET` | `/api/v1/programs/` | Viewer | List programs |
| `POST` | `/api/v1/programs/` | Operator | Create program |
| `GET` | `/api/v1/assets/` | Viewer | List assets |
| `POST` | `/api/v1/assets/` | Operator | Create/update asset |
| `GET` | `/api/v1/satellites/` | Viewer | Satellites with propagated position |
| `GET` | `/api/v1/satellites/stream` | Viewer | Live satellite SSE stream |
| `POST` | `/api/v1/satellites/` | Operator | Register/update satellite TLE |
| `GET` | `/api/v1/agent-keys/` | Admin | List agent key/device state |
| `POST` | `/api/v1/agent-keys/` | Admin | Issue agent key |
| `DELETE` | `/api/v1/agent-keys/{agentID}` | Admin | Revoke agent key |
| `POST` | `/api/v1/agents/me/device` | Agent key | Report authenticated SDR device metadata |
| `POST` | `/api/v1/rf/ingest` | Agent key or Operator | Ingest RF reading |
| `GET` | `/api/v1/rf/readings` | Viewer | Recent RF readings |
| `GET` | `/api/v1/rf/anomalies` | Viewer | Recent RF anomalies |
| `GET` | `/api/v1/detection-rules/` | Viewer | List typed RF detection rules |
| `POST` | `/api/v1/detection-rules/` | Admin | Create a detection rule |
| `PATCH` | `/api/v1/detection-rules/{ruleID}` | Admin | Update or disable a detection rule |
| `GET` | `/api/v1/incidents/` | Viewer | Filtered/paginated security incidents |
| `POST` | `/api/v1/incidents/` | Operator | Create incident; assignment is admin-only |
| `GET` | `/api/v1/incidents/stream` | Viewer | Live incident SSE snapshots |
| `GET` | `/api/v1/incidents/{incidentID}` | Viewer | Incident detail and timeline |
| `PATCH` | `/api/v1/incidents/{incidentID}` | Operator | Update workflow fields; assignment is admin-only |
| `POST` | `/api/v1/incidents/{incidentID}/notes` | Operator | Add investigation note |
| `POST` | `/api/v1/incidents/{incidentID}/evidence` | Operator | Add immutable typed evidence reference |
| `POST` | `/api/v1/tmtc/decode/ax25` | Viewer | Decode AX.25 |
| `POST` | `/api/v1/tmtc/decode/ccsds` | Viewer | Decode CCSDS Space Packet |
| `POST` | `/api/v1/tmtc/decode/pus` | Viewer | Decode PUS |
| `POST` | `/api/v1/tmtc/decode/csp` | Viewer | Decode CSP v1 |
| `GET` | `/api/v1/threat-intel/space-cves` | Viewer | Filtered/paginated space CVEs |
| `GET` | `/api/v1/threat-intel/space-cves/keywords` | Viewer | Available CVE keywords |
| `GET` | `/api/v1/asset-alerts/` | Viewer | CVE-to-asset alerts |
| `POST` | `/api/v1/asset-alerts/{alertID}/acknowledge` | Operator | Acknowledge alert |
| `GET` | `/api/v1/audit-log` | Admin | Security audit trail |

## Pagination and CVE filters

Paginated endpoints accept `limit` and `offset`; `limit` defaults to 25 and is capped at 100.

`GET /api/v1/threat-intel/space-cves` also supports:

- `severity`
- `keyword`
- `published_from=YYYY-MM-DD`
- `published_to=YYYY-MM-DD`
- `sort`
- `order=asc|desc`

## Satellite SSE

`GET /api/v1/satellites/stream` returns `text/event-stream`, sends an initial `satellites` event immediately, then refreshes every five seconds until the client disconnects.

## Redis-dependent behavior

Redis enables distributed rate limiting, JWT revocation, active-session tracking, response caching, and the CVE event queue. Development can fall back for selected non-authorization features, but production startup fails when Redis is unavailable so revocation guarantees are not silently lost.

## Incident response

Incidents support filters for status, severity, agent, and satellite, and use validated workflow values for severity and status. Incident creation accepts an optional `Idempotency-Key`, and duplicate keys return the original incident instead of creating a second record. Viewers have read-only access, operators can create incidents, change status/severity, and add timeline notes, while assignment changes are restricted to admins. RF anomalies are evaluated against typed detection rules. Rules support a fixed metric set, directional or absolute z-score comparisons, validated thresholds, severity, cooldown suppression, and optional automatic incident creation. Only admins can create or modify rules; viewers can inspect them. The platform ships with high and critical default deviation rules that preserve the previous escalation behavior. Multiple rule matches for the same anomaly are correlated into one incident using the highest matching auto-incident severity.

The `source_anomaly_id` uniqueness constraint prevents one RF anomaly from generating duplicate incidents, while persisted rule matches and per-rule/per-agent cooldown locks reduce alert storms. Workflow transitions are enforced in both Go and PostgreSQL. Operator notes and immutable evidence references are recorded in the incident timeline, and security-relevant mutations are also written to the audit log.


### User lifecycle

Admin-only endpoints:

- `PATCH /api/v1/users/{userID}/role` changes the role and invalidates existing authorization state.
- `PATCH /api/v1/users/{userID}/status` enables or disables the account.
- `DELETE /api/v1/users/{userID}` performs privacy-preserving soft deletion.

`auditor` can read `/api/v1/audit-log` without user-management privileges. `analyst` can update incident workflow state, add notes, and attach evidence without gaining operator-level RF or asset mutation permissions.
