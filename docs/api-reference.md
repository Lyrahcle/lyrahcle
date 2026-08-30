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

Long-running SDR agents should use v9.0 certificate enrollment and mTLS. An administrator creates a single-use enrollment token, the agent generates its private key and CSR locally, and the backend returns a short-lived client certificate. The private key never leaves the agent.

Legacy `X-Agent-Key` credentials remain available only for migration. When a client certificate is configured, the SDR agent does not send its API key.

`POST /api/v1/agents/me/device` and `POST /api/v1/rf/ingest` derive the agent identity from the validated client certificate or legacy agent key. A request cannot select a different authenticated agent.

See `docs/agent-enrollment.md` for certificate setup, rotation, quarantine, and revocation.

The embedded OpenAPI document is OpenAPI 3.0.3. It documents mTLS requirements in endpoint descriptions; the actual client-certificate authentication is enforced by the Go TLS/authentication layer rather than represented as an OpenAPI 3.0 security scheme.

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
| `GET` | `/api/v1/users/` | `users.read` | List users |
| `POST` | `/api/v1/users/` | `users.manage` | Create user with role |
| `PATCH` | `/api/v1/users/{userID}/role` | `users.manage` | Change role; last active admin cannot be demoted |
| `PATCH` | `/api/v1/users/{userID}/status` | `users.manage` | Enable or disable an account |
| `DELETE` | `/api/v1/users/{userID}` | `users.manage` | Privacy-preserving soft delete |
| `GET` | `/api/v1/programs/` | `programs.read` | List programs |
| `POST` | `/api/v1/programs/` | `programs.manage` | Create program |
| `GET` | `/api/v1/assets/` | `assets.read` | List assets |
| `POST` | `/api/v1/assets/` | `assets.manage` | Create/update asset |
| `GET` | `/api/v1/satellites/` | `orbital.read` | Satellites with propagated position |
| `GET` | `/api/v1/satellites/{satelliteID}/passes` | `orbital.read` | Predict AOS/LOS and peak elevation for a ground station; rejects stale/unusable TLEs |
| `GET` | `/api/v1/satellites/stream` | `orbital.read` | Live satellite SSE stream |
| `POST` | `/api/v1/satellites/` | `orbital.manage` | Register/update satellite TLE |
| `GET` | `/api/v1/agents/` | `agents.read` or `agents.manage` | List mTLS agent identity, certificate, version, and device posture |
| `POST` | `/api/v1/agents/enrollment-tokens` | `agents.manage` | Issue single-use agent enrollment token |
| `POST` | `/api/v1/agents/enroll` | Enrollment token | Exchange CSR for client certificate |
| `PATCH` | `/api/v1/agents/{agentID}/status` | `agents.manage` | Quarantine, activate, or permanently revoke an agent |
| `POST` | `/api/v1/agents/me/certificate/rotate` | mTLS agent | Rotate the current client certificate |
| `GET` | `/api/v1/agent-keys/` | `agents.read` or `agents.manage` | List legacy agent key/device state |
| `POST` | `/api/v1/agent-keys/` | `agents.manage` | Issue agent key |
| `DELETE` | `/api/v1/agent-keys/{agentID}` | `agents.manage` | Revoke agent key |
| `POST` | `/api/v1/agents/me/device` | mTLS agent or legacy agent key | Report authenticated SDR device metadata |
| `POST` | `/api/v1/rf/ingest` | mTLS agent, legacy agent key, or `rf.ingest` | Ingest RF reading |
| `GET` | `/api/v1/rf/readings` | `rf.read` | Recent RF readings |
| `GET` | `/api/v1/rf/anomalies` | `rf.read` | Recent RF anomalies |
| `GET` | `/api/v1/detection-rules/` | `detections.read` | List typed RF detection rules |
| `POST` | `/api/v1/detection-rules/` | `detections.manage` | Create a detection rule |
| `PATCH` | `/api/v1/detection-rules/{ruleID}` | `detections.manage` | Update or disable a detection rule |
| `GET` | `/api/v1/incidents/` | `incidents.read` | Filtered/paginated security incidents |
| `POST` | `/api/v1/incidents/` | `incidents.manage` | Create incident; assignment is allowed for incident managers |
| `GET` | `/api/v1/incidents/stream` | `incidents.read` | Live incident SSE snapshots |
| `GET` | `/api/v1/incidents/{incidentID}` | `incidents.read` | Incident detail and timeline |
| `PATCH` | `/api/v1/incidents/{incidentID}` | `incidents.manage` | Update workflow fields, including assignment |
| `POST` | `/api/v1/incidents/{incidentID}/notes` | `incidents.manage` | Add investigation note |
| `POST` | `/api/v1/incidents/{incidentID}/evidence` | `incidents.manage` | Add immutable typed evidence reference |
| `POST` | `/api/v1/tmtc/decode/ax25` | Viewer | Decode AX.25 |
| `POST` | `/api/v1/tmtc/decode/ccsds` | Viewer | Decode CCSDS Space Packet |
| `POST` | `/api/v1/tmtc/decode/pus` | Viewer | Decode PUS |
| `POST` | `/api/v1/tmtc/decode/csp` | Viewer | Decode CSP v1 |
| `GET` | `/api/v1/threat-intel/space-cves` | Viewer | Filtered/paginated space CVEs |
| `GET` | `/api/v1/threat-intel/space-cves/keywords` | Viewer | Available CVE keywords |
| `GET` | `/api/v1/asset-alerts/` | `alerts.read` | CVE-to-asset alerts |
| `POST` | `/api/v1/asset-alerts/{alertID}/acknowledge` | `alerts.manage` | Acknowledge alert |
| `GET` | `/api/v1/audit-log` | `audit.read` | Security audit trail |

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

