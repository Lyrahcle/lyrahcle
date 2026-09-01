# Documentation

This directory contains the technical documentation for Lyrahcle. The root `README.md` stays intentionally short. The backend embeds the OpenAPI specification at `/openapi.yaml` and serves Swagger UI at `/docs`.

## Start here

| Document | Canonical purpose |
|---|---|
| [Architecture](./architecture.md) | Components, trust boundaries, data flows, and storage ownership |
| [API reference](./api-reference.md) | Human/agent authentication, roles, endpoint behavior, SSE, request IDs |
| [Configuration](./configuration.md) | Environment variables and runtime configuration for every service |
| [Deployment](./deployment.md) | Production checklist, TLS, backups, monitoring, updates, WebAuthn, agent mTLS |
| [Kubernetes deployment](./kubernetes-deployment.md) | Cloud-neutral Helm deployment, HA controls, probes, secrets, HPA/PDB, multi-replica safety |
| [Kubernetes staging and observability](./kubernetes-observability.md) | kind smoke environment, ServiceMonitor/PrometheusRule integration, metrics NetworkPolicy path |
| [AWS EKS migration](./aws-eks-migration.md) | Future EKS mapping for ECR/RDS/Redis, ALB/NLB, Secrets Manager and Pod Identity |
| [Improvement roadmap](./improvement-roadmap.md) | Canonical shipped/partial/next/later product backlog |
| [Documentation maintenance](./documentation-maintenance.md) | Source-of-truth hierarchy and consistency rules |

## Identity, access, and RBAC

| Document | Canonical purpose |
|---|---|
| [User administration](./user-administration.md) | Roles, bootstrap admin, account lifecycle, TOTP, WebAuthn, recovery |
| [Security keys and recovery](./security-keys-and-recovery.md) | WebAuthn and password-recovery trust model |
| [Browser authentication security](./browser-auth-security.md) | Bearer-token model and CSRF assessment for browser API requests |
| [Granular RBAC](./rbac.md) | v11.x organization-scoped permissions, custom roles, and capability enforcement |
| [RBAC endpoint matrix](./rbac-endpoint-matrix.md) | Per-domain read/manage capability mapping to representative routes |
| [v11.7 detection and operational RBAC completion](./detection-rbac-v117.md) | Tenant-local detection-rule policy and remaining capability migrations |
| [Platform administrator boundary](./platform-admin-boundary.md) | Control-plane organization management vs. tenant-local admin authority |
| [Administrative audit coverage](./admin-audit-coverage.md) | Which security-sensitive admin mutations are audited, and where |
| [Sensitive endpoint audit](./sensitive-endpoint-audit.md) | Security-sensitive endpoint inventory and audit expectations |
| [Signing key management](./key-management.md) | `KeyProvider` boundary, local Ed25519 provider, and key rotation |
| [Secret management](./secret-management.md) | Secret provider boundary (`env`/`file`/production requirements) |

## Multi-tenancy and platform isolation

| Document | Canonical purpose |
|---|---|
| [Organizations and multi-tenancy](./multi-tenancy.md) | v10.x tenant-scoping rollout across operational domains |
| [PostgreSQL row-level security](./postgres-row-level-security.md) | v14.2 database-enforced tenant isolation via `dbscope` and RLS |
| [Tenant isolation audit](./tenant-isolation-audit.md) | v10.6 release gate: tenant-owned domains and cross-tenant guarantees |
| [Tenant runtime integrity](./tenant-runtime-integrity.md) | Preserving `organization_id` end-to-end into incidents and audit records |

## SDR fleet, edge, and orbital operations

