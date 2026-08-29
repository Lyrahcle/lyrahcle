# Configuration reference

Every environment variable across every Lyrahcle service, in one place. The
backend-specific ones also appear in the main [README](../README.md); this
page is the complete cross-service reference.

## backend (`apps/backend`)

| Variable | Default | Required in production | Description |
|---|---|---|---|
| `DATABASE_URL` | `postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable` | yes | Postgres connection string |
| `REDIS_URL` | `localhost:6379` | no | Redis address for the backend's optional rate limiting, JWT revocation (logout, password change), and response caching. If unreachable at startup, the backend logs a warning and falls back to an in-memory rate limiter with revocation/caching disabled — it does not fail to start. |
| `PORT` | `8080` | no | HTTP(S) listen port |
| `CELESTRAK_GROUPS` | `stations` | no | Comma-separated [Celestrak groups](https://celestrak.org/NORAD/elements/) to sync TLEs from on startup and every 6h |
| `JWT_SECRET` | insecure dev default (logs a warning) | **yes** | Secret used to sign auth tokens (HS256). Use a long random value, e.g. `openssl rand -hex 32` |
| `TLS_CERT_FILE` | unset (plain HTTP, logs a warning) | recommended | Path to a TLS certificate; see [deployment guide](./deployment.md#tls) |
| `TLS_KEY_FILE` | unset | recommended | Path to the matching TLS private key |
| `SEED_ADMIN_EMAIL` | `admin` | **yes, before first boot** | Bootstrap admin account email/username, only used if no users exist yet |
| `SEED_ADMIN_PASSWORD` | `admin` | **yes, before first boot** | Bootstrap admin account password — `admin`/`admin` is a well-known default; rotate it via `POST /auth/change-password` immediately after first login if you didn't override this |
| `ENV` | `development` | **yes** | Set to `production` to enable a startup safety check: the backend refuses to start if `JWT_SECRET`, `SEED_ADMIN_PASSWORD`, or TLS are still on their insecure defaults, instead of just logging a warning |

## frontend (`apps/frontend`, served by nginx in the container)

The frontend has no runtime environment variables — it's a static build. The
only configuration surface is the nginx config file mounted into the
container (`deploy/docker/nginx.frontend.conf` for HTTP, or
`nginx.frontend.tls.conf` for HTTPS), which sets where `/api/` requests are
proxied. Change `proxy_pass` there if the backend isn't reachable at
`http://backend:8080` in your deployment.

## sdr-agent (`apps/sdr-agent`)

| Variable | Default | Description |
|---|---|---|
| `BACKEND_URL` | `http://backend:8080` | Base URL of the backend this agent reports to. Use `https://` once TLS is enabled |
| `AGENT_ID` | `sdr-agent-dev` | This agent's identifier; must match the `agent_id` an `AGENT_API_KEY` was issued for |
| `AGENT_API_KEY` | unset | Preferred credential — create one via `POST /api/v1/agent-keys` (see [README](../README.md#sdr-agent-authentication)) |
| `AGENT_TOKEN` | unset | Fallback credential (shared operator/admin JWT). Used only if `AGENT_API_KEY` is unset |
| `AGENT_INSECURE_SKIP_VERIFY` | `false` | Skip TLS certificate verification. **Local development only**, never against a production backend |
| `FREQUENCY_HZ` | `437500000` | Center frequency the (currently mock) capture source reports readings for |
| `CAPTURE_INTERVAL_SECONDS` | `10` | Seconds between simulated captures |
| `IQ_CAPTURE_DIR` | unset (disabled) | Directory to record periodic raw IQ captures to, for later forensic replay — see [docs/forensics.md](./forensics.md) |
| `IQ_CAPTURE_INTERVAL_SECONDS` | `60` | Seconds between IQ recordings, when `IQ_CAPTURE_DIR` is set |
| `IQ_CAPTURE_SAMPLES` | `8192` | Samples per IQ recording |
| `REPLAY_FILE` | unset (disabled) | Path to a saved capture's `.json` metadata file — when set, the agent replays it as historical RF readings and exits instead of running the live loop |
| `REPLAY_WINDOW_SAMPLES` | `65536` | Samples per re-ingested power reading during replay |
| `REPLAY_DELAY_MS` | `300` | Delay between posted readings during replay |

## cve-sync (`apps/cve-sync`)

The single source of truth for the space-CVE feed (`space_cves` table). Runs as
its own worker with a Redis-backed dedup cache and Prometheus metrics — the
backend only reads this table (`GET /api/v1/threat-intel/space-cves`) and never
writes to it. See [CHANGELOG.md](../CHANGELOG.md) for why the backend's earlier,
duplicate in-process sync was removed.

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | `postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle` | Postgres connection string |
| `REDIS_ADDR` | `localhost:6379` | Redis address, used to skip re-fetching a keyword that was synced recently |
| `PROMETHEUS_PORT` | `9091` | Port the `/metrics` endpoint is served on |
| `SYNC_INTERVAL_SEC` | `3600` | Seconds between sync cycles |
| `NVD_RESULTS_PER_PAGE` | `100` | Results requested per page from the NVD API |
| `NVD_RATE_LIMIT_PER_MIN` | `10` | Client-side rate limit applied to NVD API calls |
| `LOG_LEVEL` | `info` | Log verbosity |
| `ENVIRONMENT` | `development` | Free-text environment label included in worker logs |

Sync keywords (`satellite`, `space`, `RF`, `TM/TC`, `firmware`, `orbit`) are
currently fixed in code (`apps/cve-sync/internal/config/config.go`), not an
environment variable.

## docker-compose

| Variable (host env, not container env) | Default | Description |
|---|---|---|
| `JWT_SECRET` | `insecure-dev-secret-change-me` | Passed through to the `backend` service |
| `AGENT_API_KEY` | *(empty)* | Passed through to the `sdr-agent` service |
| `AGENT_TOKEN` | *(empty)* | Passed through to the `sdr-agent` service |
| `AGENT_INSECURE_SKIP_VERIFY` | `false` | Passed through to the `sdr-agent` service |

Profiles: `sdr-agent` only starts with `docker compose --profile sdr up`, since
it depends on either a live agent key or hardware not everyone running the
stack has yet.

## Precedence

Docker Compose reads host environment variables (or a `.env` file in the repo
root, which Compose loads automatically) to fill in the `${VAR:-default}`
placeholders in `docker-compose.yml`. Outside Compose — e.g. running the Go
binaries directly, or in a non-Compose production deployment — set these as
regular process environment variables.
