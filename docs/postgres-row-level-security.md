# PostgreSQL row-level security

Lyrahcle v14.2 adds database-enforced tenant isolation to core operational tables in addition to application-layer organization filters.

## Request model

Every HTTP request is marked with `dbscope.Middleware`. After JWT authentication, the authenticated `organization_id` is attached to the request context. `db.Connect` transfers that scope to PostgreSQL when a pooled connection is acquired:

- `lyrahcle.rls_request=on`
- `lyrahcle.organization_id=<uuid>`

The values are reset before the connection is returned to the pool. This reset is mandatory because pooled PostgreSQL sessions are reused across requests.

An HTTP request without an organization has `lyrahcle.rls_request=on` and an empty organization value, so RLS returns no tenant rows and rejects tenant writes. This is fail-closed behavior for accidentally unscoped handlers.

Trusted non-HTTP contexts such as migrations and background jobs are not marked as HTTP requests and retain cross-tenant access. Code must not reuse an unmarked background context to service HTTP requests.

## Protected tables

Migration `0040_postgres_row_level_security.sql` enables and forces RLS for core tables carrying `organization_id`, including programs, assets, incidents, RF readings/anomalies, orbital events, detection rules/matches and fleet rollout state.

Shared reference catalogs such as satellites and space CVEs intentionally remain outside tenant RLS.

## Agent ingestion

Machine authentication initially resolves an agent identity before the organization is known. After that lookup, the RF ingestion handler attaches the resolved organization to the request context before reading or writing RLS-protected RF data.

## Validation

Run the backend integration suite against PostgreSQL and add any new tenant-owned table to migration 0040 (or a later RLS migration) before release. A tenant table without a reviewed RLS policy is considered a security regression.
