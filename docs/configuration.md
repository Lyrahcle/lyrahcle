# Configuration reference

Every environment variable across every Lyrahcle service, in one place. The
backend-specific ones also appear in the main [README](../README.md); this
page is the complete cross-service reference.

## backend (`apps/backend`)

| Variable | Default | Required in production | Description |
|---|---|---|---|
| `DATABASE_URL` | `postgres://lyrahcle:lyrahcle@localhost:5432/lyrahcle?sslmode=disable` | yes | Postgres connection string |
| `REDIS_URL` | `localhost:6379` | no (not yet used by any code path) | Redis connection string, reserved for future queueing/caching |
| `PORT` | `8080` | no | HTTP(S) listen port |
| `CELESTRAK_GROUPS` | `stations` | no | Comma-separated [Celestrak groups](https://celestrak.org/NORAD/elements/) to sync TLEs from on startup and every 6h |
| `JWT_SECRET` | insecure dev default (logs a warning) | **yes** | Secret used to sign auth tokens (HS256). Use a long random value, e.g. `openssl rand -hex 32` |
| `TLS_CERT_FILE` | unset (plain HTTP, logs a warning) | recommended | Path to a TLS certificate; see [deployment guide](./deployment.md#tls) |
| `TLS_KEY_FILE` | unset | recommended | Path to the matching TLS private key |

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

## cve-sync (`apps/cve-sync`)

Placeholder service, not yet implemented — no configuration surface yet.

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
