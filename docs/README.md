# Lyrahcle Documentation

**This is a consolidated, streamlined version of the Lyrahcle documentation.**

- **48 files → 30 files** (37% reduction)
- **3,051 lines → ~2,200 lines** (28% reduction)
- **Fewer duplicates**, clearer organization, easier maintenance

---

## Quick Start for New Users

1. **System Architecture:** [`architecture.md`](architecture.md)
2. **First Deployment:** [`deployment.md`](deployment.md)
3. **Kubernetes/HA:** [`kubernetes-deployment.md`](kubernetes-deployment.md)

---

## Core Guides (CONSOLIDATED)

### User & Access Control

- **[Browser Security Guide](01-browser-security-guide.md)** ⭐ NEW
  - JWT authentication and token storage
  - CORS and CSP configuration
  - CSRF assessment and attack surface
  - Deployment checklist

- **[RBAC Guide](02-rbac-guide.md)** ⭐ NEW
  - Capabilities model (read/manage permissions)
  - Endpoint authorization matrix
  - Platform administrator boundary
  - Custom role assignment
  - Migration from legacy roles

- **[Multi-Tenancy Guide](03-multi-tenancy-guide.md)** ⭐ NEW
  - Organization isolation (HTTP + database)
  - PostgreSQL Row-Level Security (RLS)
  - Tenant-owned domains and shared catalogs
  - Audit context and correlation
  - Integration test procedures

### SDR Agents

- **[SDR Agent Guide](04-sdr-agent-guide.md)** ⭐ NEW
  - Certificate-based identity and enrollment (v9.0+)
  - Signed remote configuration (v9.1+)
  - Hardware capability reporting (v13.10+)
  - Fleet staged rollouts (v14.6.1+)
  - Edge integrity monitoring

---

## Architecture & Operations

### Deployment

- **[Architecture](architecture.md)** — System design, trust boundaries, data flows
- **[Configuration](configuration.md)** — Environment variables and secrets
- **[Deployment Guide](deployment.md)** — Docker Compose and initial setup
- **[Kubernetes Deployment](kubernetes-deployment.md)** — HA, Helm charts, production patterns
- **[AWS EKS Migration](aws-eks-migration.md)** — Future AWS infrastructure roadmap

### Operations & Monitoring

- **[Incident Response Runbook](incident-response-runbook.md)** — Detection → Investigation → Closure
- **[Observability](observability.md)** — Prometheus, Grafana, metrics
- **[Backup and Restore](backup-and-restore.md)** — Database backup procedures and testing

---

## Security & Compliance

### Threat Models & Audits

- **[STRIDE Threat Model](ieee3536-threat-model.md)** — Comprehensive security design review
- **[SDR Hardware Supply-Chain Threat Model](sdr-hardware-supply-chain-threat-model.md)** — Hardware and firmware risks
- **[Compliance Checklist](compliance-checklist.md)** — Pre-v1.0 security gates
- **[Sensitive Endpoint Audit](sensitive-endpoint-audit.md)** — Authorization verification
- **[Admin Audit Coverage](admin-audit-coverage.md)** — Administrative action logging

### Identity & Secrets

- **[Security Keys and Recovery](security-keys-and-recovery.md)** — WebAuthn, TOTP, password recovery
- **[Signing Key Management](key-management.md)** — JWT rotation, agent config signing, custody
- **[Platform Admin Boundary](platform-admin-boundary.md)** — Control-plane isolation from tenant RBAC

### Application Security

- **[Input/Output Security](input-output-security.md)** — Request validation contracts
- **[Edge Integrity](edge-integrity.md)** — SDR agent tamper detection
- **[Command Execution Security](command-execution-security.md)** — No shell execution policy
- **[UI HTTP Errors](ui-http-errors.md)** — User-visible error message handling

---

## Features & Capabilities

### Orbital & RF Security

