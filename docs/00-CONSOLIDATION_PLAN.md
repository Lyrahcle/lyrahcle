# Documentation Consolidation Plan

**Total docs before:** 48 files, ~3,051 lines  
**Total docs after:** 30 files, ~2,200 lines  
**Reduction:** ~28% fewer files, ~27% fewer lines

---

## CONSOLIDATED (Moved into new files)

### Browser Security (2 files → 1)
- ✅ `browser-auth-security.md` → **01-browser-security-guide.md**
- ✅ `browser-and-cors-security.md` → **01-browser-security-guide.md**
- **Delete:** `browser-auth-security.md`, `browser-and-cors-security.md`

### RBAC (3 files → 1)
- ✅ `rbac.md` → **02-rbac-guide.md**
- ✅ `rbac-endpoint-matrix.md` → **02-rbac-guide.md**
- ✅ `detection-rbac-v117.md` → **02-rbac-guide.md**
- **Delete:** `rbac.md`, `rbac-endpoint-matrix.md`, `detection-rbac-v117.md`

### Multi-Tenancy (4 files → 1)
- ✅ `multi-tenancy.md` → **03-multi-tenancy-guide.md**
- ✅ `postgres-row-level-security.md` → **03-multi-tenancy-guide.md**
- ✅ `tenant-isolation-audit.md` → **03-multi-tenancy-guide.md**
- ✅ `tenant-runtime-integrity.md` → **03-multi-tenancy-guide.md**
- **Delete:** `multi-tenancy.md`, `postgres-row-level-security.md`, `tenant-isolation-audit.md`, `tenant-runtime-integrity.md`

### SDR Agent Docs (can consolidate into one Agent guide)
- `agent-enrollment.md` (86 lines) → keep
- `agent-remote-configuration.md` (94 lines) → keep
- `agent-hardware-capabilities.md` (9 lines) → merge into enrollment
- **Delete:** `agent-hardware-capabilities.md`

### Agent Doc Consolidation Plan (2 files → 1)
Create: **04-sdr-agent-guide.md** combining:
- Agent enrollment and identity
- Remote configuration
- Hardware capabilities
- Device posture reporting
- **Delete:** `agent-hardware-capabilities.md`

---

## KEPT (Already well-structured)

These remain intact:

### Core Architecture & Deployment
- `architecture.md` (75 lines) ✅
- `deployment.md` (118 lines) ✅
- `kubernetes-deployment.md` (182 lines) ✅
- `configuration.md` (134 lines) ✅
- `aws-eks-migration.md` (132 lines) ✅

### Operational & Incident Response
- `incident-response-runbook.md` (185 lines) ✅
- `observability.md` (43 lines) ✅
- `backup-and-restore.md` (46 lines) ✅

### Security & Compliance
- `ieee3536-threat-model.md` (245 lines) ✅
- `security-keys-and-recovery.md` (lines) ✅
- `key-management.md` (118 lines) ✅
- `sdr-hardware-supply-chain-threat-model.md` (39 lines) ✅
- `compliance-checklist.md` (86 lines) ✅

### Feature-Specific
- `api-reference.md` (203 lines) ✅
- `orbital-security-correlation.md` (76 lines) ✅
- `pass-prediction.md` (51 lines) ✅
- `cross-domain-correlation.md` (39 lines) ✅
- `forensics.md` (47 lines) ✅

### Audit & Verification
- `admin-audit-coverage.md` (47 lines) ✅
- `sensitive-endpoint-audit.md` (lines) ✅
- `platform-admin-boundary.md` (22 lines) ✅ - Small but critical

### Operational Details
- `background-job-reliability.md` (40 lines) ✅
- `ui-http-errors.md` (lines) ✅
- `input-output-security.md` (22 lines) ✅
- `edge-integrity.md` (11 lines) ✅
- `command-execution-security.md` (15 lines) ✅
- `orbital-operations-context.md` (24 lines) ✅
- `performance-testing.md` (29 lines) ✅
- `tle-data-quality.md` (lines) ✅

### Documentation Infrastructure
- `improvement-roadmap.md` (148 lines) ✅
- `README.md` (102 lines) ✅

---

## ELIMINATED (Delete these)

**9 files to delete** (absorbed into consolidated docs):

