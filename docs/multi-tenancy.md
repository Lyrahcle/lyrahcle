# Organizations and multi-tenancy

## v10.0–v10.4
Organizations/JWT context, user administration, critical operational domains and the agent security chain are organization-scoped.

## v10.5 — RF, programs, orbital context and search

Tenant-local ownership now covers:

- programs;
- RF readings and RF anomalies;
- operator/trusted-integration orbital events;
- asset CVE alert listing/acknowledgement;
- global search results for assets and incidents.

RF ingest resolves the organization from the authenticated JWT or agent identity/key and persists it with the reading; anomaly records inherit the same tenant.

### Shared catalogs

`satellites` and `space_cves` intentionally remain global reference catalogs. Search may return those shared references to every tenant, while tenant-owned assets/incidents remain isolated.

### Remaining work

Before declaring complete enterprise multi-tenancy, background workers/correlation paths and any remaining domain-specific queries should be audited for tenant propagation, followed by true database-backed Org-A/Org-B negative integration tests.


## v10.6 — isolation audit gate

The multi-tenant foundation now includes automated source-contract checks, a complete tenant-owned domain snapshot and an optional PostgreSQL-backed Org-A/Org-B integration test controlled by `LYRAHCLE_TEST_DATABASE_URL`.

This is the release gate before moving to granular RBAC and SSO. New tenant-owned domains should not be accepted without organization ownership and negative cross-tenant tests.