| Document | Canonical purpose |
|---|---|
| [SDR agent enrollment and mTLS](./agent-enrollment.md) | Enrollment, certificate rotation, quarantine, revocation, legacy-key migration |
| [Signed remote agent configuration](./agent-remote-configuration.md) | v9.1 immutable signed configuration and rollback |
| [Agent hardware capabilities](./agent-hardware-capabilities.md) | v13.10 SDR hardware/firmware inventory reporting and compatibility labels |
| [Edge node integrity / tamper detection](./edge-integrity.md) | Deterministic file-manifest baselining and healthy/changed classification |
| [Orbital-aware security correlation](./orbital-security-correlation.md) | v9.2/v9.2.2 orbital/RF evidence, severity gating, operator workflow |
| [Orbital operations context](./orbital-operations-context.md) | v9.5 maneuver/conjunction persistence, trust model, and candidate-only correlation |
| [Cross-domain correlation](./cross-domain-correlation.md) | v9.4 explainable incident correlation across RF, orbital, agent posture, audit, and context-only asset alerts |
| [Satellite pass prediction](./pass-prediction.md) | v9.3 AOS/LOS, peak elevation, prediction windows, and TLE safety |
| [TLE data quality](./tle-data-quality.md) | Fresh/aging/stale/unusable policy and safe orbital-escalation defaults |
| [IQ forensics](./forensics.md) | Recording, replaying, and handling SDR IQ captures |
| [Core telemetry visualizations](./telemetry-visualizations.md) | RF power/spectrum/coverage and anomaly-intensity UI panels |

## Application security contracts

| Document | Canonical purpose |
|---|---|
| [Input/output security contracts](./input-output-security.md) | SQL parameterization and browser-XSS regression contracts kept in CI |
| [Runtime command-execution security contract](./command-execution-security.md) | Prohibition on runtime OS command execution outside dedicated review |
| [Browser, CSP, and CORS security](./browser-and-cors-security.md) | Same-origin default, CSP, and exact-allowlist CORS for split origins |
| [UI HTTP error handling contract](./ui-http-errors.md) | Typed frontend handling of 401/403/404/409 instead of generic errors |
| [Background job reliability](./background-job-reliability.md) | At-least-once delivery with idempotent durable effects for async work |
| [STRIDE threat model](./ieee3536-threat-model.md) | Assets, trust boundaries, STRIDE threats, mitigations, residual risk |
| [SDR hardware supply-chain threat model](./sdr-hardware-supply-chain-threat-model.md) | STRIDE analysis and acceptance gates for SDR hardware, firmware, FPGA, USB, and edge dependencies |
| [Software supply chain](./supply-chain.md) | SBOMs, dependency gates, signed releases, provenance |

## Operations

| Document | Canonical purpose |
|---|---|
| [Observability](./observability.md) | Prometheus, Grafana, request IDs, alerts, and optional OpenTelemetry tracing |
| [Performance testing](./performance-testing.md) | Current performance controls and production-like validation work |
| [Backup and restore](./backup-and-restore.md) | Backup verification, guarded restore, and recovery recommendations |
| [Incident response runbook](./incident-response-runbook.md) | Containment, key/agent compromise, tenant breach, evidence, and recovery procedures |
| [Compliance checklist](./compliance-checklist.md) | Pre-v1 security evidence and readiness tracker |

## Process and governance

| Document | Canonical purpose |
|---|---|
| [Security policy](../SECURITY.md) | Private vulnerability reporting and supported scope |
| [Security roadmap](../SECURITY_ROADMAP.md) | Security controls, evidence, and remaining v1.0 security gates |
| [Development plan](../PLAN.md) | Execution order and milestone intent |
| [Contributing](../CONTRIBUTING.md) | Local development, testing, conventions, PR expectations |
| [Releasing](../RELEASING.md) | Versioning and GHCR release process |
| [Changelog](../CHANGELOG.md) | Historical milestone/release record |

## Documentation status model

- `docs/improvement-roadmap.md` is the canonical product backlog.
- `SECURITY_ROADMAP.md` is the canonical security-control/gate tracker.
- `PLAN.md` explains execution order; it must not duplicate a second backlog.
- `CHANGELOG.md` is historical and must not be interpreted as current status.
- `apps/backend/cmd/server/openapi.yaml` is the canonical machine-readable HTTP API description; endpoint descriptions must track the route/authentication implementation.
- `docs/configuration.md` is the canonical human-readable configuration reference; code defaults and production validation remain authoritative if a discrepancy is discovered.

Every file added under `docs/` must be linked from one section above in the same change — this index is the discovery path for the rest of the documentation set, and an unlinked doc is effectively lost.
