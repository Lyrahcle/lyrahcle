#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
required = [
    'apps/frontend/tsconfig.json',
    'apps/frontend/src/vite-env.d.ts',
    'apps/frontend/src/lib/api.ts',
    'apps/frontend/src/lib/auth.ts',
    'apps/frontend/src/lib/permissions.ts',
    'apps/frontend/src/lib/risk.ts',
    'apps/frontend/src/lib/groundStations.ts',
    'apps/frontend/src/lib/routeTabs.ts',
    'apps/frontend/src/lib/I18nContext.tsx',
    'apps/frontend/src/lib/lazyComponent.tsx',
]
legacy = [
    'apps/frontend/src/lib/api.js',
    'apps/frontend/src/lib/auth.js',
    'apps/frontend/src/lib/permissions.js',
    'apps/frontend/src/lib/risk.js',
    'apps/frontend/src/lib/groundStations.js',
    'apps/frontend/src/lib/routeTabs.js',
    'apps/frontend/src/lib/I18nContext.jsx',
    'apps/frontend/src/lib/lazyComponent.jsx',
]
missing = [item for item in required if not (ROOT / item).is_file()]
present_legacy = [item for item in legacy if (ROOT / item).exists()]
package = (ROOT / 'apps/frontend/package.json').read_text()
local_gate = (ROOT / 'scripts/test-local-v1.sh').read_text()
ci = (ROOT / '.github/workflows/ci.yml').read_text()
errors = []
if missing:
    errors.append('missing: ' + ', '.join(missing))
if present_legacy:
    errors.append('legacy files still present: ' + ', '.join(present_legacy))
if '"typecheck"' not in package or 'typescript' not in package:
    errors.append('frontend TypeScript dependencies or typecheck script missing')
if 'npm run typecheck' not in local_gate:
    errors.append('local gate does not run typecheck')
if 'npm run typecheck' not in ci:
    errors.append('CI does not run typecheck')
if errors:
    print('typescript migration contract: ERROR')
    print('\n'.join(errors))
    sys.exit(1)
print('typescript migration contract: OK')
