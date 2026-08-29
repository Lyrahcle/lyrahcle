# Improvement Roadmap

This roadmap tracks the current product and security backlog after v8.5.

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
| Cross-domain correlation | Next | v8.3-v9 |
| RF heat map | Next | v9 |
| Telemetry, RF, CVE, and operations charts | Next | v8.3-v9 |
| Orbital-aware RF detection | Next | v9 |
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
| Remote SDR configuration | Later | v9 |
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

| Agent quarantine and irreversible certificate revocation | Shipped | v9.0 |
| Signed remote SDR configuration | Next | v9.1 |
| Orbital-aware RF correlation | Next | v9.2 |
