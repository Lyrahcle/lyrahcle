# Configuration

## Backend

| Variable | Default | Production guidance |
|---|---|---|
| `DATABASE_URL` | local Lyrahcle Postgres URL | Required; use protected credentials and TLS where applicable |
| `REDIS_URL` | `localhost:6379` | Required in production for revocation, authorization-state checks, rate limiting, and caching |
| `PORT` | `8080` | API listen port |
| `CELESTRAK_GROUPS` | `stations` | Comma-separated Celestrak groups |
| `TLE_CORRELATION_MAX_AGE` | `24h` | Maximum TLE age allowed to influence automatic orbital severity |
| `TLE_STALE_MAX_AGE` | `72h` | TLE age after which propagation is considered stale and is not used for orbital context |
| `ORBITAL_AUTO_ESCALATION_ENABLED` | `false` | Enables experimental orbital severity escalation only when TLE quality passes the freshness gate |
| `JWT_SECRET` | insecure development value | Current signing key; production requires at least 32 bytes |
| `JWT_SECRET_PREVIOUS` | unset | Previous signing key during a controlled rotation window |
| `JWT_ACCESS_TTL` | `30m` | Access-token lifetime; production refuses values above `2h` |
| `TLS_CERT_FILE` | unset | Required in `ENV=production` by the backend's current safety policy |
| `TLS_KEY_FILE` | unset | Required with `TLS_CERT_FILE` |
| `SEED_ADMIN_EMAIL` | `admin` | Set before first boot |
| `SEED_ADMIN_PASSWORD` | `admin` | Set before first boot; production requires the normal strong-password policy |
| `ENV` | `development` | Set to `production` to enable startup safety checks |
| `WEBAUTHN_RP_ID` | `localhost` | Relying-party domain; use the production application domain |
| `WEBAUTHN_ORIGINS` | `http://localhost:5173` | Exact comma-separated frontend origins; HTTPS is required in production |
| `PASSWORD_RECOVERY_ENABLED` | `false` | Enables self-service password recovery only when SMTP is configured |
| `PASSWORD_RESET_BASE_URL` | `http://localhost:5173/` | Frontend URL that receives the temporary `token` query parameter |
| `SMTP_HOST` | unset | Password-recovery SMTP host |
| `SMTP_PORT` | `465` | Implicit-TLS SMTP port |
| `SMTP_USER` | unset | SMTP username |
| `SMTP_PASSWORD` | unset | SMTP password |
| `SMTP_FROM` | unset | Recovery-email sender |
| `SMTP_SERVER_NAME` | unset | TLS server name used to verify the SMTP certificate |
| `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` | unset | Optional OTLP/HTTP traces endpoint; HTTPS is required in production |
| `RATE_LIMIT_PLATFORM_CEILING_PER_MINUTE` | `50000` | Coarse per-IP emergency ceiling; intentionally higher than tenant budgets so it does not become the normal noisy-neighbor boundary |
| `RATE_LIMIT_TENANT_READ_PER_MINUTE` | `1200` | Aggregate authenticated read budget per tenant |
| `RATE_LIMIT_ACTOR_READ_PER_MINUTE` | `300` | Authenticated read budget per user within a tenant |
| `RATE_LIMIT_TENANT_WRITE_PER_MINUTE` | `300` | Aggregate authenticated mutation budget per tenant |
| `RATE_LIMIT_ACTOR_WRITE_PER_MINUTE` | `120` | Authenticated mutation budget per user within a tenant |
| `RATE_LIMIT_TENANT_AGENT_PER_MINUTE` | `12000` | Aggregate machine/API budget per tenant, isolated from human budgets |
| `RATE_LIMIT_AGENT_PER_MINUTE` | `3000` | Per-agent machine/API budget within a tenant |


Authenticated rate limiting is tenant-aware. Human traffic is checked against both an aggregate organization budget and a per-user budget, with independent read/write namespaces. Valid SDR-agent traffic uses a separate organization+agent namespace so high-rate ingest cannot consume the human API allowance. Redis keys contain hashes of tenant/actor subjects rather than raw UUIDs. The platform IP ceiling remains as a deliberately higher emergency guard, while login/recovery retain stricter pre-authentication abuse limits. If Redis becomes unavailable after startup, the tenant limiter falls back to an in-process fixed-window counter and emits `lyrahcle_rate_limit_backend_fallback_total`. Production still requires Redis for distributed enforcement across replicas.

The bootstrap admin is created only when the users table is empty and is marked for mandatory password change. Changing `SEED_ADMIN_PASSWORD` later does not reset an existing account. Passwords are validated at 12-72 bytes and must include uppercase, lowercase, a number, and a symbol.

JWT rotation uses an overlap window: deploy the new value as `JWT_SECRET`, keep the old value temporarily in `JWT_SECRET_PREVIOUS`, wait longer than `JWT_ACCESS_TTL`, then remove the previous key. Password, Google Authenticator, role, and account-status changes increment a database-backed authorization version and invalidate older JWTs even when Redis is unavailable. Active session metadata is tracked in Redis and exposed only to the owning user.

## Frontend

The frontend is a static Vite build served by nginx. Runtime API routing is controlled by the nginx configuration under `deploy/docker/`.

## SDR agent

