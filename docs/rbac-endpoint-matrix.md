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
