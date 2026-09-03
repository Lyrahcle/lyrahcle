#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]
required = {
    'apps/frontend/src/App.tsx': ['useHashRoute()', "import('./components/Dashboard.tsx')"],
    'apps/frontend/src/components/Dashboard.tsx': ['interface DashboardProps', 'activeTab: RouteTab', "import('./RBACPanel.tsx')", "import('./RateLimitPolicyPanel.tsx')", 'type CVEFilters', 'type IncidentFilters', 'const dashboardQuery = useQuery({', "lazyComponent<any>(() => import('./SatellitesPanel.jsx'))", 'const UntypedWorldMap = WorldMap as ComponentType<any>'],
    'apps/frontend/src/components/RBACPanel.tsx': ['interface RBACPanelProps', 'FormEvent<HTMLFormElement>', 'error instanceof Error'],
    'apps/frontend/src/components/RateLimitPolicyPanel.tsx': ['interface RateLimitState', 'useState<RateLimitState | null>'],
    'apps/frontend/src/components/SecretProviderPanel.tsx': ['interface SecretProviderState', 'useState<SecretProviderState | null>'],
    'apps/frontend/src/components/IncidentResponsePanel.tsx': ['interface IncidentPlaybook', 'IncidentPlaybook[]'],
    'apps/frontend/src/components/Panel.tsx': ['interface PanelProps', 'titleExtra?: ReactNode'],
    'apps/frontend/src/hooks/useHashRoute.ts': ['RouteTab', 'setTabInLocation', 'subscribeTabChanges'],
    'apps/frontend/src/components/dashboard/navigation.ts': ['DashboardNavigationAccess', 'buildDashboardNavigation', "id: 'settings'"],
    'apps/frontend/src/components/dashboard/dashboardQuery.ts': ['export async function fetchDashboardTab', 'const tasks: Promise<void>[] = []'],
}
legacy = [
    'apps/frontend/src/App.jsx',
    'apps/frontend/src/components/Dashboard.jsx',
    'apps/frontend/src/components/RBACPanel.jsx',
    'apps/frontend/src/components/RateLimitPolicyPanel.jsx',
    'apps/frontend/src/components/SecretProviderPanel.jsx',
    'apps/frontend/src/components/IncidentResponsePanel.jsx',
    'apps/frontend/src/components/Panel.jsx',
    'apps/frontend/src/components/permission_driven_ui_v119.test.jsx',
    'apps/frontend/src/components/navigation_menu_contract_v146.test.jsx',
]
errors = []
for relative, markers in required.items():
    path = ROOT / relative
    if not path.is_file():
        errors.append(f'missing {relative}')
        continue
    source = path.read_text()
    for marker in markers:
        if marker not in source:
            errors.append(f'{relative} missing {marker}')
for relative in legacy:
    if (ROOT / relative).exists():
        errors.append(f'legacy frontend path still present: {relative}')
app = (ROOT / 'apps/frontend/src/App.tsx').read_text() if (ROOT / 'apps/frontend/src/App.tsx').is_file() else ''
if 'subscribeTabChanges' in app or 'setTabInLocation' in app or 'getTabFromLocation' in app:
    errors.append('App.tsx still owns hash-routing mechanics instead of useHashRoute')
if errors:
    print('frontend TypeScript phase 2 contract: ERROR')
    print('\n'.join(errors))
    sys.exit(1)
print('frontend TypeScript phase 2 contract: OK')
