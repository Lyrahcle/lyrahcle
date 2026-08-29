# Observability

Lyrahcle exposes Prometheus metrics from the backend and CVE worker. Docker Compose can additionally provision Grafana with a read-only Lyrahcle overview dashboard.

## Start the stack

```bash
docker compose --profile observability up --build
```

Prometheus is available on `http://localhost:9090`. Grafana is available on `http://localhost:3000`.

For local development, Grafana uses `admin` / `lyrahcle-dev` unless `GRAFANA_ADMIN_USER` and `GRAFANA_ADMIN_PASSWORD` are set. Never use the development password outside an isolated local environment.

## Backend metrics

The backend currently exports:

- `lyrahcle_http_requests_total`
- `lyrahcle_http_request_duration_seconds`
- `lyrahcle_auth_logins_total`
- `lyrahcle_sse_connections`

HTTP metrics use bounded route labels rather than raw URLs. Authentication metrics expose only result categories and never usernames, tokens, passwords, TOTP secrets, or request bodies.

## Alerts

Prometheus loads `deploy/prometheus/alerts.yml`. Initial alerts cover elevated server-error rates, p95 latency, authentication failures, and unusually high SSE connection counts.

## Correlation

Every backend response carries `X-Request-ID`. Backend request logs use the same identifier, allowing an operator to correlate application errors with HTTP metrics and audit events without logging sensitive payloads.

Distributed tracing remains a pre-v1 follow-up. Request IDs are the current cross-layer correlation primitive.
