# Documentation

This directory contains the technical documentation for Lyrahcle. The root `README.md` stays intentionally short. The backend embeds the OpenAPI specification at `/openapi.yaml` and serves Swagger UI at `/docs`.

## Start here

| Document | Canonical purpose |
|---|---|
| [Architecture](./architecture.md) | Components, trust boundaries, data flows, and storage ownership |
| [Backend bootstrap architecture](./backend-bootstrap-architecture.md) | Composition root, dependency builders and incremental server-domain refactor boundary |
| [Frontend TypeScript migration](./frontend-typescript-migration.md) | Gradual TS/TSX migration boundary, typed routing/navigation and next component candidates |
| [Frontend UX foundation](./frontend-ux-foundation.md) | Shared UI primitives for transient feedback, async state, and keyboard navigation |
| [API reference](./api-reference.md) | Human/agent authentication, roles, endpoint behavior, SSE, request IDs |
| [Configuration](./configuration.md) | Environment variables and runtime configuration for every service |
| [Deployment](./deployment.md) | Production checklist, TLS, backups, monitoring, updates, WebAuthn, agent mTLS |
| [Kubernetes deployment](./kubernetes-deployment.md) | Cloud-neutral Helm deployment, HA controls, probes, secrets, HPA/PDB, multi-replica safety |
| [Kubernetes staging and observability](./kubernetes-observability.md) | kind smoke environment, ServiceMonitor/PrometheusRule integration, metrics NetworkPolicy path |
| [AWS EKS migration](./aws-eks-migration.md) | EKS mapping for ECR/RDS/Redis, ALB/NLB, Secrets Manager and Pod Identity |
| [AWS infrastructure as code](./aws-infrastructure-as-code.md) | Terraform/OpenTofu foundation for VPC, EKS, ECR, RDS, KMS, Secrets Manager and optional encrypted ElastiCache |
| [AWS production readiness](./aws-production-readiness.md) | Route53/ACM, ExternalDNS, immutable ECR image digests and release handoff |
| [AWS Secrets Manager](./aws-secrets-manager.md) | Synchronizing application secrets from AWS Secrets Manager into Kubernetes |
| [Improvement roadmap](./improvement-roadmap.md) | Canonical shipped/partial/next/later product backlog |

## Identity, access, and RBAC

| Document | Canonical purpose |
|---|---|
| [User administration](./user-administration.md) | Roles, bootstrap admin, account lifecycle, TOTP, WebAuthn, recovery |
| [Security keys and recovery](./security-keys-and-recovery.md) | WebAuthn and password-recovery trust model |
| [Browser security guide](./01-browser-security-guide.md) | Bearer-token model, CSRF assessment, CSP, and CORS for browser API requests |
| [Authorization and RBAC guide](./02-rbac-guide.md) | Organization-scoped capabilities, endpoint matrix, and detection-rule/operational RBAC |
| [Platform administrator boundary](./platform-admin-boundary.md) | Control-plane organization management vs. tenant-local admin authority |
| [Administrative audit coverage](./admin-audit-coverage.md) | Which security-sensitive admin mutations are audited, and where |
| [Sensitive endpoint audit](./sensitive-endpoint-audit.md) | Security-sensitive endpoint inventory and audit expectations |
| [Signing key management](./key-management.md) | `KeyProvider` boundary, local Ed25519 provider, and key rotation |
| [Secret management](./secret-management.md) | Secret provider boundary (`env`/`file`/production requirements) |

## Multi-tenancy and platform isolation

| Document | Canonical purpose |
|---|---|
| [Multi-tenancy and organization isolation guide](./03-multi-tenancy-guide.md) | v10.x tenant-scoping rollout, PostgreSQL row-level security, and tenant-boundary/runtime-integrity guarantees |

## SDR fleet, edge, and orbital operations

