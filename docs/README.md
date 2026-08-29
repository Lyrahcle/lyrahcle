# Documentation

This directory contains the technical documentation for Lyrahcle. The root README stays intentionally short.

| Document | Use it for |
|---|---|
| [Architecture](./architecture.md) | Components, trust boundaries, data flows, and storage ownership |
| [API reference](./api-reference.md) | Authentication, roles, endpoints, SSE, agent API keys, request IDs |
| [Configuration](./configuration.md) | Environment variables for every service |
| [Deployment](./deployment.md) | Production checklist, TLS, backups, monitoring, updates |
| [IQ forensics](./forensics.md) | Recording and replaying SDR IQ captures |
| [Observability](./observability.md) | Prometheus metrics, Grafana provisioning, alerts, and request correlation |
| [STRIDE threat model](./ieee3536-threat-model.md) | Assets, trust boundaries, STRIDE threats, mitigations, residual work |
| [Compliance checklist](./compliance-checklist.md) | Pre-v1 security evidence and readiness tracking |
| [Software supply chain](./supply-chain.md) | SBOM generation, dependency gates, signed releases and provenance |
| [Security policy](../SECURITY.md) | Private vulnerability reporting and supported scope |
| [Security roadmap](../SECURITY_ROADMAP.md) | Hardening completed and remaining before v1.0 |
| [Development plan](../PLAN.md) | Product and architecture priorities |
| [Improvement roadmap](./improvement-roadmap.md) | Shipped, next, and long-term product/security backlog |
| [Contributing](../CONTRIBUTING.md) | Local development, testing, conventions, PRs |
| [Releasing](../RELEASING.md) | Versioning and GHCR release process |

The backend also embeds an OpenAPI specification at `/openapi.yaml` and Swagger UI at `/docs`.

- [User administration and permissions](user-administration.md)
- [Backup and restore](backup-and-restore.md)
- [Sensitive endpoint audit](sensitive-endpoint-audit.md)

