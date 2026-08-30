# Documentation maintenance

Lyrahcle keeps product history, current backlog, API contracts, security evidence, and deployment guidance separate so a shipped milestone cannot remain accidentally documented as future work.

## Source-of-truth hierarchy

| Topic | Canonical source |
|---|---|
| Current product backlog/status | `docs/improvement-roadmap.md` |
| Security controls and pre-v1 security gates | `SECURITY_ROADMAP.md` |
| Execution order and milestone intent | `PLAN.md` |
| Historical shipped changes | `CHANGELOG.md` |
| HTTP API surface | `apps/backend/cmd/server/openapi.yaml` plus registered Go routes |
| Runtime configuration/defaults | implementation under `apps/*` |
| Human-readable environment reference | `docs/configuration.md` |
| Deployment requirements | `docs/deployment.md` |
| Threat assumptions/residual risk | `docs/ieee3536-threat-model.md` |
| Evidence/readiness checklist | `docs/compliance-checklist.md` |

When sources disagree, fix the documentation to match implemented behavior first, then update status trackers. Do not weaken implementation safety checks to match stale documentation.

## Required documentation updates

A change must update documentation in the same pull request when it changes any of the following:

- HTTP path, method, request/response shape, authentication, authorization, or status behavior;
- environment variable, default, startup validation, secret requirement, or deployment topology;
- user role, account lifecycle, TOTP/WebAuthn/recovery behavior, session invalidation, or machine identity;
- agent enrollment, certificate rotation, signed configuration, orbital correlation, or severity policy;
- migration, backup/restore, observability, release, SBOM/signing, or other operator procedure;
- roadmap status of an item that has shipped or been intentionally deferred.

## Pre-merge consistency checks

Before merging a documentation-affecting change:

1. Confirm all local Markdown links resolve.
2. Parse `apps/backend/cmd/server/openapi.yaml` successfully.
3. Compare sensitive endpoint access language against the registered Go middleware/role checks.
4. Compare documented environment defaults against implementation defaults and `docker-compose.yml`.
5. Search for stale milestone wording such as `planned`, `future`, or `next` around features that have shipped.
6. Keep legacy compatibility mechanisms explicitly labeled as legacy; do not present them as the preferred production path.
7. Keep `ORBITAL_AUTO_ESCALATION_ENABLED=false` documented as the safe default until representative physical SDR validation is complete.

## Current pre-v1 compatibility notes

- Long-running SDR agents should use certificate-backed mTLS. `X-Agent-Key` remains a migration compatibility mechanism.
- Signed remote SDR configuration is shipped and requires active mTLS identity for polling/acknowledgement.
- WebAuthn security keys are shipped; they are not a future identity feature.
- OpenTelemetry tracing is available when configured; request IDs remain always-on.
- Guarded backup/restore is exercised in CI, while a production-like DR exercise and recorded RPO/RTO remain open.
- Orbital correlation and TLE freshness gating are shipped, but automatic orbital severity escalation stays disabled by default pending physical SDR validation.
