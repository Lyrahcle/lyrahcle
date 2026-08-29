# Performance testing

Lyrahcle includes bounded parallel orbital-position calculation, targeted search indexes, short-lived response caches, and an authenticated global-search endpoint.

Use `scripts/load-test.sh` only against infrastructure you own or are explicitly authorized to test. The script requires `LOAD_TEST_ACK=1` and caps its default helper parameters.

```bash
LOAD_TEST_ACK=1 BASE_URL=http://localhost:8080 REQUESTS=5000 CONCURRENCY=50 ./scripts/load-test.sh
```

To test protected endpoints, provide a short-lived user token:

```bash
LOAD_TEST_ACK=1 TOKEN="$TOKEN" ./scripts/load-test.sh
```

Measure p50/p95/p99 latency, error rate, PostgreSQL CPU/locks, Redis latency, backend memory, and orbital calculation latency. Run tests against an isolated environment with production-like data volumes, not a shared production instance.
