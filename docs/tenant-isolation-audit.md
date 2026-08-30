# Tenant isolation audit

v10.6 is the release gate for the multi-tenant foundation.

## Tenant-owned domains

The following records are expected to carry `organization_id` and must never be listed or mutated through a global user-facing query:

- users
- agent identities and agent credential/config state
- assets
- incidents
- audit events
- programs
- RF readings/anomalies
- orbital events
- asset/CVE alert relationships

## Shared reference catalogs

These remain global by design:

- satellites/TLE catalog
- CVE catalog

Tenant-local relationships to those references remain scoped.

## Automated gates

The repository contains source-contract tests that reject known unsafe global store calls in tenant-owned HTTP surfaces and verify search/SSE policies.

A database-backed integration test is also available when `LYRAHCLE_TEST_DATABASE_URL` is set. It creates two organizations and verifies tenant-local snapshots independently.

## Remaining review

Background workers, correlation and scheduled jobs must preserve organization context whenever they operate on tenant-owned records. Any new table containing customer-specific operational data must receive `organization_id` at creation time and negative cross-tenant tests before merge.
