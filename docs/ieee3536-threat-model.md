# Lyrahcle STRIDE Threat Model

This document is the pre-v1 threat model for the open-core Lyrahcle platform. The filename is retained from the project roadmap; this document does not claim certification or conformance with an IEEE standard.

## Scope

In scope:

- browser operator interface
- Go backend API
- PostgreSQL and Redis
- CVE synchronization worker
- SDR agents and RF ingest
- JWT/TOTP authentication and agent API keys
- incident, audit, orbital, RF, and threat-intelligence data
- CI/CD and published container images

Out of scope for this revision:

- third-party ground-station networks not controlled by Lyrahcle
- spacecraft command authorization outside Lyrahcle
- proprietary enterprise integrations
- physical security of SDR hardware and hosting facilities

## Security objectives

1. Prevent unauthorized command, configuration, and administrative actions.
2. Preserve integrity and provenance of RF, orbital, incident, and audit evidence.
3. Keep authentication secrets, TOTP material, agent keys, and private infrastructure data confidential.
4. Remain available under malformed, replayed, or high-volume network and RF input.
5. Make security-relevant actions attributable through authenticated identity, audit events, and request IDs.
6. Keep software dependencies and release artifacts reviewable through reproducible CI evidence and SBOMs.

## Trust boundaries

```text
Operator browser
      |
      | HTTPS + Bearer JWT
      v
Reverse proxy / Backend API
      |             |
      |             +---- Redis
      |                   rate limits, revocation, cache
      |
      +------------------ PostgreSQL
      |                   operational and audit data
      |
      +<--- HTTPS + agent credential --- SDR agents
      |
      +<--- external data --------------- Celestrak / NVD
```

The highest-risk boundaries are browser-to-API, SDR-agent-to-API, API-to-Redis/PostgreSQL, and CI-to-release infrastructure.

## Assets

| Asset | Security property |
|---|---|
| JWT signing keys | Confidentiality, integrity |
| TOTP secrets | Confidentiality |
| Agent API keys | Confidentiality, revocability |
| User roles | Integrity |
| RF observations and IQ captures | Integrity, provenance, availability |
| Incident timelines and evidence | Integrity, auditability |
| Audit log | Integrity, retention |
| PostgreSQL data | Confidentiality, integrity, availability |
| Redis authorization state | Integrity, availability |
| Container images and dependencies | Integrity, provenance |

## STRIDE analysis

### Spoofing

Threats:

- stolen JWT used as an operator
- stolen agent key used to impersonate an SDR station
- spoofed RF source presented as an expected satellite
- forged external threat-intelligence data

Current controls:

- bcrypt credentials, TOTP, JWT issuer/audience validation, short access-token TTL
- JTI and user-wide token revocation
- role enforcement in backend middleware
- per-agent API keys with server-derived agent identity
- TLS with optional private CA trust for SDR agents
- RF anomaly detection and deterministic correlation

Residual work:

- agent enrollment and automatic credential rotation
- mTLS for long-running agents
- RF fingerprint and expected-pass verification

### Tampering

Threats:

- modification of incident state or evidence
- SQL injection or malformed API inputs
- modification of captured IQ files
- dependency or release artifact replacement

Current controls:

- parameterized database access and strict typed validation
- database constraints and transaction-backed incident timelines
- immutable incident evidence references through the API
- role-gated mutations and audit logging
- CI vulnerability/security scans and CycloneDX SBOM generation

Residual work:

- cryptographic hashes/signatures for forensic captures
- signed release provenance and image signing
- append-only or externally anchored audit retention

### Repudiation

Threats:

- operator denies a role, incident, or detection-rule change
- agent activity cannot be tied to a station identity
- request cannot be correlated across logs

Current controls:

- audit events for security-sensitive actions
- request IDs propagated in responses and structured logs
- authenticated user/agent identity at mutation boundaries
- immutable incident timeline entries

Residual work:

- long-term audit export to external immutable storage/SIEM
- cryptographic evidence manifests for forensic exports

### Information disclosure

Threats:

- credentials leaked through logs or backups
- TOTP/JWT/agent secrets returned unintentionally
- database/cache exposed outside trusted networks
- detailed readiness errors expose infrastructure topology

Current controls:

- no request-body logging
- redacted operational logging and request-size bounds
- backup scripts avoid printing database credentials and use restrictive permissions
- readiness response does not reveal the failing dependency
- production TLS and strong JWT-secret checks

Residual work:

- external secrets manager integration
- automated secret rotation
- deployment-specific network policy templates

### Denial of service

Threats:

- login brute force
- oversized API input
- RF ingest or detection-rule alert storms
- Redis/PostgreSQL exhaustion
- hostile IQ capture causing CPU/memory pressure

Current controls:

- independent Redis-backed global and login rate limits
- request body size limits and pagination caps
- bounded active detection-rule evaluation
- per-rule/per-agent cooldown with transactional advisory locks
- bounded fuzz inputs and defensive replay parsing
- readiness probes and graceful shutdown

Residual work:

- per-agent ingest quotas
- deployment-level autoscaling/circuit breaking
- explicit storage and retention budgets

### Elevation of privilege

Threats:

- lower-privileged roles perform unauthorized analyst/operator/admin mutations
- stale JWT retains privileges after a role downgrade
- agent credential reaches human-only endpoints
- last administrator is accidentally removed

Current controls:

- `viewer`, `auditor`, `analyst`, `operator`, and `admin` checks enforced server-side
- role changes revoke all previously issued user sessions
- agent-key middleware provides a separate agent identity context
- last-admin demotion protection
- admin-only assignment, user management, and detection-rule mutation

Residual work:

- privileged-action re-authentication for selected administrative operations
- finer-grained enterprise permissions if multi-tenancy is introduced

## Abuse cases to keep in CI

- logout token is rejected and a fresh login succeeds
- role downgrade invalidates old tokens
- password/TOTP changes invalidate previous sessions
- viewer cannot mutate incidents or detection rules
- operator cannot perform admin-only assignment
- invalid agent key cannot report device/RF data
- duplicated RF anomaly does not create duplicate correlated incidents
- malformed IQ input cannot panic replay/forensic code


## Browser session risk

The current web client sends JWTs explicitly in the `Authorization` header rather than relying on ambient authentication cookies. This substantially limits conventional CSRF exposure, but the current browser token storage remains accessible to JavaScript and therefore makes XSS prevention a critical control. Before v1.0, evaluate migration to an HttpOnly, Secure, SameSite session model with an explicit CSRF design, or document the accepted bearer-token storage risk and compensating controls.

## Review triggers

Update this threat model when any of the following changes:

- authentication or token storage model
- agent enrollment/credential model
- command uplink capability
- multi-tenancy or licensing enforcement
- new externally reachable service
- new persistent data category
- new third-party threat-intelligence source
- production deployment topology

## v8.4 identity boundary update

WebAuthn ceremonies are short-lived, single-use state stored in Redis and bound to the authenticated user. Cross-platform authenticators require user verification, and key enrollment/removal requires the current password. Existing sessions are invalidated when security keys change.

Password recovery is a separate trust boundary. The public request endpoint is non-enumerating; reset secrets are random, hashed at rest, single-use, and short-lived. TOTP-protected accounts require a current TOTP during recovery. WebAuthn-only accounts cannot downgrade through email recovery and require administrator-assisted restoration.

OTLP tracing must never carry bearer tokens, WebAuthn responses, TOTP secrets, password-reset URLs, passwords, agent keys, or raw RF evidence as attributes.
