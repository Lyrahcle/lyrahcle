# Organizations and multi-tenancy

## v10.0 — foundation
Organizations, default-organization migration, user membership and JWT organization context.

## v10.1 — user administration isolation
User list/create/role/status/delete operations are restricted to the authenticated organization and last-admin protection is tenant-local.

## v10.2 — critical-domain tenant boundary

The database now carries mandatory `organization_id` ownership for:

- agents
- assets
- incidents
- audit events

Existing rows are backfilled into the stable default organization. Foreign keys and tenant-first indexes make the ownership boundary explicit.

A reusable tenant-scope store performs organization-keyed existence/count queries, and the admin-only `/api/v1/organization-scope/summary` endpoint provides a safe tenant-local diagnostic view.

### Important isolation gate

v10.2 establishes the schema and query primitives; it does **not** yet claim that every legacy agent/asset/incident/audit handler has been migrated. Those handlers must be converted to organization-scoped store methods before production multi-tenancy is enabled.

## Next

v10.3 migrates the actual critical-domain read/write handlers and adds negative Org-A/Org-B integration tests. RF, orbital, programs, CVE relationships, background jobs, search and SSE follow after that gate.