| Document | Canonical purpose |
|---|---|
| [SDR agent guide](./04-sdr-agent-guide.md) | Enrollment, mTLS, certificate rotation, signed remote configuration, and hardware-capability reporting |
| [Raspberry Pi plug-and-play station](./raspberry-pi-station.md) | Reproducible ARM64 install/image profile, enrollment, diagnostics, pass configuration, signed updates, and hardware acceptance |
| [Edge node integrity / tamper detection](./edge-integrity.md) | Deterministic file-manifest baselining and healthy/changed classification |
| [Orbital-aware security correlation](./orbital-security-correlation.md) | v9.2/v9.2.2 orbital/RF evidence, severity gating, operator workflow |
| [Orbital operations context](./orbital-operations-context.md) | v9.5 maneuver/conjunction persistence, trust model, and candidate-only correlation |
| [Cross-domain correlation](./cross-domain-correlation.md) | v9.4 explainable incident correlation across RF, orbital, agent posture, audit, and context-only asset alerts |
| [Correlation constellation](./correlation-constellation.md) | Interactive orbital-focus graph for satellite, station, RF, incident, asset and CVE investigation paths |
| [Evidence provenance](./evidence-provenance.md) | Trust labels, confidence and the boundary between context, inference and confirmed exposure |
| [CVE intelligence synchronization](./cve-intelligence.md) | NVD pagination, incremental cursors, enriched CVE metadata, and applicability limits |
| [Real satellite sources](./satellite-real-sources.md) | NORAD-linked SatNOGS metadata, transmitters, observations, and source freshness |
| [Satellite pass prediction](./pass-prediction.md) | v9.3 AOS/LOS, peak elevation, prediction windows, and TLE safety |
| [TLE data quality](./tle-data-quality.md) | Fresh/aging/stale/unusable policy and safe orbital-escalation defaults |
| [IQ forensics](./forensics.md) | Recording, replaying, and handling SDR IQ captures |
| [Core telemetry visualizations](./telemetry-visualizations.md) | RF power/spectrum/coverage and anomaly-intensity UI panels |
| [Ground station operations](./ground-station-operations.md) | Ground-segment view over SDR agents with trusted station coordinates |
| [Risk scoring](./risk-scoring.md) | Deterministic mission/satellite/ground-station risk scores and inputs |

## Application security contracts

| Document | Canonical purpose |
|---|---|
| [Input/output security contracts](./input-output-security.md) | SQL parameterization and browser-XSS regression contracts kept in CI |
| [Runtime command-execution security contract](./command-execution-security.md) | Prohibition on runtime OS command execution outside dedicated review |
| [UI HTTP error handling contract](./ui-http-errors.md) | Typed frontend handling of 401/403/404/409 instead of generic errors |
| [Background job reliability](./background-job-reliability.md) | At-least-once delivery with idempotent durable effects for async work |
| [STRIDE threat model](./ieee3536-threat-model.md) | Assets, trust boundaries, STRIDE threats, mitigations, residual risk |
| [SDR hardware supply-chain threat model](./sdr-hardware-supply-chain-threat-model.md) | STRIDE analysis and acceptance gates for SDR hardware, firmware, FPGA, USB, and edge dependencies |
| [Software supply chain](./supply-chain.md) | SBOMs, dependency gates, signed releases, provenance |

## Operations

| Document | Canonical purpose |
|---|---|
| [Observability](./observability.md) | Prometheus, Grafana, request IDs, alerts, and optional OpenTelemetry tracing |
| [Backend reliability](./backend-reliability.md) | Error-chain conventions and reliability practices in the Go backend |
| [Performance testing](./performance-testing.md) | Current performance controls and production-like validation work |
| [Backup and restore](./backup-and-restore.md) | Backup verification, guarded restore, and recovery recommendations |
| [Resilience and disaster recovery](./resilience-disaster-recovery.md) | Local kind chaos drills, restore evidence, RTO/RPO targets and AWS DR boundaries |
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
