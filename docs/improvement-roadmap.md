# Improvement Roadmap

This is the canonical shipped/next/later backlog for the current pre-v1 line. Milestone history lives in `CHANGELOG.md`; security-specific evidence and gates live in `SECURITY_ROADMAP.md`.

Status values: **Shipped**, **Partial**, **Next**, and **Later**.

## Identity, authentication, and authorization

| Improvement | Status | Target |
|---|---|---|
| Google Authenticator TOTP | Shipped | Current |
| Active session inventory and revocation | Shipped | Current |
| Login-specific Redis rate limiting | Shipped | Current |
| Strict input validation | Shipped | Current |
| JWT signing-key rotation overlap and user-wide revocation | Shipped | Current |
| WebAuthn security-key second factor | Shipped | v8.4 |
| Password recovery with TOTP and temporary link | Shipped | v8.4 |
| Force password change on first default-admin login | Shipped | Current |
| Stronger password policy | Shipped | Current |
| Enforce TLS for all production authentication traffic | Shipped | Current |
| CSRF protection if cookie-based authentication is introduced | Partial | Before v1.0 |
| Automatic agent certificate rotation | Shipped | v9.0 |
| Agent enrollment and mTLS | Shipped | v9.0 |
| Additional roles such as analyst and auditor | Shipped | Current |
| Administrator subroles | Later | v9 |
| Explicit permission matrix | Shipped | Current |
| Disable and soft-delete users | Shipped | Current |
| Complete sensitive-endpoint authorization review | Partial | Before v1.0 |
| Complete administrative audit-coverage review | Partial | Before v1.0 |

## Detection, RF, and orbital security

| Improvement | Status | Target |
|---|---|---|
| PUS and CSP decoders | Shipped | Current |
| Robust RF anomaly detection | Shipped | Current |
| Typed RF detection rules and deterministic correlation | Shipped | Current |
| Explainable cross-domain correlation | Shipped | v9.4 |
| Correlation-driven severity policy after production validation | Later | Before v1.0 |
| RF heat map | Next | v9 |
| Telemetry, RF, CVE, and operations charts | Next | v8.3-v9 |
| Orbital-aware RF detection and TLE freshness gating | Shipped | v9.2/v9.2.2 |
| Physical SDR validation of Doppler/visibility correlation | Next | Before auto-escalation/v1.0 |
| Pass prediction and AOS/LOS scheduling | Shipped | v9.3 |
| Maneuver/conjunction context persistence and candidate-only correlation | Shipped | v9.5 |
| Automatic maneuver inference and trusted conjunction screening feeds | Next | v9.x |
| More threat-intelligence sources | Later | v9 |
| ML-assisted anomaly detection | Later | Post-v1 |
| Advanced/proprietary RF decoders | Later | Enterprise |
| SIEM integrations | Later | Enterprise |

## Performance and scale

| Improvement | Status | Target |
|---|---|---|
| CVE filter indexes | Shipped | Current |
| Response/orbital/CVE caching | Partial | v8.3-v9 |
| Satellite-load optimization | Shipped | v8.5 |
| Index review for all filtered searches | Partial | v8.5-v9 |
| SQL query optimization | Partial | v8.5-v9 |
| Load-test tooling | Shipped | v8.4 |
| Station federation | Later | v9+ |
| Multi-tenant architecture | Later | Enterprise |

## Platform security and supply chain

| Improvement | Status | Target |
|---|---|---|
| CycloneDX SBOM in CI/releases | Shipped | Current |
| STRIDE threat model | Shipped | Current |
| SDR-agent fuzzing | Shipped | Current |
| Dependency and vulnerability gates | Shipped | Current |
| Cosign image signing and build provenance | Shipped | Current |
| Security design review for enterprise modules | Later | Enterprise |
| Compliance-readiness checklist | Shipped | Current |
| Full >80% application coverage | Partial | Before v1.0 |
| Security E2E coverage | Partial | Before v1.0 |
| HashiCorp Vault integration | Later | v9 / production |
| Automated backup and restore drills | Shipped | v8.5 |

## Observability and operations

| Improvement | Status | Target |
|---|---|---|
| Prometheus metrics | Shipped | Current |
| Provisioned Grafana dashboard | Shipped | Current |
| Prometheus alert rules | Shipped | Current |
| Request correlation IDs | Shipped | Current |
| OpenTelemetry distributed tracing | Shipped | v8.4 |
| Automated staging/production CD | Later | v9 |
| Container image scanning | Shipped | Current |
| Signed remote SDR configuration | Shipped | v9.1 |
| IQ forensic capture and playback | Partial | v9 |

## UX and documentation

| Improvement | Status | Target |
|---|---|---|
| EN/ES/FR/DE i18n | Shipped | Current |
| Light/dark mode | Shipped | Current |
| Accessibility work | Partial | Before v1.0 |
| Header search | Shipped | v8.4 |
| Search across satellites/assets/CVEs/incidents | Shipped | v8.4 |
| OpenAPI and Swagger | Shipped | Current |
| Refactored technical documentation | Shipped | Current |
| Updated README for auth, realtime, and languages | Shipped | Current |
| TLS and secrets deployment guide | Shipped | Current |
| User administration and role guide | Shipped | Current |
| Connectivity/device support guide | Next | v8.3 |
| CVE user guide | Next | v8.3 |
| RF heat-map and chart documentation | Later | With feature |
| Complete user guide PDF | Later | Before v1.0 |
| Changelog maintenance | Shipped | Ongoing |

## Enterprise architecture

| Improvement | Status | Target |
|---|---|---|
| Executive/compliance reports | Later | Enterprise |
| Licensing | Later | Enterprise |
| Multi-tenancy | Later | Enterprise |
| Shared `rf-lib` | Later | Post-v1 |
| Shared `tle-lib` | Later | Post-v1 |
| Shared UI component library | Later | Post-v1 |
| Fleet policy and staged agent rollouts | Next | v9.x |
| Agent hardware capability validation | Next | v9.x |
| Stronger remote-config signing-key custody | Next | Before v1.0 |

| Agent quarantine and irreversible certificate revocation | Shipped | v9.0 |
| Signed remote SDR configuration | Shipped | v9.1 |
| Orbital-aware RF correlation | Shipped | v9.2 |

## Current pre-v1 priorities

1. Validate v9.2/v9.2.2 Doppler and visibility behavior with representative physical SDR captures before enabling automatic orbital escalation.
2. Extend deterministic correlation across asset alerts, orbital events, agent posture, and audit evidence.
3. Add maneuver/conjunction context and improve orbital confidence handling.
4. Complete sensitive-endpoint authorization review, browser token/CSRF review, HA guidance, and production-like query/load validation.
5. Complete fleet policy, staged rollouts, hardware capability validation, and stronger signing-key custody.
6. Finish operator-facing RF/telemetry/CVE/operations visualizations and connectivity/CVE guides.

## v9.1 delivered

- Signed declarative SDR remote configuration
- Immutable configuration history and acknowledgements
- Runtime frequency and capture-interval updates
- Safe rollback through a new signed version
- No generic remote command execution

Next agent-fleet work should focus on staged rollout, policy constraints, hardware capability validation, and signing-key custody rather than expanding the payload into arbitrary commands.
