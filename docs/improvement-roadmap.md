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
| CSRF protection if cookie-based authentication is introduced | Shipped | Current (N/A today: bearer-token-only auth, no cookies; `check-csrf-cookie-auth.py` runs in local v1 validation and GitHub Actions, failing the build the moment a handler sets one without CSRF protection) |
| Automatic agent certificate rotation | Shipped | v9.0 |
| Agent enrollment and mTLS | Shipped | v9.0 |
| Additional roles such as analyst and auditor | Shipped | Current |
| Administrator subroles | Shipped | v9 |
| Explicit permission matrix | Shipped | Current |
| Disable and soft-delete users | Shipped | Current |
| Complete sensitive-endpoint authorization review | Shipped | Current (`check-route-authorization.py` audits every route in `cmd/server/*.go` in local v1 validation and GitHub Actions, failing the build if a new route lacks a recognizable role/permission/mTLS/token gate) |
| Complete administrative audit-coverage review | Partial | Before v1.0 |
| Multi-tenant organizations with tenant-scoped RBAC | Shipped | v10.0-v10.6 |
| Database-enforced tenant isolation (PostgreSQL row-level security) | Shipped | v14.2 |
| Platform-administrator control-plane boundary | Shipped | v10.x |
| Granular per-organization RBAC with custom roles and capability enforcement | Shipped | v11.0-v11.1 |
| Detection-rule and operational RBAC capability migration | Shipped | v11.7 |
| Signing-key management abstraction (`KeyProvider`) | Shipped | Current |
| Secret-provider boundary (`env`/`file`, production hardening) | Shipped | Current |

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
| Response/orbital/CVE caching | Partial | v8.3-v9 (now also covers incidents, asset-alerts, RF readings/anomalies; response caching for other high-traffic reads still open) |
| Satellite-load optimization | Shipped | v8.5 |
| Index review for all filtered searches | Partial | v8.5-v9 |
| SQL query optimization | Partial | v8.5-v9 |
| Load-test tooling | Shipped | v8.4 |
| Station federation | Later | v9+ |
| Multi-tenant architecture | Shipped | v10.x (see Identity, authentication, and authorization) |

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
| Shared `rf-lib` | Later | Post-v1 |
| Shared `tle-lib` | Later | Post-v1 |
| Shared UI component library | Later | Post-v1 |
| Fleet policy and staged agent rollouts | Next | Before v1.0 |
| Agent hardware capability reporting | Shipped | v13.10 |
| Agent hardware capability physical lab-validation gate | Next | Before v1.0 |
| Stronger remote-config signing-key custody | Next | Before v1.0 |

Note: multi-tenancy moved from "Enterprise architecture / Later" to the core Identity, authentication, and authorization table once it shipped in v10.0; it is core, not an enterprise-only module.

## Current pre-v1 priorities

1. Validate orbital-aware Doppler/visibility correlation with representative physical SDR captures before enabling automatic orbital escalation (currently disabled).
2. Reach and hold >80% coverage on security-critical code paths, including security-focused end-to-end coverage.
3. Publish HA/DR and production-load guidance, and complete fleet policy plus staged agent rollouts.
4. Complete the agent hardware-capability physical lab-validation gate (RTL-SDR/HackRF) — `compatible` currently reflects self-reported software capability only.
5. Ship the RF heat map and finish remaining operator-facing user guides.

Per-release delivery detail (what shipped in v9.1, v10.x, v11.x, and later) lives in `CHANGELOG.md`, not here, to avoid a second history that can drift out of sync with this backlog.
