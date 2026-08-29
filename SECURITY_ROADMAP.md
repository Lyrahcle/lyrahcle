# Security Roadmap

## Shipped

- [x] JWT authentication with bcrypt password hashing
- [x] `viewer`, `operator`, and `admin` RBAC
- [x] TOTP two-factor authentication
- [x] Admin-managed user creation and role changes; public self-registration disabled
- [x] Per-agent revocable API keys and authenticated device self-report
- [x] Redis-backed JWT revocation, distributed rate limiting, and response caching with graceful degradation
- [x] Independent rate-limit namespaces for global traffic and login attempts
- [x] Audit logging for security-relevant actions
- [x] Request correlation IDs with safe client ID propagation
- [x] Strict request validation, body-size limits, and API security headers
- [x] TLS support and production startup checks for insecure defaults
- [x] Versioned, idempotent PostgreSQL migrations shared by production and integration tests
- [x] Unit, integration, HTTP, fuzz, accessibility, and smoke testing in CI
- [x] `govulncheck`, `npm audit`, Dependabot, and Trivy scanning
- [x] Prometheus metrics and graceful process shutdown
- [x] IQ capture and forensic replay
- [x] Robust RF anomaly baselines using median/MAD and EWMA techniques
- [x] CVE-to-asset alerting and operator acknowledgement
- [x] EN/ES/FR/DE localization and WCAG-oriented frontend improvements
- [x] SSE orbital streaming preserved through middleware instrumentation
- [x] Incident response with role-gated workflow changes, transaction-backed timelines, audit events, and RF anomaly escalation
- [x] Typed RF detection rules with admin-only mutation, persisted matches, bounded evaluation, serialized cooldown suppression, and deterministic incident correlation

## Before v1.0

- [ ] Validate real SDR hardware integrations such as RTL-SDR/HackRF
- [ ] Add agent credential rotation and enrollment lifecycle
- [ ] Evaluate mTLS for long-running SDR agents
- [ ] Produce SBOM, provenance, and signed release artifacts
- [ ] Document and test backup restore and disaster recovery procedures
- [ ] Add production deployment patterns for redundant backend/Redis/PostgreSQL infrastructure
- [ ] Extend correlation beyond RF into asset alerts, orbital events, agent posture, and audit evidence

## v7.1 hardening

- [x] Strict incident status transitions
- [x] Idempotent incident creation
- [x] Typed incident evidence references
- [x] Readiness checks for database and Redis
- [x] Structured request logs without request bodies or credentials
- [x] Pagination defaults capped at 25/100
- [x] Race detector and gosec CI gates
- [x] SDR TLS verification through trusted CA bundles without verification bypass

## v8 detection engineering

- [x] Non-executable typed RF detection-rule model
- [x] Admin-only rule creation and mutation
- [x] Per-rule/per-agent cooldown suppression with transactional advisory locks
- [x] Maximum active-rule evaluation bound
- [x] Persisted detection matches linked to correlated incidents
- [x] Highest-severity deterministic correlation for multiple matches on one anomaly