## WebAuthn and recovery

- `POST /api/v1/auth/webauthn/login/start` starts password + security-key authentication.
- `POST /api/v1/auth/webauthn/login/finish` validates the one-time WebAuthn assertion and issues the session.
- `GET /api/v1/auth/webauthn/credentials/` lists the current user's keys.
- `POST /api/v1/auth/webauthn/credentials/register/start` requires the current password and starts enrollment.
- `POST /api/v1/auth/webauthn/credentials/register/finish` completes enrollment and revokes prior sessions.
- `DELETE /api/v1/auth/webauthn/credentials/{credentialID}` requires the current password and removes one owned key.
- `POST /api/v1/auth/password-reset/request` requests a generic, non-enumerating temporary-link recovery flow when enabled.
- `POST /api/v1/auth/password-reset/complete` consumes the single-use token and enforces the account's recovery-factor policy.

## Global search

`GET /api/v1/search?q=...` searches satellites, assets, CVEs, and incidents for authenticated users. Queries are length-limited and parameterized; results are capped by the backend.

## SDR agent remote configuration

v9.1 adds admin-managed, immutable configuration versions for active mTLS agents.

| Method | Path | Access | Purpose |
|---|---|---|---|
| `GET` | `/api/v1/agents/{agentID}/config` | admin | Read desired version and acknowledgement |
| `PUT` | `/api/v1/agents/{agentID}/config` | admin | Publish frequency/interval settings |
| `GET` | `/api/v1/agents/{agentID}/config/history` | admin | Read latest immutable versions |
| `POST` | `/api/v1/agents/{agentID}/config/rollback` | admin | Create a new version from historical settings |
| `GET` | `/api/v1/agents/me/config` | mTLS agent | Poll a newer signed version |
| `POST` | `/api/v1/agents/me/config/ack` | mTLS agent | Record `applied` or `rejected` |

The agent endpoints require a currently active client certificate. API keys and human JWTs cannot be used to fetch or acknowledge remote configuration.


## Orbital security context

`PUT /api/v1/agents/{agentID}/station` is admin-only and stores trusted ground-station latitude, longitude, and altitude for orbital correlation.

Signed agent configuration accepts optional `satellite_id`. RF ingestion accepts the same optional UUID and rejects unknown satellites. Satellite-tagged anomalies retain the association in incidents and evidence.


## v9.4 cross-domain correlation evidence

Automatically generated RF detection incidents may contain `correlation` evidence with `evidence_ref` `cross-domain-v9.4`. The metadata includes a bounded confidence score, corroborating domains, context-only domains, candidate severity, and human-readable reasons. The v9.4 result is explainability/candidate evidence only and does not independently mutate incident severity.

Recent agent audit activity may be attached as `audit` evidence. Recent unacknowledged asset CVE alerts may be attached as `asset_alert_context`; these entries are explicitly `context_only` because the current data model does not establish a direct causal link between a global asset alert and a specific RF incident.
