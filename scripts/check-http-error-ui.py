#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
api = (root / 'apps/frontend/src/lib/api.ts').read_text()
app = (root / 'apps/frontend/src/App.jsx').read_text()
dash = (root / 'apps/frontend/src/components/Dashboard.jsx').read_text()
banner = (root / 'apps/frontend/src/components/ApiStatusBanner.jsx').read_text()
login = (root / 'apps/frontend/src/components/Login.jsx').read_text()
translations = (root / 'apps/frontend/src/lib/translations.js').read_text()
runner = (root / 'scripts/test-local-v1.sh').read_text()

checks = {
    'typed ApiError': 'class ApiError extends Error' in api,
    '401 typed session expiry': 'class UnauthorizedError extends ApiError' in api and "kind: 'unauthorized'" in api,
    '403 mapping': "status === 403" in api and "'forbidden'" in api,
    '404 mapping': "status === 404" in api and "'not_found'" in api,
    '409 mapping': "status === 409" in api and "'conflict'" in api,
    '429 mapping': "status === 429" in api and "'rate_limited'" in api and 'Retry-After' in api,
    '5xx mapping': "status >= 500" in api and "'server'" in api,
    'network mapping': "kind: 'network'" in api and 'trackedFetch' in api,
    'global event': 'API_ERROR_EVENT' in api and 'dispatchEvent' in api,
    'banner mounted': '<ApiStatusBanner' in dash and 'API_ERROR_EVENT' in dash,
    'manual retry': 'retryFailedRequests' in dash and 'retryNonce' in dash,
    '429 polling cooldown': 'rateLimitedUntilRef' in dash,
    'startup not blank': 'if (!permissionsReady)' in app and 'return null' not in app,
    'startup retry': 'permissionsRetry' in app and '<ApiStatusBanner' in app,
    'session notice': "notice === 'session_expired'" in login and 'sessionExpired' in translations,
    'localized states': all(f'{key}:' in translations for key in ['forbidden', 'not_found', 'conflict', 'rate_limited', 'server', 'network']),
    'local gate': 'check-contracts.sh' in runner,
}

missing = [name for name, ok in checks.items() if not ok]
if missing:
    for name in missing:
        print(f'http error UI contract missing: {name}', file=sys.stderr)
    raise SystemExit(1)

if 'if (!res.ok) throw new Error' in api:
    print('http error UI contract: raw response errors still discard HTTP metadata', file=sys.stderr)
    raise SystemExit(1)

print('HTTP error UI contract: OK')
