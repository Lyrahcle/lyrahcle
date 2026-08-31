# Secret management

Lyrahcle resolves sensitive backend configuration through a provider boundary instead of reading every secret directly from the process environment.

## Providers

- `SECRET_PROVIDER=env` — development compatibility only. Production refuses to start with this provider.
- `SECRET_PROVIDER=file` — reads one root-owned/owner-only regular file per secret from `SECRET_FILE_DIR`. Symlinks and group/world-readable files are rejected.
- `SECRET_PROVIDER=remote` — retrieves secrets from an HTTPS service using `GET /v1/secrets/{KEY}` and checks `/healthz`. This is the cloud-neutral integration point for Vault, AWS Secrets Manager/Parameter Store, or another broker/gateway.

For a remote provider, configure `SECRET_PROVIDER_ENDPOINT`. Bootstrap authentication can use `SECRET_PROVIDER_TOKEN` or preferably `SECRET_PROVIDER_TOKEN_FILE` when the runtime can mount a workload credential. The bootstrap token is never exposed through the Lyrahcle API.

The backend currently resolves `DATABASE_URL`, `REDIS_URL`, `JWT_SECRET`, `JWT_SECRET_PREVIOUS`, `SEED_ADMIN_PASSWORD`, `SMTP_PASSWORD`, and `AGENT_CONFIG_SIGNER_TOKEN` through this boundary. Non-secret operational settings remain normal environment configuration.

## Platform UI

Platform administrators can inspect provider type, health, location, and the names of managed secret references. Secret values are never returned to the browser. Rotation and writes remain the responsibility of the custody system; exposing secret mutation through Lyrahcle would weaken the trust boundary.

## AWS path

The `remote` provider is intentionally provider-neutral. A future AWS deployment can place an IAM-authenticated broker in front of Secrets Manager/Parameter Store, or add a native AWS implementation behind the same provider interface without changing consumers.
