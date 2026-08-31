#!/usr/bin/env python3
"""CSRF regression guard.

Lyrahcle's authentication is bearer-token only: the JWT is sent in the
Authorization header and held in the frontend's sessionStorage
(see docs/browser-and-cors-security.md). No handler sets an authentication
cookie, so cross-site request forgery does not apply to the current
session model -- a forged cross-origin request cannot carry a bearer token
the browser attaches automatically the way it would a cookie.

This script is the enforcement half of that fact: it fails the build the
moment any backend handler starts setting a cookie, so a cookie-based
session can never be introduced silently without someone also adding CSRF
protection (double-submit token, SameSite=Strict, or an equivalent) and
updating this allowlist with a justification.
"""
from pathlib import Path
import re
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.')
server_dir = root / 'apps/backend'

# Handlers allowed to set a cookie, and why CSRF is still not a concern for
# them. Empty on purpose: today nothing in the backend sets a cookie.
ALLOWED_COOKIE_SETTERS = {
    # 'apps/backend/cmd/server/example.go': 'reason CSRF does not apply here',
}

COOKIE_RE = re.compile(r'http\.SetCookie\(|http\.Cookie\{')

failures = []
scanned = 0
for path in sorted(server_dir.rglob('*.go')):
    if path.name.endswith('_test.go'):
        continue
    scanned += 1
    text = path.read_text()
    if COOKIE_RE.search(text):
        rel = str(path.relative_to(root))
        if rel not in ALLOWED_COOKIE_SETTERS:
            failures.append(
                f"{rel}: sets a cookie but is not in ALLOWED_COOKIE_SETTERS. "
                "Cookie-based auth requires CSRF protection (double-submit "
                "token or SameSite=Strict) before this can ship -- add it, "
                "then allowlist this file here with a one-line reason."
            )

if failures:
    print('CSRF/cookie-auth regression guard: FAIL')
    for f in failures:
        print(' -', f)
    raise SystemExit(1)

print(f'CSRF/cookie-auth regression guard: OK ({scanned} files scanned, no cookie-based auth introduced)')
