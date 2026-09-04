# Authorization and RBAC Guide

## Overview

Lyrahcle Core uses granular, organization-scoped capabilities for authorization. Legacy role names remain in JWT tokens for compatibility, but sensitive endpoints now enforce capabilities rather than fixed role hierarchy.

## Capability Model

### v11.0 — Foundation
- Organization-scoped permissions
- Custom roles with assignable capabilities
- Per-organization role assignments
- Compatibility mappings to legacy roles

### v11.1 — Permission Enforcement
Sensitive HTTP operations now enforce explicit capabilities:

**Read capabilities:**
- `assets.read` — view assets and programs
- `incidents.read` — view incidents and correlations
- `agents.read` — view agents and device inventory
- `audit.read` — view audit logs
- `rf.read` — view RF readings and anomalies
- `orbital.read` — view orbital context
- `detections.read` — view detection rules
- `alerts.read` — view CVE asset alerts
- `users.read` — view user directory (self only for non-admin)
- `rbac.read` — view RBAC configuration (authenticated users can view own)

**Manage capabilities:**
- `assets.manage` — create/modify assets and programs
- `incidents.manage` — escalate, reassign, close incidents
- `agents.manage` — enroll, configure, revoke agents
- `audit.manage` — (reserved for future compliance features)
- `rf.ingest` — ingest RF readings (human + machine paths)
- `orbital.manage` — create/update stations and orbital context
- `detections.manage` — create/modify detection rules (admin only)
- `alerts.manage` — acknowledge CVE alerts
- `users.manage` — create/modify users, assign roles
- `rbac.manage` — create/modify roles and capabilities

The `/rbac/me` endpoint remains available to all authenticated users for frontend capability discovery.

## Endpoint Authorization Matrix

| Domain | Read Capability | Manage Capability | Representative Routes |
|--------|-----------------|-------------------|----------------------|
| Users | `users.read` | `users.manage` | `/api/v1/users/*` |
| Agents | `agents.read` | `agents.manage` | `/api/v1/agents/*`, `/api/v1/agent-keys/*` |
| Assets | `assets.read` | `assets.manage` | `/api/v1/assets/*` |
| Programs | `programs.read` | `programs.manage` | `/api/v1/programs/*` |
| Incidents | `incidents.read` | `incidents.manage` | `/api/v1/incidents/*` |
| RF | `rf.read` | `rf.ingest` | `/api/v1/rf/*` |
| Detection Rules | `detections.read` | `detections.manage` | `/api/v1/detection-rules/*` |
| Asset Alerts | `alerts.read` | `alerts.manage` | `/api/v1/asset-alerts/*` |
| Orbital | `orbital.read` | `orbital.manage` | Orbital event routes |
| Audit | `audit.read` | — | `/api/v1/audit-log` |
| Organization Scope | `organizations.read` | `organizations.manage` (reserved) | `/api/v1/organization-scope/*` |
| Organizations (Control Plane) | platform administrator boundary | platform administrator boundary | `/api/v1/organizations/*` |
| RBAC | `rbac.read`/authenticated self view | `rbac.manage` | `/api/v1/rbac/*` |

## Special Cases

### Machine-Only Self-Service Routes
- `/api/v1/agents/me/device` — agent device inventory (mTLS or agent-key credential)
- `/api/v1/agents/me/config` — agent configuration polling (mTLS or agent-key credential)
- Agent certificate rotation — credential-scoped, no human permission check

### Hybrid RF Ingest
RF ingestion is intentionally hybrid:
- mTLS/agent-key traffic is authenticated as the agent identity (machine)
- Human JWT traffic must hold `rf.ingest` capability
- There is no `RoleOperator` fallback

### Platform Administration Boundary
Global organization management is a control-plane capability and is outside tenant-local RBAC:

- `/api/v1/organizations/*` uses the **platform-administrator boundary**, not tenant-local permissions
- Tenant admins manage users, agents, incidents, assets, RBAC within their own organization only
- Tenant admins cannot list or create organizations merely because they hold the tenant `admin` role
- The `platform_administrators` list is the explicit control-plane allowlist

Upgrade behavior promotes only the oldest active admin in the default organization when no platform administrator exists. Fresh installations grant platform-administrator status to the initial seeded administrator.

## Pre-v1.0 Authorization Audit Gate

The release audit in `apps/backend/cmd/server/authorization_audit_test.go` is a CI tripwire for authorization consistency:

**Audit Requirements:**
- Tenant-sensitive administration and analyst routes are inventoried with the capability that must guard them
- Direct legacy `admin`, `operator`, `analyst`, or `auditor` role middleware is rejected for new server routes
- The small set of `viewer`-only boundaries is counted explicitly and reviewed

**Accepted Viewer-Only Boundaries:**
- Authentication and self-service endpoints
- Stateless TM/TC decoding
- Search and system metadata
- Read-only CVE intelligence
- Control-plane wrappers (separate platform-administrator check)
- Machine self-service (credential-scoped)

## Implementation Details

### Capability Enforcement

New tenant-sensitive routes must use one of:
- `RequirePermissionWithFallback`
- `RequireAnyPermissionWithFallback`
- Explicit `requireRequestPermission` capability check

All three must update the RBAC endpoint matrix above before merge.

### PostgreSQL Row-Level Security (v14.2)

Tenant authorization remains permission-driven at the HTTP layer. PostgreSQL independently enforces organization boundaries for core operational tables through row-level security (RLS):

- HTTP database access without an authenticated organization is **fail-closed**
- Connection-scoped RLS settings are reset before pool reuse to prevent tenant scope leakage between requests
- RLS provides defense-in-depth for database-level isolation

## Migrating Legacy Role Code

Do **not** use role-name checks in new code. Old patterns to avoid:

```go
// ❌ Bad
if user.Role == "admin" { ... }

// ✅ Good
if requester.HasCapability("users.manage") { ... }
```

## Checklist

- [ ] All new routes enforce explicit capabilities
- [ ] Authorization tests cover positive (allowed) and negative (denied) cases
- [ ] RBAC endpoint matrix updated for new domains
- [ ] Role-based legacy checks removed from new code
- [ ] Platform-admin and tenant-admin boundaries clear
- [ ] `authorization_audit_test.go` passes in CI
