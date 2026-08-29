# Security Roadmap

## Shipped

- [x] JWT authentication with bcrypt password hashing
- [x] `viewer`, `operator`, and `admin` RBAC
- [x] Google Authenticator TOTP two-factor authentication with session revocation after security-state changes
- [x] Admin-managed user creation and role changes; public self-registration disabled
- [x] Per-agent revocable API keys and authenticated device self-report
- [x] Redis-backed JTI and user-wide JWT revocation, distributed rate limiting, and response caching
- [x] Independent rate-limit namespaces for global traffic and login attempts
- [x] Audit logging for security-relevant actions
- [x] Request correlation IDs with safe client ID propagation
- [x] Strict request validation, body-size limits, and API security headers
- [x] TLS support and production startup checks for insecure defaults
- [x] Versioned, idempotent PostgreSQL migrations shared by production and integration tests
- [x] Unit, integration, HTTP, SDR fuzz, accessibility, authentication E2E, and smoke testing in CI
- [x] `govulncheck`, `npm audit`, Dependabot, dependency review, and Trivy scanning
- [x] Prometheus metrics, provisioned Grafana dashboard/alerts, and graceful process shutdown
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
- [ ] Complete browser token-storage and CSRF design review
- [x] Generate CycloneDX SBOMs in CI and attach them to tagged releases
- [x] Sign tagged container images with keyless Cosign and publish GitHub build provenance attestations
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
## v8.1 pre-v1 security baseline

- [x] Formal STRIDE threat model and compliance-readiness checklist
- [x] CycloneDX SBOM generation with SHA-256 manifest in CI and releases
- [x] Pull-request dependency review gate
- [x] SDR-agent fuzz targets with bounded corpora and CI execution
- [x] >80% coverage gate across critical SDR processing packages
- [x] Authentication E2E test for login, authenticated request, logout revocation, and relogin
- [x] JWT issuer/audience/subject validation and short configurable access-token TTL
- [x] Previous signing-key overlap for controlled JWT key rotation
- [x] User-wide session revocation after password, TOTP, or role changes
- [x] Production fail-closed startup when Redis revocation state is unavailable
- [x] Fuzz-driven hardening for non-finite IQ samples, bounded forensic filenames, and restrictive capture permissions


## v8.2 observability and session hardening

- [x] Redis-backed active-session inventory for each user
- [x] Per-session revocation and sign-out-everywhere
- [x] Session deletion after password, role, and Google Authenticator changes
- [x] Google Authenticator-specific setup UX with first-code verification
- [x] Prometheus authentication and SSE operational metrics
- [x] Provisioned Grafana dashboard and Prometheus alert rules
- [x] Keyless Cosign signatures for tagged container images
- [x] GitHub build-provenance attestations bound to immutable image digests
- [x] Authentication E2E coverage for Google Authenticator enable/disable and session invalidation
- [ ] Distributed tracing with OpenTelemetry

## Next identity and authorization review

- [ ] Force password change for the default seeded administrator on first successful login
- [ ] Strengthen password policy and compromised-password guidance
- [ ] Add WebAuthn security-key support as an isolated authentication milestone
- [ ] Add password recovery with short-lived links and second-factor verification
- [ ] Define an explicit permission matrix before expanding roles and administrator subroles
- [ ] Add user disable/edit/delete lifecycle with full audit coverage
- [ ] Review every sensitive endpoint against the permission matrix
- [ ] Add automatic SDR agent credential rotation before mTLS enrollment
