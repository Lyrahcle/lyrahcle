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
