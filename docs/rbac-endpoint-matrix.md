# RBAC endpoint matrix

Lyrahcle Core authorizes human API requests with organization-scoped capabilities.
Legacy role names remain compatibility metadata, but sensitive administration routes
must enforce capabilities rather than `RoleAdmin`/`RoleOperator` directly.

| Domain | Read capability | Manage capability | Representative routes |
| --- | --- | --- | --- |
| Users | `users.read` | `users.manage` | `/api/v1/users/*` |
| Agents | `agents.read` | `agents.manage` | `/api/v1/agents/*`, `/api/v1/agent-keys/*` |
| Assets | `assets.read` | `assets.manage` | `/api/v1/assets/*` |
| Programs | `programs.read` | `programs.manage` | `/api/v1/programs/*` |
| Incidents | `incidents.read` | `incidents.manage` | `/api/v1/incidents/*` |
| RF | `rf.read` | `rf.ingest` | `/api/v1/rf/*` |
| Detection rules | `detections.read` | `detections.manage` | `/api/v1/detection-rules/*` |
| Asset alerts | `alerts.read` | `alerts.manage` | `/api/v1/asset-alerts/*` |
| Orbital | `orbital.read` | `orbital.manage` | orbital event routes |
| Audit | `audit.read` | — | `/api/v1/audit-log` |
| Organization scope (tenant) | `organizations.read` | `organizations.manage` (reserved) | `/api/v1/organization-scope/*` |
| Organizations (global control plane) | platform administrator boundary | platform administrator boundary | `/api/v1/organizations/*` |
| RBAC | `rbac.read`/authenticated self view | `rbac.manage` | `/api/v1/rbac/*` |

Global organization administration is intentionally outside tenant-local RBAC: `/api/v1/organizations/*` uses the platform-administrator boundary. `organizations.read` currently protects the tenant scope summary; `organizations.manage` remains reserved for future tenant-local organization settings.

Machine-only self-service routes such as `/api/v1/agents/me/device` and certificate rotation remain credential-scoped. RF ingestion is intentionally hybrid: mTLS/agent-key traffic is authenticated as the agent, while human JWT traffic must hold `rf.ingest`; there is no `RoleOperator` fallback. Static regression tests cover the sensitive human domains and explicitly reject legacy role coupling.

## Pre-v1.0 authorization audit gate

The release audit distinguishes **authentication-only boundaries** from tenant-sensitive
capability checks. `apps/backend/cmd/server/authorization_audit_v140_test.go` is a CI
tripwire for this distinction:

- tenant-sensitive administration and analyst routes are inventoried with the capability
  that must guard them;
- direct legacy `admin`, `operator`, `analyst`, or `auditor` role middleware is rejected
  for server routes;
- the small set of `viewer`-only boundaries is counted explicitly. A new viewer-only
  boundary fails the test until it is reviewed and deliberately added to the audit.

Viewer-only boundaries are currently accepted only for authentication/self-service,
stateless TM/TC decoding, search/system metadata, read-only CVE intelligence, and
control-plane wrappers that perform a separate platform-administrator check. Machine
self-service remains credential-scoped as described above.

This is intentionally a release guard rather than a claim that route authorization can be
inferred from role names. New tenant-sensitive routes should use
`RequirePermissionWithFallback`, `RequireAnyPermissionWithFallback`, or an equivalent
explicit `requireRequestPermission` capability check and must update this matrix.

## PostgreSQL RLS defense in depth (v14.2)

Tenant authorization remains permission-driven at the HTTP layer, while PostgreSQL now independently enforces organization boundaries for core operational tables through `0040_postgres_row_level_security.sql`. HTTP database access without an authenticated organization is fail-closed. Connection-scoped RLS settings are reset before pool reuse to prevent tenant scope leakage between requests.
