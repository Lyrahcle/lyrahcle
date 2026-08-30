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
| Orbital | `orbital.read` | `orbital.manage` | orbital event routes |
| Audit | `audit.read` | — | `/api/v1/audit-log` |
| Organizations | `organizations.read` | `organizations.manage` | `/api/v1/organizations/*` |
| RBAC | `rbac.read`/authenticated self view | `rbac.manage` | `/api/v1/rbac/*` |

Machine-agent routes remain credential-scoped and are not converted into human RBAC
checks. Static regression tests cover the sensitive human domains and explicitly reject
the historical `incidents.manage` guard on agent-key administration.
