#!/usr/bin/env python3
from pathlib import Path
import sys

from check_source_utils import read_source_tree

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.')
checks = []

def require(path, needle, label):
    target = root / path
    if not target.exists():
        checks.append(f"{label}: missing file {path}")
        return
    text = target.read_text()
    if needle not in text:
        checks.append(f"{label}: missing {needle!r} in {path}")

def require_server(needle, label):
    text = read_source_tree(root / 'apps/backend/cmd/server')
    if needle not in text:
        checks.append(f"{label}: missing {needle!r} in backend server sources")

require('apps/backend/internal/ratelimit/tenant.go', 'AllowHuman', 'human tenant limiter')
require('apps/backend/internal/ratelimit/tenant.go', 'AllowAgent', 'agent tenant limiter')
require('apps/backend/internal/ratelimit/tenant.go', 'organizationID.String() + ":" + userID.String()', 'actor key includes tenant')
require('apps/backend/internal/ratelimit/tenant.go', 'sha256.Sum256([]byte(subject))', 'redis keys hash subjects')
require('apps/backend/internal/ratelimit/tenant.go', 'tenant_agent', 'machine namespace isolated')
require_server('authenticatedTenantRateLimit(cfg.JWTSecret, cfg.JWTPreviousSecret, tenantLimiter)', 'authenticated API limiter installed')

require_server('enforceAgentRateLimit(w, r, tenantLimiter', 'agent device/RF limiter')
require('apps/backend/cmd/server/agent_config_v91.go', 'enforceAgentRateLimit(w, r, tenantLimiter', 'remote config machine limiter')
require('apps/backend/cmd/server/agent_identity_v90.go', 'enforceAgentRateLimit(w, r, tenantLimiter', 'certificate rotation limiter')
require('apps/backend/internal/config/config.go', 'RATE_LIMIT_TENANT_READ_PER_MINUTE', 'tenant read configuration')
require('apps/backend/internal/config/config.go', 'RATE_LIMIT_TENANT_AGENT_PER_MINUTE', 'tenant agent configuration')
require('apps/backend/internal/config/config.go', 'RATE_LIMIT_PLATFORM_CEILING_PER_MINUTE', 'platform ceiling configuration')
require('apps/backend/internal/metrics/metrics.go', 'lyrahcle_rate_limit_decisions_total', 'rate limit metrics')
require('apps/backend/cmd/server/rate_limit_status_v138.go', '/platform/rate-limits', 'platform rate-limit status endpoint')
require('apps/frontend/src/components/RateLimitPolicyPanel.jsx', 'fetchPlatformRateLimits', 'platform rate-limit UI')
require('apps/frontend/src/components/Dashboard.jsx', '<RateLimitPolicyPanel />', 'platform UI integration')
require('apps/frontend/src/lib/api.ts', '/api/v1/platform/rate-limits', 'rate-limit API client')
require('apps/backend/cmd/server/rate_limit_v138_test.go', 'TestAuthenticatedTenantRateLimitIsolatesNoisyNeighbor', 'HTTP noisy-neighbor test')
require('apps/backend/internal/ratelimit/tenant_test.go', 'TestHumanAndAgentBudgetsAreIndependent', 'human/agent isolation test')

cfg = (root / 'apps/backend/internal/config/config.go').read_text()
if '50000' not in cfg or '12000' not in cfg:
    checks.append('defaults: expected platform ceiling to remain comfortably above tenant agent budget')

if checks:
    print('tenant rate-limit contract: FAIL')
    for item in checks:
        print(' -', item)
    raise SystemExit(1)
print('tenant rate-limit contract: OK')
