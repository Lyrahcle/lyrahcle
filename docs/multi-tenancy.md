# Organizations and multi-tenancy

## v10.0–v10.3
Organizations/JWT context, user administration isolation, critical-domain ownership, and tenant-isolated asset/incident/audit handlers.

## v10.4 — agent security boundary

Agent administration is now organization-scoped:

- admin agent listing;
- enrollment-token issuance;
- station updates;
- lifecycle/status changes;
- remote configuration read/history/create/rollback;
- legacy agent-key list/create/revoke.

The dependent credential/config tables carry explicit `organization_id`: enrollment tokens, certificates, config versions/state/acks and legacy keys. Existing rows inherit ownership from `agent_identities`; unmatched legacy keys fall back to the default organization for backward compatibility.

mTLS self-service endpoints continue to authenticate from the certificate-to-agent binding. Administrative JWT operations additionally require the agent identity to belong to the caller organization.

## Remaining multi-tenant work

RF/orbital/program/CVE relationships, global search, background jobs and remaining streaming surfaces still need tenant propagation. Those should be completed before multi-tenancy is advertised as fully production-isolated.