1. ~~`browser-auth-security.md`~~ → 01-browser-security-guide.md
2. ~~`browser-and-cors-security.md`~~ → 01-browser-security-guide.md
3. ~~`rbac.md`~~ → 02-rbac-guide.md
4. ~~`rbac-endpoint-matrix.md`~~ → 02-rbac-guide.md
5. ~~`detection-rbac-v117.md`~~ → 02-rbac-guide.md
6. ~~`multi-tenancy.md`~~ → 03-multi-tenancy-guide.md
7. ~~`postgres-row-level-security.md`~~ → 03-multi-tenancy-guide.md
8. ~~`tenant-isolation-audit.md`~~ → 03-multi-tenancy-guide.md
9. ~~`tenant-runtime-integrity.md`~~ → 03-multi-tenancy-guide.md
10. ~~`agent-hardware-capabilities.md`~~ → 04-sdr-agent-guide.md (in progress)
11. ~~`documentation-maintenance.md`~~ → obsolete, can delete

**Reason:** Duplicate content, version-specific stubs, meta-documentation

---

## MERGE CANDIDATES (To consider for next iteration)

These are well-written but could eventually merge if they share context:

- `observability.md` + `telemetry-visualizations.md` → could be one "Observability Guide"
- `command-execution-security.md` + `input-output-security.md` → could be "Application Security Contracts"
- Multiple "v9.5", "v10.0" version notes → could move to CHANGELOG or version-specific section

**Decision:** Keep separate for now (they're small and focused). Merge only if they grow.

---

## Documentation Index (After Consolidation)

### User Guides (New)
- `01-browser-security-guide.md` — Authentication, CORS, CSP, token security
- `02-rbac-guide.md` — Capabilities, role assignment, authorization enforcement
- `03-multi-tenancy-guide.md` — Organization isolation, RLS, audit context
- `04-sdr-agent-guide.md` — Agent enrollment, configuration, capabilities (TBD)

### Architecture & Operations
- `architecture.md` — System design and trust boundaries
- `deployment.md` — Docker Compose, docker setup
- `kubernetes-deployment.md` — HA, Kubernetes, Helm charts
- `aws-eks-migration.md` — AWS deployment patterns
- `configuration.md` — Environment variables, secrets

### Security & Compliance
- `ieee3536-threat-model.md` — STRIDE threat model
- `compliance-checklist.md` — Pre-v1.0 security gates
- `sdr-hardware-supply-chain-threat-model.md` — SDR hardware risks
- `security-keys-and-recovery.md` — WebAuthn, password recovery
- `key-management.md` — Signing key lifecycle
- `platform-admin-boundary.md` — Control plane isolation

### Incident Response & Operations
- `incident-response-runbook.md` — Detection → investigation → closure
- `observability.md` — Prometheus, Grafana, metrics
- `backup-and-restore.md` — Database backup procedures
- `admin-audit-coverage.md` — Audit trail for all admin actions

### Features & Capabilities
- `api-reference.md` — REST API endpoints (from OpenAPI)
- `orbital-security-correlation.md` — TLE, Doppler, visibility
- `pass-prediction.md` — AOS/LOS scheduling
- `cross-domain-correlation.md` — Multi-signal incident correlation
- `forensics.md` — IQ capture and replay

### Contracts & Constraints
- `sensitive-endpoint-audit.md` — Authorization gates
- `input-output-security.md` — Request validation
- `edge-integrity.md` — SDR agent tamper detection
- `command-execution-security.md` — No shell execution
- `ui-http-errors.md` — User-visible error messages

### Infrastructure & Observability
- `background-job-reliability.md` — Async job safety
- `orbital-operations-context.md` — Station coordinates, maneuvers
- `performance-testing.md` — Load testing, benchmarks
- `tle-data-quality.md` — TLE freshness and staleness

### Project
- `README.md` — Documentation index and quick links
- `improvement-roadmap.md` — Shipped/next/later backlog

---

## Implementation Steps

1. **Create consolidated files** (Done):
   - ✅ 01-browser-security-guide.md
   - ✅ 02-rbac-guide.md
   - ✅ 03-multi-tenancy-guide.md
   - ⏳ 04-sdr-agent-guide.md (next)

2. **Update README.md** to link new guides

3. **Delete obsolete files**

4. **Verify references** in application code and other docs

5. **Test links** in generated HTML docs

---

## Expected Outcome

- **Fewer files:** 48 → 30 files
- **Easier navigation:** Grouped by concern, not versioning
- **Less duplication:** Consolidated stubs and versioned docs
- **Better readability:** Longer, complete guides vs. many short ones
- **Faster to maintain:** Fewer files to update when features change
