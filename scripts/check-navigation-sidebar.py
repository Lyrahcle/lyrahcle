#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]
dashboard = (ROOT / 'apps/frontend/src/components/Dashboard.jsx').read_text()
header = (ROOT / 'apps/frontend/src/components/Header.jsx').read_text()
navigation = (ROOT / 'apps/frontend/src/components/NavigationMenu.jsx').read_text()
translations = (ROOT / 'apps/frontend/src/lib/translations.js').read_text()
routes = (ROOT / 'apps/frontend/src/lib/routeTabs.ts').read_text()
required = [
    ('render persistent sidebar', '<NavigationMenu' in dashboard and 'fixed inset-y-0 left-0' in navigation),
    ('support collapsed desktop width', "collapsed ? 'w-20' : 'w-64'" in navigation and "navigationCollapsed ? 'lg:pl-20' : 'lg:pl-64'" in dashboard),
    ('support mobile drawer', 'mobileOpen &&' in navigation and 'onOpenNavigation' in header and 'setMobileNavigationOpen(true)' in dashboard),
    ('close mobile drawer after navigation', 'onCloseMobile?.()' in navigation),
    ('group primary destinations', "SECTION_ORDER = ['overview', 'mission', 'security', 'rf', 'platform', 'preferences']" in navigation),
    ('preserve permission-aware destinations', 'connectivityAccess ?' in dashboard and 'adminAccess ?' in dashboard and 'platformAdmin ?' in dashboard),
    ('preserve hash routing', "export const DEFAULT_TAB = 'orbital'" in routes and "'stations'" in routes),
    ('persist desktop preference', "localStorage.setItem('lyra_nav_collapsed'" in dashboard),
    ('provide multilingual navigation copy', 'navigationShellTranslations' in translations and 'RF operations' in translations and 'Operaciones RF' in translations),
]
failed = [name for name, ok in required if not ok]
if failed:
    print('navigation sidebar contract: ERROR: ' + ', '.join(failed))
    sys.exit(1)
print('navigation sidebar contract: OK')
