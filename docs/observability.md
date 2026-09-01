# Observability

Lyrahcle exposes Prometheus metrics from the backend and CVE worker. Docker Compose can additionally provision Grafana with a read-only Lyrahcle overview dashboard.
Kubernetes deployments can expose the same metrics through optional scrape annotations or Prometheus Operator `ServiceMonitor`/`PrometheusRule` resources; see `docs/kubernetes-observability.md`.

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
- `lyrahcle_signing_keys{domain,status}`
- `lyrahcle_signing_key_rotation_due{domain}`
- `lyrahcle_signing_key_staged_pending{domain}`
- `lyrahcle_signing_key_warnings{domain,code}`
- `lyrahcle_signing_key_automation_total{domain,result}`

HTTP metrics use bounded route labels rather than raw URLs. Authentication metrics expose only result categories and never usernames, tokens, passwords, TOTP secrets, or request bodies.

## Alerts

Prometheus loads `deploy/prometheus/alerts.yml`. Alerts cover elevated server-error rates, p95 latency, authentication failures, unusually high SSE connection counts, TLE/correlation degradation, and remote-config signing-key lifecycle health. Signing-key alerts include no active key, overdue rotation, staged-key distribution backlog, grace expiry, and automation failures.

## Correlation

Every backend response carries `X-Request-ID`. Backend request logs use the same identifier, allowing an operator to correlate application errors with HTTP metrics and audit events without logging sensitive payloads.

Request IDs remain the always-on cross-layer correlation primitive. Optional distributed tracing is available through OpenTelemetry when an OTLP/HTTP endpoint is configured.

## OpenTelemetry tracing

Set `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT` to an OTLP/HTTP collector endpoint to enable backend tracing. Incoming W3C trace context is extracted, backend spans are created per HTTP request, and `X-Trace-ID` is returned for correlation. Production configuration requires an HTTPS collector endpoint. Do not attach credentials, tokens, reset links, TOTP secrets, or RF payloads as span attributes.
