# Backend bootstrap architecture

Lyrahcle keeps the executable entry point in `apps/backend/cmd/server`, but construction of infrastructure and security dependencies is centralized under `apps/backend/internal/bootstrap`.

## Composition boundary

`cmd/server/main.go` is the composition root. It loads configuration, connects shared infrastructure, starts background workers and schedules, builds the HTTP router, starts the server and coordinates graceful shutdown.

It does not own repository constructors, Redis connection details, agent signing or CA construction, WebAuthn construction, rate-limiter construction or HTTP server timeout construction.

## Bootstrap builders

| File | Responsibility |
|---|---|
| `stores.go` | Builds the PostgreSQL-backed repository set used by HTTP routes and workers |
| `security.go` | Builds agent configuration signing, agent CA, Redis connectivity, WebAuthn and token-revocation checks |
| `ratelimit.go` | Builds platform, login and tenant/actor rate limiters |
| `services.go` | Builds request infrastructure such as sessions, response cache, WebAuthn flows and mail client |
| `http.go` | Builds the HTTP server with the operational timeout policy |

Builders return errors when construction can fail. The executable decides whether an error is fatal based on the existing environment policy.

## Server source layout

The server package remains one Go package while the version-suffixed route files are migrated incrementally. Shared runtime responsibilities are separated from `main.go`:

| File | Responsibility |
|---|---|
| `http_helpers.go` | Request logging, request IDs, request size, filters, cache keys and security headers |
| `mission_runtime.go` | TLE synchronization, RF anomaly/correlation execution and satellite position enrichment |
| `validation_audit.go` | Shared request validation and audit helpers |
| `startup.go` | Initial administrator provisioning |

Keeping these files in `package main` avoids a high-risk big-bang route rewrite. Future domain-package moves can be made one bounded domain at a time while route and authorization contracts remain active.

## Regression gates

`check-backend-bootstrap-architecture.py` prevents the executable from becoming monolithic again and verifies that construction stays in `internal/bootstrap`.

The existing route authorization, tenant rate limiting, Redis TLS/auth, signing-key, background-job, RLS and application tests remain authoritative for behavioral equivalence.
