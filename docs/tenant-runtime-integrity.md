# Tenant runtime integrity

Automated and administrative writes must preserve the owning organization all the
way into incidents and audit records.

## Audit events

Authenticated HTTP handlers derive `organization_id` from JWT claims. System
workers that do not have user claims must provide an explicit `organization_id`
in their audit context. The audit helper writes via `AuditStore.InsertInOrganization`;
the legacy `Insert` method is retained only for default-organization compatibility.

## Automated RF correlation

RF anomalies carry `OrganizationID`. Detection-rule incident creation copies that
value into `CreateIncidentInput.OrganizationID`, and the resulting incident scans
the persisted organization back into the returned object. Correlation audit events
use the same organization.

## CVE background alerts

Global CVE matching may scan assets across organizations, but each matched asset now
carries its `OrganizationID`, and the worker records the audit event inside that
asset's organization.
