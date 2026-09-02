# Performance testing

Lyrahcle limits background dashboard traffic by polling only the active workspace and opening live SSE streams only while their corresponding tab is visible. PostgreSQL includes targeted search indexes plus composite indexes for RF, audit, satellite, recovery, and WebAuthn access patterns.

Use `scripts/load-test.sh` only against infrastructure you own or are explicitly authorized to test. The script requires `LOAD_TEST_ACK=1` and caps request/concurrency parameters.

Request-count mode:

```bash
LOAD_TEST_ACK=1 BASE_URL=http://localhost:8080 REQUESTS=5000 CONCURRENCY=50 ./scripts/load-test.sh
```

Duration mode:

```bash
LOAD_TEST_ACK=1 BASE_URL=http://localhost:8080 DURATION=60s CONCURRENCY=50 ./scripts/load-test.sh
```

Protected endpoints require a short-lived user token:

```bash
LOAD_TEST_ACK=1 TOKEN="$TOKEN" OUTPUT_DIR=artifacts/load ./scripts/load-test.sh
```

The authenticated suite covers satellites, global search, and incidents in addition to the public health endpoint. Store results outside the repository when they may reveal production topology or timing data.

Measure p50/p95/p99 latency, HTTP error rate, PostgreSQL CPU/locks, Redis latency, backend memory, SSE connection count, and orbital calculation latency. Run tests against an isolated environment with production-like data volumes rather than a shared production instance.

For SQL tuning, compare representative query plans before and after index changes with `EXPLAIN (ANALYZE, BUFFERS)` in a non-production database populated with realistic data volumes. Avoid running `ANALYZE` load experiments on a live mission database.

## Runtime performance instrumentation and initial budgets

The backend exports bounded-cardinality Prometheus metrics for the critical request path. Route labels use Chi route templates (for example `/api/v1/incidents/{id}`), never raw URLs, SQL text, resource IDs, satellite IDs, or cache keys.

| Metric | Meaning |
| --- | --- |
| `lyrahcle_http_request_duration_seconds` | End-to-end HTTP latency by method and route |
| `lyrahcle_db_queries_total` | PostgreSQL query count by route and normalized operation |
| `lyrahcle_db_query_duration_seconds` | PostgreSQL query latency by route and normalized operation |
| `lyrahcle_db_rows_total` | Rows returned or affected by PostgreSQL commands |
| `lyrahcle_response_cache_requests_total` | Response-cache lookups classified as `hit`, `miss`, `error`, or `bypass` |

Initial production guardrails are intentionally conservative until representative production-like load tests establish tighter service-level objectives:

- backend HTTP p95: **< 1 s** over 10 minutes;
- PostgreSQL query p95: **< 250 ms** over 10 minutes;
- response-cache hit ratio: **>= 70%** over 15 minutes when cached read paths are receiving traffic;
- HTTP 5xx rate: retain the existing alert threshold and investigate together with query latency and cache results.

These are operational warning budgets, not claims that every individual request must remain under the threshold. Use per-route PromQL when diagnosing a specific workspace or API path.

`lyrahcle_db_rows_total` reports rows returned or affected because pgx/PostgreSQL does not expose a trustworthy generic "rows examined" value for every query at runtime. For query-plan row examination, buffer pressure, and planner estimates, use `EXPLAIN (ANALYZE, BUFFERS)` against a production-like non-production database as described above. Do not label Prometheus series with raw SQL to approximate this: that would create unbounded cardinality and can leak sensitive query detail.

Useful PromQL examples:

```promql
sum by (route) (rate(lyrahcle_db_queries_total[5m]))
```

```promql
histogram_quantile(0.95, sum by (le, route) (rate(lyrahcle_db_query_duration_seconds_bucket[5m])))
```

```promql
sum by (route) (rate(lyrahcle_response_cache_requests_total{result="hit"}[10m]))
/
clamp_min(sum by (route) (rate(lyrahcle_response_cache_requests_total{result=~"hit|miss"}[10m])), 0.001)
```
