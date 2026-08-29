# Security Policy

## Reporting a vulnerability

Do not open a public issue for suspected vulnerabilities. Report them privately to `manu.ciberseguridad@gmail.com` with the affected component, reproduction steps, expected impact, and any proof of concept that is safe to share.

We aim to acknowledge reports within 72 hours. Please allow time for triage and remediation before public disclosure.

## Scope

This policy covers the open-core repository, including `apps/`, `packages/`, deployment files, CI workflows, and the documented API surface. Third-party infrastructure and external services are outside this repository's direct scope.

## Security expectations

Production deployments must use a strong `JWT_SECRET`, non-default bootstrap credentials, TLS, protected PostgreSQL/Redis access, individual SDR agent credentials, and current dependencies. The backend refuses several insecure defaults when `ENV=production`.

See [SECURITY_ROADMAP.md](./SECURITY_ROADMAP.md) for implementation status and [docs/deployment.md](./docs/deployment.md) for deployment requirements.
