# Security and Compliance Readiness Checklist

This checklist is an engineering evidence tracker, not a certification claim. Map it to the framework required by a customer or regulator during a formal assessment.

## Identity and access

- [x] Server-side RBAC for all privileged API mutations
- [x] Google Authenticator TOTP available for human accounts
- [x] Public self-registration disabled
- [x] Short-lived JWT access tokens with issuer/audience validation
- [x] Logout, active-session inventory, per-session revocation, and user-wide session revocation
- [x] Signing-key overlap for controlled JWT key rotation
- [x] Per-agent revocable credentials
- [ ] Automated agent credential rotation and enrollment
- [ ] mTLS for production SDR agents

- [ ] Complete browser token-storage and CSRF design review before v1.0

## Secure development lifecycle

- [x] Unit and integration tests in CI
- [x] Race detector, `go vet`, `gosec`, `govulncheck`, and `npm audit`
- [x] Trivy release image scanning
- [x] Dependency review on pull requests
- [x] SDR-agent fuzz targets executed in CI
- [x] >80% coverage gate for critical SDR packages
- [x] Authentication E2E smoke flow
- [x] CycloneDX SBOM generated in CI and release workflow
- [ ] Repository-wide >80% statement coverage
- [x] Signed build provenance and keyless container signatures

## Data protection

- [x] TLS required by backend production startup policy
- [x] Database credentials excluded from backup output
- [x] Restrictive backup and forensic-capture file permissions
- [x] Request logs avoid bodies and authentication secrets
- [x] Input sizes and pagination are bounded
- [ ] Centralized secrets manager
- [ ] Formal data retention schedule
- [ ] Encrypted backup policy with tested key recovery

## Detection and response

- [x] Security audit log
- [x] Request correlation IDs
- [x] Incident lifecycle and immutable evidence references
- [x] RF anomaly detection and typed detection rules
- [x] Deterministic incident correlation with cooldown controls
- [ ] External SIEM export
- [ ] Incident response runbooks and tabletop exercise evidence
- [ ] Tamper-evident long-term audit archive

## Resilience

- [x] Liveness and readiness endpoints
- [x] Redis required in production for authorization-state guarantees
- [x] Graceful backend shutdown
- [x] Database backup tooling
- [ ] Automated restore test in CI/staging
- [ ] Documented RPO/RTO
- [ ] HA PostgreSQL/Redis/backend reference architecture

## Supply chain

- [x] Lockfiles and Go module checksums
- [x] CycloneDX SBOMs with SHA-256 manifest
- [x] Dependency and vulnerability review gates
- [x] Release image vulnerability scan
- [ ] GitHub Actions pinned to immutable commit SHAs
- [x] GitHub build-provenance attestations for release image digests
- [x] Keyless Cosign signatures for release image digests
- [ ] Enforced signature/provenance verification policy at deployment time

## Governance evidence

Before a production release, retain:

- CI run URL and commit SHA
- SBOM files and `SHA256SUMS`
- vulnerability scan results
- approved release/change record
- backup/restore evidence
- current threat-model review date
- unresolved risk acceptance records
