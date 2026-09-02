#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
checks = {
    'apps/backend/internal/metrics/metrics.go': [
        'lyrahcle_db_queries_total',
        'lyrahcle_db_query_duration_seconds',
        'lyrahcle_db_rows_total',
        'lyrahcle_response_cache_requests_total',
        'type DBQueryTracer struct{}',
        '[]string{"route", "operation"}',
    ],
    'apps/backend/internal/db/db.go': ['cfg.ConnConfig.Tracer = metrics.DBQueryTracer{}'],
    'apps/backend/internal/respcache/respcache.go': [
        'RecordResponseCacheLookup(ctx, "hit")',
        'RecordResponseCacheLookup(ctx, "miss")',
        'RecordResponseCacheLookup(ctx, "error")',
        'RecordResponseCacheLookup(ctx, "bypass")',
    ],
    'deploy/prometheus/alerts.yml': [
        'LyrahcleHighDatabaseP95Latency',
        'LyrahcleLowResponseCacheHitRatio',
    ],
    'docs/performance-testing.md': [
        'backend HTTP p95: **< 1 s**',
        'PostgreSQL query p95: **< 250 ms**',
        'response-cache hit ratio: **>= 70%**',
        'EXPLAIN (ANALYZE, BUFFERS)',
    ],
}
errors=[]
for rel, needles in checks.items():
    path=root/rel
    if not path.exists():
        errors.append(f'missing {rel}')
        continue
    text=path.read_text(encoding='utf-8')
    for needle in needles:
        if needle not in text:
            errors.append(f'{rel}: missing {needle!r}')
metrics=(root/'apps/backend/internal/metrics/metrics.go').read_text(encoding='utf-8')
for forbidden in ('[]string{"sql"', '[]string{"key"', '[]string{"satellite_id"'):
    if forbidden in metrics:
        errors.append(f'high-cardinality metric label found: {forbidden}')
if errors:
    print('performance observability contract: FAIL', file=sys.stderr)
    for error in errors:
        print(f'- {error}', file=sys.stderr)
    raise SystemExit(1)
print('performance observability contract: OK')
