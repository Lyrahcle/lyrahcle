# Multi-Tenancy and Organization Isolation Guide

## Overview

Lyrahcle is a multi-tenant platform where each organization operates independently. Tenant isolation is enforced at both the HTTP layer (permission-driven) and the database layer (PostgreSQL RLS) for defense-in-depth.

## Multi-Tenant Foundation (v10.0–v10.6)

### Organization Scope (v10.0–v10.4)

The following are **organization-scoped**:
- Users and authentication context
- Agent identities and credentials
- Asset management
- Incident response and workflow
- Audit events
- Background job and correlation results

### Tenant-Local Ownership (v10.5–v10.6)

Tenant-local ownership now covers:

**Operational data:**
- Programs (satellite mission programs)
- RF readings and RF anomalies
- Orbital events (operator/trusted-integration)
- CVE asset alert relationships
- Search results (for assets and incidents)

**Shared catalogs (global, intentional):**
- `satellites` — global TLE catalog (shared reference)
- `space_cves` — global CVE catalog (shared reference)
- Tenant-owned relationships to those references remain scoped

### Multi-Tenant Isolation Audit Gate (v10.6)

The foundation includes:
- Automated source-contract checks
- Complete tenant-owned domain snapshot
- Optional PostgreSQL-backed integration tests (`LYRAHCLE_TEST_DATABASE_URL`)
- Negative cross-tenant regression tests

**Release gate:** New tenant-owned domains cannot be accepted without organization ownership and negative cross-tenant tests.

## Tenant-Owned Domains

The following records **must** carry `organization_id` and **never** be listed or mutated through global user-facing queries:

- `users` — user accounts, roles, auth state
- `agent_identities` — SDR agent credentials and configuration
- `assets` — satellite and ground station inventory
- `incidents` — security incidents and workflow state
- `audit_log` — security and administrative audit events
- `programs` — mission programs and deployments
- `rf_readings` — RF ingestion data and statistics
- `rf_anomalies` — detected anomalies
- `orbital_events` — orbital tracking and prediction
- `asset_alerts` — CVE/security alert relationships

## HTTP-Layer Isolation

All tenant-owned HTTP endpoints derive `organization_id` from JWT claims or agent identity. Requests cannot specify another organization.

**Isolation enforcement:**
- `/api/v1/assets` — returns only organization-owned assets
- `/api/v1/incidents` — returns only organization incidents
- `/api/v1/audit-log` — returns only organization audit events
- Search endpoints — filter to organization scope automatically

Legacy routes that previously returned global results now enforce tenant scope automatically. Requests from Org-A cannot access Org-B's data even with valid bearer tokens.

## PostgreSQL Row-Level Security (v14.2)

PostgreSQL RLS provides **database-level enforcement** independent of application logic. This is defense-in-depth: if application code has a bug, RLS still blocks unauthorized access.

### RLS Enforcement

- All tenant-owned tables have RLS policies
- Policies check connection-scoped `app.current_organization_id` setting
- HTTP database connections without an authenticated organization are **fail-closed** (no rows returned)

### Connection Pool Safety

Connection-scoped RLS settings are reset before pool reuse to prevent tenant scope leakage between requests:

```go
// Pseudocode
defer conn.ResetRLSContext()  // Cleared before returning to pool
```

### Policy Examples

```sql
-- RLS policy for incidents table
CREATE POLICY org_isolation_incidents ON incidents
  USING (organization_id = current_setting('app.current_organization_id')::uuid);

-- Applied to all SELECT, UPDATE, DELETE
```

## Audit Events and Organization Context

### HTTP Audit Events

Authenticated HTTP handlers derive `organization_id` from JWT claims:

```go
// JWT payload includes organization_id
AuditStore.InsertInOrganization(ctx, event)
```

### Automated Correlation Audit

System workers (background jobs, correlation engines) that do not have user JWT claims **must** provide explicit `organization_id`:

```go
// Worker audit context
AuditStore.InsertInOrganization(ctx, event, organizationID)
```

### RF Anomaly Correlation

1. RF anomalies carry `OrganizationID` at ingestion
2. Detection-rule incident creation copies that value:
   ```go
   CreateIncidentInput.OrganizationID = rfAnomaly.OrganizationID
   ```
3. Resulting incident scans organization back into the returned object
4. Correlation audit events use the same organization

### CVE Background Alerts

Global CVE matching scans assets across organizations, but:
1. Each matched asset carries its `OrganizationID`
2. The background worker records the audit event **inside that asset's organization**

This ensures compliance events are visible to the correct tenant's auditors.

## Configuration

No special configuration is required. Tenant isolation is always enforced.

### Test Database

To run cross-tenant negative integration tests, set:

```bash
export LYRAHCLE_TEST_DATABASE_URL="postgres://user:pass@localhost/test_db"
make test
```

This creates two temporary organizations (Org-A, Org-B) and verifies:
- Org-A cannot read/modify Org-B incidents
- Org-A cannot read/modify Org-B audit events
- Global search returns correct org-scoped results
- RLS policies block cross-tenant queries

## Remaining Review

Before v1.0:
- [ ] All background workers preserve organization context
- [ ] Correlation and scheduled jobs pass organization to audit helpers
- [ ] New tables containing customer data receive `organization_id` at creation
- [ ] Negative cross-tenant tests exist for all sensitive domains
- [ ] Load-test validation of RLS performance impact

## Common Issues

### Q: Can an organization admin access another org's data?
**A:** No. HTTP layer and RLS both reject cross-organization access. Tenant admins are scoped to their organization only.

### Q: What about shared satellites or CVEs?
**A:** Global reference catalogs (satellites, CVEs) are shared. Tenant-owned relationships (which satellite an org monitors, which CVEs it acknowledges) remain scoped.

### Q: If RLS fails, does the app still work?
**A:** No. RLS blocks all cross-tenant queries at the database layer. If RLS is disabled by mistake, queries return no data and the application fails safely.

### Q: How is organization_id set for new connections?
**A:** In the middleware, before business logic executes:

```go
SetRLSOrganization(db, user.OrganizationID)  // Set before query
defer ResetRLSContext(db)                      // Reset after
```

## Checklist for New Features

- [ ] New domain has `organization_id` foreign key
- [ ] HTTP endpoints enforce tenant scope in WHERE clauses
- [ ] Audit events record the correct organization
- [ ] Background jobs pass organization to audit helpers
- [ ] RLS policy created for database-level defense
- [ ] Negative cross-tenant tests written and passing
- [ ] RBAC permissions respect organization boundaries
