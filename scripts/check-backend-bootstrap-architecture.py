#!/usr/bin/env python3
from pathlib import Path
import re
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
errors = []
main_path = root / 'apps/backend/cmd/server/main.go'
main = main_path.read_text()

if len(main.splitlines()) > 400:
    errors.append(f'main.go remains too large: {len(main.splitlines())} lines')

for token in [
    'bootstrap.NewStores(pool)',
    'bootstrap.BuildAgentConfigSigner(cfg)',
    'bootstrap.ConnectRedis(cfg)',
    'bootstrap.NewHTTPServer(cfg, r)',
    'bootstrap.ConfigureAgentClientTLS(srv, cfg)',
    'newRouterWithBootstrap',
]:
    if token not in main:
        errors.append(f'main.go missing {token}')

for forbidden in ['store.NewAssetStore(', 'httprate.Limit(', 'func detectRFAnomalies(', 'func structuredRequestLogger(']:
    if forbidden in main:
        errors.append(f'main.go still owns {forbidden}')

function_count = len(re.findall(r'^func ', main, re.MULTILINE))
if function_count > 3:
    errors.append(f'main.go should only contain composition functions, found {function_count}')

checks = {
    'apps/backend/internal/bootstrap/stores.go': ['type Stores struct', 'func NewStores(', 'store.NewRBACStore(pool)', 'store.NewAgentIdentityStore(pool)'],
    'apps/backend/internal/bootstrap/security.go': ['func BuildAgentConfigSigner(', 'func BuildAgentCA(', 'func ConfigureAgentClientTLS(', 'func BuildWebAuthnHandler(', 'func ConnectRedis(', 'func RevocationCheck('],
    'apps/backend/internal/bootstrap/ratelimit.go': ['type RateLimiters struct', 'func BuildRateLimiters(', 'ratelimit.NewTenantLimiter('],
    'apps/backend/internal/bootstrap/services.go': ['type RequestServices struct', 'func BuildRequestServices('],
    'apps/backend/internal/bootstrap/http.go': ['func NewHTTPServer('],
    'apps/backend/internal/bootstrap/bootstrap_test.go': ['TestNewStoresBuildsAllRepositories', 'TestBuildRateLimitersWithoutRedis', 'TestNewHTTPServerKeepsOperationalTimeouts'],
    'apps/backend/cmd/server/http_helpers.go': ['func structuredRequestLogger(', 'func securityHeaders('],
    'apps/backend/cmd/server/mission_runtime.go': ['func syncTLEs(', 'func detectRFAnomalies(', 'func satellitesWithPositions('],
    'apps/backend/cmd/server/validation_audit.go': ['func validateDetectionRule(', 'func logAudit(', 'func clientIP('],
    'apps/backend/cmd/server/startup.go': ['func seedAdmin('],
}
for rel, tokens in checks.items():
    path = root / rel
    if not path.exists():
        errors.append(f'missing {rel}')
        continue
    text = path.read_text()
    for token in tokens:
        if token not in text:
            errors.append(f'{rel} missing {token}')

for rel, token in {
    'apps/backend/cmd/server/agent_config.go': 'func buildAgentConfigSigner(',
    'apps/backend/cmd/server/agent_identity.go': 'func buildAgentCA(',
    'apps/backend/cmd/server/auth.go': 'func buildWebAuthnHandler(',
}.items():
    if token in (root / rel).read_text():
        errors.append(f'{rel} still contains bootstrap helper {token}')

if errors:
    print('backend bootstrap architecture contract: FAIL')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)

print('backend bootstrap architecture contract: OK')
