# Organizations and multi-tenancy foundation (v10.0)

v10.0 introduces the first backward-compatible organization boundary.

## Included

- `organizations` persistence with a stable default organization.
- Existing users migrate automatically into the default organization.
- Users expose `organization_id`.
- Newly issued user JWTs carry `organization_id`.
- Admin-only organization list/create API.
- Authentication revocation rejects organization mismatches.

## Isolation status

This is a foundation, not a claim of complete tenant isolation. Agents, assets, incidents, audit, RF, satellites and orbital data still need domain-by-domain organization scoping and negative cross-tenant tests before production multi-tenancy is enabled.

## Next gate

Scope critical domain stores and APIs to the authenticated organization, then add explicit tests proving Org A cannot read, mutate, search, stream or correlate Org B data.
