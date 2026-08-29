# Configuration

## Backend

| Variable | Default | Production guidance |
|---|---|---|
| `DATABASE_URL` | local Lyrahcle Postgres URL | Required; use protected credentials and TLS where applicable |
| `REDIS_URL` | `localhost:6379` | Recommended for distributed rate limiting, JWT revocation, caching, and queue-backed features |
| `PORT` | `8080` | API listen port |
| `CELESTRAK_GROUPS` | `stations` | Comma-separated Celestrak groups |
| `JWT_SECRET` | insecure development value | Required and strong; production refuses the default |
| `TLS_CERT_FILE` | unset | Required in `ENV=production` by the backend's current safety policy |
| `TLS_KEY_FILE` | unset | Required with `TLS_CERT_FILE` |
| `SEED_ADMIN_EMAIL` | `admin` | Set before first boot |
| `SEED_ADMIN_PASSWORD` | `admin` | Set before first boot; production refuses `admin` |
| `ENV` | `development` | Set to `production` to enable startup safety checks |

The bootstrap admin is created only when the users table is empty. Changing `SEED_ADMIN_PASSWORD` later does not reset an existing account.

## Frontend

The frontend is a static Vite build served by nginx. Runtime API routing is controlled by the nginx configuration under `deploy/docker/`.

## SDR agent

| Variable | Default | Description |
|---|---|---|
| `BACKEND_URL` | `http://backend:8080` | Backend base URL |
| `AGENT_ID` | `sdr-agent-dev` | Agent identity used for RF readings |
| `AGENT_API_KEY` | unset | Preferred per-agent credential |
| `AGENT_TOKEN` | unset | Legacy/fallback human operator JWT |
| `AGENT_INSECURE_SKIP_VERIFY` | `false` | Disables TLS verification; development only |
| `FREQUENCY_HZ` | `437500000` | Center frequency |
| `CAPTURE_INTERVAL_SECONDS` | `10` | RF reading interval |
| `IQ_CAPTURE_DIR` | unset | Enables IQ recording |
| `IQ_CAPTURE_INTERVAL_SECONDS` | `60` | IQ recording interval |
| `IQ_CAPTURE_SAMPLES` | `8192` | Samples per IQ recording |
| `REPLAY_FILE` | unset | Capture metadata file for one-shot replay |
| `REPLAY_WINDOW_SAMPLES` | `65536` | Samples reduced into each replay reading |
| `REPLAY_DELAY_MS` | `300` | Delay between replay uploads |

Create an agent key as an admin through the dashboard or `POST /api/v1/agent-keys/`. The device self-report endpoint derives identity from `X-Agent-Key`; the body does not choose another agent.

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

`docker-compose.yml` currently consumes `JWT_SECRET`, `SEED_ADMIN_EMAIL`, `SEED_ADMIN_PASSWORD`, `AGENT_API_KEY`, `AGENT_TOKEN`, and `AGENT_INSECURE_SKIP_VERIFY` from the host or root `.env` file where configured.

Never commit a production `.env` file or private credentials.