| Variable | Default | Description |
|---|---|---|
| `BACKEND_URL` | `http://backend:8080` | Backend base URL |
| `AGENT_ID` | `sdr-agent-dev` | Agent identity used for RF readings |
| `AGENT_API_KEY` | unset | Legacy per-agent credential retained for controlled migration; mTLS is preferred for long-running agents |
| `AGENT_TOKEN` | unset | Legacy development fallback only; do not use human JWTs for unattended production agents |
| `AGENT_CA_CERT` | empty | Optional PEM CA bundle for private or development PKI |
| `FREQUENCY_HZ` | `437500000` | Center frequency |
| `CAPTURE_INTERVAL_SECONDS` | `10` | RF reading interval |
| `IQ_CAPTURE_DIR` | unset | Enables IQ recording |
| `IQ_CAPTURE_INTERVAL_SECONDS` | `60` | IQ recording interval |
| `IQ_CAPTURE_SAMPLES` | `8192` | Samples per IQ recording |
| `REPLAY_ROOT` | `.` for relative paths | Base directory that confines one-shot replay; required for absolute `REPLAY_FILE` |
| `REPLAY_FILE` | unset | Capture metadata file for one-shot replay |
| `REPLAY_WINDOW_SAMPLES` | `65536` | Samples reduced into each replay reading |
| `REPLAY_DELAY_MS` | `300` | Delay between replay uploads |

For production, use the enrollment/mTLS variables below. Legacy agent keys can still be created through the dashboard or `POST /api/v1/agent-keys/` during migration. Device self-report and RF ingest derive machine identity from the validated client certificate or legacy `X-Agent-Key`; an authenticated agent cannot choose a different identity in the body.

## CVE sync

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | local Lyrahcle Postgres URL | Shared application database |
| `REDIS_ADDR` | `localhost:6379` | Dedup/event Redis connection |
| `PROMETHEUS_PORT` | `9091` | Worker metrics port |
| `SYNC_INTERVAL_SEC` | `3600` | Time between sync cycles |
| `NVD_RESULTS_PER_PAGE` | `100` | NVD page size |
| `NVD_RATE_LIMIT_PER_MIN` | `10` | Client-side NVD request limit |
| `LOG_LEVEL` | `info` | Worker log level |
| `ENVIRONMENT` | `development` | Environment label |

The sync keyword set is currently defined in code.

## Docker Compose host variables

`docker-compose.yml` accepts host/root-`.env` overrides for `JWT_SECRET`, `JWT_SECRET_PREVIOUS`, `JWT_ACCESS_TTL`, `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`, `WEBAUTHN_RP_ID`, `WEBAUTHN_ORIGINS`, password-recovery/SMTP settings, `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`, Grafana administrator credentials, and the legacy SDR-agent compatibility variables `AGENT_API_KEY`, `AGENT_TOKEN`, and `AGENT_CA_CERT`. Backend mTLS identity and remote-configuration signing files are mounted/configured explicitly rather than being enabled by the default development Compose profile.

Never commit a production `.env` file or private credentials.


## Observability

The `observability` Docker Compose profile provisions Grafana against the bundled Prometheus service.

| Variable | Default | Description |
|---|---|---|
| `GRAFANA_ADMIN_USER` | `admin` | Local Grafana administrator username |
| `GRAFANA_ADMIN_PASSWORD` | `lyrahcle-dev` | Local development password; override outside isolated development |

Prometheus alert rules live in `deploy/prometheus/alerts.yml`; Grafana provisioning lives under `deploy/grafana/`.

## SDR agent mTLS

| Variable | Purpose | Default |
|---|---|---|
| `AGENT_CA_CERT_FILE` | Backend-side SDR Agent CA certificate | disabled |
| `AGENT_CA_KEY_FILE` | Backend-side SDR Agent CA signing key | disabled |
| `AGENT_CERTIFICATE_TTL` | Issued client-certificate lifetime | `168h` |
| `AGENT_ENROLLMENT_TTL` | One-time enrollment-token lifetime | `15m` |
| `AGENT_CERT_FILE` | SDR agent client-certificate path | disabled |
| `AGENT_KEY_FILE` | SDR agent private-key path | disabled |
| `AGENT_ENROLLMENT_TOKEN` | One-time bootstrap token | unset |
| `AGENT_ROTATE_BEFORE` | Rotate certificate before expiry | `24h` |
| `AGENT_IDENTITY_CHECK_INTERVAL` | Runtime certificate-maintenance interval | `10m` |
| `AGENT_VERSION` | Agent software version reported at enrollment | `v9.1` |
| `AGENT_FIRMWARE_VERSION` | Optional connected-device firmware version | unset |
| `AGENT_CONFIG_SIGNING_KEY_FILE` | Backend Ed25519 private key for signing remote agent configuration | disabled |
| `AGENT_CONFIG_SIGNING_KEY_DIR` | Managed backend-only directory for staged/active/grace/revoked remote-config signing keys | disabled |
| `AGENT_CONFIG_SIGNING_GRACE_PERIOD` | Compatibility window for the previous remote-config signing key after activation | `168h` |

Rotation scheduling is configured from **Platform → Remote Config Signing Keys** when `AGENT_CONFIG_SIGNING_KEY_DIR` enables the managed provider. The persisted policy controls prepare-only automation and warning windows; activation always remains manual after public-key distribution.
| `AGENT_CONFIG_VERIFY_KEY` | SDR agent legacy/default Ed25519 public key; required only for pre-KID envelopes | disabled |
| `AGENT_CONFIG_VERIFY_KEY_DIR` | SDR agent directory of trusted Ed25519 public-key PEM files resolved by KID | disabled |
| `AGENT_CONFIG_POLL_INTERVAL` | SDR agent desired-config polling interval | `30s` |

See `docs/agent-enrollment.md` for certificate identity and `docs/agent-remote-configuration.md` for signed remote configuration.
