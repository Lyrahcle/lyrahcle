# Documentation

This directory contains the technical documentation for Lyrahcle. The root `README.md` stays intentionally short.

| Document | Canonical purpose |
|---|---|
| [Architecture](./architecture.md) | Components, trust boundaries, data flows, and storage ownership |
| [API reference](./api-reference.md) | Human/agent authentication, roles, endpoint behavior, SSE, request IDs |
| [Configuration](./configuration.md) | Environment variables and runtime configuration for every service |
| [Deployment](./deployment.md) | Production checklist, TLS, backups, monitoring, updates, WebAuthn, agent mTLS |
| [Observability](./observability.md) | Prometheus, Grafana, request IDs, alerts, and optional OpenTelemetry tracing |
| [Performance testing](./performance-testing.md) | Current performance controls and production-like validation work |
| [User administration](./user-administration.md) | Roles, bootstrap admin, account lifecycle, TOTP, WebAuthn, recovery |
| [Security keys and recovery](./security-keys-and-recovery.md) | WebAuthn and password-recovery trust model |
| [SDR agent enrollment and mTLS](./agent-enrollment.md) | Enrollment, certificate rotation, quarantine, revocation, legacy-key migration |
| [Signed remote agent configuration](./agent-remote-configuration.md) | v9.1 immutable signed configuration and rollback |
| [Orbital-aware security correlation](./orbital-security-correlation.md) | v9.2/v9.2.2 orbital/RF evidence, severity gating, operator workflow |
| [TLE data quality](./tle-data-quality.md) | Fresh/aging/stale/unusable policy and safe orbital-escalation defaults |
| [IQ forensics](./forensics.md) | Recording, replaying, and handling SDR IQ captures |
| [Backup and restore](./backup-and-restore.md) | Backup verification, guarded restore, and recovery recommendations |
| [Sensitive endpoint audit](./sensitive-endpoint-audit.md) | Security-sensitive endpoint inventory and audit expectations |
| [STRIDE threat model](./ieee3536-threat-model.md) | Assets, trust boundaries, STRIDE threats, mitigations, residual risk |
| [Compliance checklist](./compliance-checklist.md) | Pre-v1 security evidence and readiness tracker |
| [Software supply chain](./supply-chain.md) | SBOMs, dependency gates, signed releases, provenance |
| [Improvement roadmap](./improvement-roadmap.md) | Canonical shipped/partial/next/later product backlog |
| [Documentation maintenance](./documentation-maintenance.md) | Source-of-truth hierarchy and consistency rules |
| [Security policy](../SECURITY.md) | Private vulnerability reporting and supported scope |
| [Security roadmap](../SECURITY_ROADMAP.md) | Security controls, evidence, and remaining v1.0 security gates |
| [Development plan](../PLAN.md) | Execution order and milestone intent |
| [Contributing](../CONTRIBUTING.md) | Local development, testing, conventions, PR expectations |
| [Releasing](../RELEASING.md) | Versioning and GHCR release process |
| [Changelog](../CHANGELOG.md) | Historical milestone/release record |

The backend embeds the OpenAPI specification at `/openapi.yaml` and serves Swagger UI at `/docs`.

## Documentation status model

- `docs/improvement-roadmap.md` is the canonical product backlog.
- `SECURITY_ROADMAP.md` is the canonical security-control/gate tracker.
- `PLAN.md` explains execution order; it must not duplicate a second backlog.
- `CHANGELOG.md` is historical and must not be interpreted as current status.
- `apps/backend/cmd/server/openapi.yaml` is the canonical machine-readable HTTP API description; endpoint descriptions must track the route/authentication implementation.
- `docs/configuration.md` is the canonical human-readable configuration reference; code defaults and production validation remain authoritative if a discrepancy is discovered.
