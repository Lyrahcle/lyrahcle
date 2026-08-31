#!/usr/bin/env python3
"""Regression guard for the pre-v1.0 sensitive-endpoint authorization review.

For every HTTP route registered in cmd/server/*.go, this verifies that one
of the following holds, using brace-depth scoping rather than a fixed line
window (so a guard on a sibling route can never be mistaken for a guard on
this one):

  - an inline `.With(...)` role/permission/credential middleware on the
    same line as the route registration,
  - an enclosing `r.Route(...)` block whose own `r.Use(...)` (inside that
    same block, not a sibling block) applies a role/permission middleware,
  - an inline authorization check inside the handler's own function body
    (permission check, mTLS client-certificate check, or single-use
    enrollment-token validation), or
  - the route is explicitly allowlisted below as intentionally public,
    with a one-line reason.

This does not replace human judgement on new routes -- it fails loudly
when a route matches none of the known patterns, so a human decides
whether it needs a guard or belongs in PUBLIC_ROUTES.
"""
from pathlib import Path
import re
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.')
server_dir = root / 'apps/backend/cmd/server'

AUTH_MARKERS = [
    'auth.RequireRoleWithFallback(',
    'auth.RequirePermissionWithFallback(',
    'auth.RequireAnyPermissionWithFallback(',
    'requirePlatformAdmin',
    'requireRequestPermission(',
    'credentialAuth',
    'agentAuth',
    'meAgentAuth',
    'agentRead',
    'agentManage',
    'r.TLS',
    'PeerCertificates',
    'hashEnrollmentToken(',
]

# Route-path suffix -> reason it is intentionally public or uses a
# non-JWT-session auth mechanism validated inline in the handler body.
PUBLIC_ROUTES = {
    '/healthz': 'liveness probe',
    '/readyz': 'readiness probe',
    '/openapi.yaml': 'public API spec',
    '/docs': 'public API docs UI',
    '/ping': 'unauthenticated connectivity check',
    '/login': 'credential exchange, rate-limited by loginLimiter',
    '/webauthn/login/start': 'pre-session WebAuthn ceremony, rate-limited',
    '/webauthn/login/finish': 'pre-session WebAuthn ceremony, rate-limited',
    '/password-reset/request': 'pre-session recovery flow, rate-limited',
    '/password-reset/complete': 'pre-session recovery flow, rate-limited, token-bound',
    '/agents/enroll': 'protected by hashed single-use enrollment token, not a session',
}

# Only matches an actual chi route registration (trimmed line starts with
# `r.` and, optionally, a `.With(...)` chain, directly followed by the verb
# call). Deliberately excludes unrelated calls that also end in
# `.Get("...")`, such as `r.Header.Get(...)` or `r.URL.Query().Get(...)`.
ROUTE_RE = re.compile(r'^r\.(?:With\((?P<with>.*)\)\.)?(Get|Post|Put|Delete|Patch)\(\s*"([^"]+)"')


class RouteBlock:
    """An r.Route(...) block: tracks whether its own r.Use(...) gates it."""

    def __init__(self, start_depth):
        self.start_depth = start_depth
        self.gated = False


# A local variable assigned from one of these is itself an auth gate, even
# though its own name (e.g. `read`, `manage`, `agentRead`) doesn't appear in
# AUTH_MARKERS. Catches indirection like `read := auth.RequireAnyPermission...`
# followed by `.With(read)`.
GATE_ASSIGNMENT_RE = re.compile(
    r'^(\w+)\s*:=\s*(?:auth\.\w*(?:RequireRoleWithFallback|RequirePermissionWithFallback|'
    r'RequireAnyPermissionWithFallback)\(|requirePlatformAdmin\b|credentialAuth\b)'
)


def line_is_gated(text, gate_vars):
    if any(mk in text for mk in AUTH_MARKERS):
        return True
    tokens = set(re.findall(r'\w+', text))
    return bool(tokens & gate_vars)


def scan_file(path, root):
    lines = path.read_text().splitlines()

    gate_vars = set()
    for line in lines:
        gm = GATE_ASSIGNMENT_RE.match(line.strip())
        if gm:
            gate_vars.add(gm.group(1))

    depth = 0
    blocks = []      # stack of enclosing RouteBlock frames
    findings = []     # handlers whose gate status is still undetermined

    for i, line in enumerate(lines):
        if blocks and 'r.Use(' in line and line_is_gated(line, gate_vars):
            blocks[-1].gated = True

        m = ROUTE_RE.match(line.strip())
        depth_before = depth
        depth += line.count('{') - line.count('}')

        if m:
            route_path = m.group(3)
            public = any(route_path.endswith(pub) for pub in PUBLIC_ROUTES)
            same_line_gate = bool(m.group('with')) and line_is_gated(m.group('with'), gate_vars)
            block_gate = blocks[-1].gated if blocks else False
            if not public and not same_line_gate and not block_gate:
                findings.append({
                    'line': i + 1,
                    'route': route_path,
                    'handler_start_depth': depth_before,
                    'body': [],
                    'open': depth > depth_before,
                })

        for f in findings:
            if f['open']:
                f['body'].append(line)
                if depth <= f['handler_start_depth']:
                    f['open'] = False

        if 'r.Route(' in line:
            blocks.append(RouteBlock(depth_before))
        while blocks and depth <= blocks[-1].start_depth:
            blocks.pop()

    failures = []
    for f in findings:
        body_text = "\n".join(f['body'])
        if line_is_gated(body_text, gate_vars):
            continue
        failures.append(
            f"{path.relative_to(root)}:{f['line']}: {f['route']!r} has no "
            "recognizable authorization gate (inline, enclosing r.Use, or "
            "in-handler check)"
        )
    return failures


failures = []
scanned = 0
for path in sorted(server_dir.glob('*.go')):
    if path.name.endswith('_test.go'):
        continue
    scanned += 1
    failures.extend(scan_file(path, root))

if failures:
    print('sensitive-endpoint authorization review: FAIL')
    for f in failures:
        print(' -', f)
    print()
    print('If a route is intentionally public or uses a non-standard auth')
    print('mechanism, add it to PUBLIC_ROUTES in this script with a reason.')
    raise SystemExit(1)

print(f'sensitive-endpoint authorization review: OK ({scanned} files scanned)')