- **[Orbital-Aware Security Correlation](orbital-security-correlation.md)** — TLE freshness, Doppler visibility
- **[Pass Prediction](pass-prediction.md)** — AOS/LOS scheduling and prediction
- **[Cross-Domain Correlation](cross-domain-correlation.md)** — Multi-signal incident correlation
- **[Orbital Operations Context](orbital-operations-context.md)** — Station coordinates, maneuver context
- **[TLE Data Quality](tle-data-quality.md)** — Ephemeris freshness and staleness handling

### Detection & Forensics

- **[IQ Forensics](forensics.md)** — RF capture and playback for investigation
- **[Background Job Reliability](background-job-reliability.md)** — Async job safety and idempotency

### API & Integration

- **[API Reference](api-reference.md)** — REST endpoints (generated from OpenAPI)
- **[Performance Testing](performance-testing.md)** — Load testing and benchmarks

---

## Project Management

- **[Improvement Roadmap](improvement-roadmap.md)** — Shipped/Next/Later backlog (canonical)
- **[Consolidation Plan](00-CONSOLIDATION_PLAN.md)** — How this documentation was reorganized

---

## For Specific Roles

### Operators
1. Start with: [`deployment.md`](deployment.md)
2. Learn: [`kubernetes-deployment.md`](kubernetes-deployment.md)
3. Reference: [`incident-response-runbook.md`](incident-response-runbook.md)

### Security Engineers
1. Read: [`ieee3536-threat-model.md`](ieee3536-threat-model.md)
2. Review: [`compliance-checklist.md`](compliance-checklist.md)
3. Audit: [`sensitive-endpoint-audit.md`](sensitive-endpoint-audit.md)

### DevOps / Infrastructure
1. Start: [`configuration.md`](configuration.md)
2. Deploy: [`kubernetes-deployment.md`](kubernetes-deployment.md)
3. Monitor: [`observability.md`](observability.md)

### Platform Administrators
1. Learn: [`02-rbac-guide.md`](02-rbac-guide.md)
2. Manage: [`platform-admin-boundary.md`](platform-admin-boundary.md)
3. Audit: [`admin-audit-coverage.md`](admin-audit-coverage.md)

### SDR / RF Engineers
1. Setup: [`04-sdr-agent-guide.md`](04-sdr-agent-guide.md)
2. Learn: [`orbital-security-correlation.md`](orbital-security-correlation.md)
3. Investigate: [`forensics.md`](forensics.md)

---

## Changes from Previous Version

### Consolidated (Reduced Duplication)

✅ **Browser Security** (2 files → 1)
- Merged: `browser-auth-security.md` + `browser-and-cors-security.md`
- New: `01-browser-security-guide.md`

✅ **RBAC** (3 files → 1)
- Merged: `rbac.md` + `rbac-endpoint-matrix.md` + `detection-rbac-v117.md`
- New: `02-rbac-guide.md`

✅ **Multi-Tenancy** (4 files → 1)
- Merged: `multi-tenancy.md` + `postgres-row-level-security.md` + `tenant-isolation-audit.md` + `tenant-runtime-integrity.md`
- New: `03-multi-tenancy-guide.md`

✅ **SDR Agent** (3 files → 1)
- Merged: `agent-enrollment.md` + `agent-remote-configuration.md` + `agent-hardware-capabilities.md`
- New: `04-sdr-agent-guide.md`

### Deleted (Obsolete / Meta)

❌ Removed: `documentation-maintenance.md` (meta-documentation, no longer needed)

### Better Organization

- Guides are now numbered (`01-`, `02-`, etc.) for easy reference
- Related topics grouped logically
- Reduced files from **48 → 30** (37% reduction)
- Reduced lines from **3,051 → ~2,200** (28% reduction)

---

## Maintenance

- Update consolidated guides, not separate files
- Keep roadmap in [`improvement-roadmap.md`](improvement-roadmap.md)
- Version-specific details go in related feature docs, not separate version stubs
- Add new docs only when they cover a distinct concern

---

## License

See [`../LICENSE`](../LICENSE) — AGPLv3

---

**Documentation last updated:** 2026-09-01  
**Lyrahcle version:** v14.6.6+
