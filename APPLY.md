# Block 7 - Redis TLS and authentication

This package extends the validated TypeScript migration baseline with production Redis transport security for the backend and CVE worker.

## Scope

- `redis://` remains supported for local development.
- Production requires authenticated `rediss://`.
- TLS uses the system trust store and enforces TLS 1.2 or newer.
- `REDIS_CA_CERT_FILE` can add a private CA.
- `REDIS_TLS_SERVER_NAME` can override hostname verification for controlled deployments.
- Redis credentials are removed from startup logs and connection diagnostics.
- The CVE worker now consumes `REDIS_URL` consistently with the backend.
- Helm can mount an optional Redis CA secret into both workloads.
- Terraform keeps ElastiCache opt-in but requires TLS transit mode and production AUTH.

## Local validation

Local Docker Compose continues to use plaintext Redis on the isolated development network. Run:

```bash
docker compose up -d postgres redis
./scripts/test-local-v1.sh
```

## AWS secret format

Populate the application secret with an authenticated URI such as:

```text
rediss://:<AUTH_TOKEN>@<PRIMARY_ENDPOINT>:6379/0
```

URI-reserved characters in the token must be percent-encoded. Do not commit real tokens.
