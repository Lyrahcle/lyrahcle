# Security Policy

## Reporting a vulnerability

Do not open a public issue for suspected vulnerabilities. Report them privately to `manu.ciberseguridad@gmail.com` with the affected component, reproduction steps, expected impact, and any proof of concept that is safe to share.

We aim to acknowledge reports within 72 hours. Please allow time for triage and remediation before public disclosure.

## Scope

This policy covers the open-core repository, including `apps/`, `packages/`, deployment files, CI workflows, and the documented API surface. Third-party infrastructure and external services are outside this repository's direct scope.

## Security expectations

Production deployments must use a `JWT_SECRET` of at least 32 bytes, a short `JWT_ACCESS_TTL`, non-default bootstrap credentials, TLS, protected PostgreSQL/Redis access, individual SDR agent credentials, and current dependencies. The backend refuses several insecure defaults when `ENV=production`.

See [docs/ieee3536-threat-model.md](./docs/ieee3536-threat-model.md) for the current STRIDE model, [docs/compliance-checklist.md](./docs/compliance-checklist.md) for readiness evidence, [SECURITY_ROADMAP.md](./SECURITY_ROADMAP.md) for implementation status and [docs/deployment.md](./docs/deployment.md) for deployment requirements.

## SDR Agent CA

The SDR Agent CA private key can issue machine identities and must be treated as a production signing secret. Do not commit it, bake it into images, or expose it through logs or API responses. Mount it with least privilege from a secrets manager or HSM-backed workflow. Agent private keys are generated and stored on each SDR agent and are never transmitted to Lyrahcle.


## Agent remote configuration signing key

The v9.1 Ed25519 configuration-signing key is a separate high-value secret from the SDR Agent CA. Keep the private key backend-only and distribute only its public verification key to agents. Remote configuration remains restricted to typed capture settings; do not extend the channel with shell commands or arbitrary executable payloads.
