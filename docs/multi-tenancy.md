# Organizations and multi-tenancy

## v10.0 — foundation
Organizations, default migration, user membership and JWT organization context.

## v10.1 — user administration isolation
Administrative user operations are restricted to the authenticated organization.

## v10.2 — critical-domain schema boundary
Organization ownership was added for agent identities, assets, incidents and audit.

## v10.3 — operational handler isolation

Production-facing handlers now enforce tenant scope for:

- asset listing and upsert;
- incident list, SSE stream, detail, update, notes and evidence;
- incident assignment to users in the same organization;
- audit-log listing.

Known UUIDs from another tenant are treated as not found by scoped incident/store operations. Asset uniqueness is tenant-local.

The v10.2 migration was also corrected to target the real mTLS agent table, `agent_identities`, rather than a non-existent `agents` table.

## Remaining gate

Agent enrollment/status/station/config/certificate flows span `agent_identities`, enrollment tokens, certificates, remote configuration and legacy keys. They are intentionally isolated in the next release rather than partially tenant-scoping the credential chain.

After agent isolation, search, RF/orbital/program/CVE relationships, background jobs and remaining SSE surfaces still require explicit organization boundaries.
