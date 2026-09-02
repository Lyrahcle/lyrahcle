#!/usr/bin/env python3
import sys
from pathlib import Path


def fail(message: str) -> None:
    print(f'mission security dashboard contract: ERROR: {message}', file=sys.stderr)
    raise SystemExit(1)


def main() -> None:
    root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
    dashboard_path = root / 'apps/frontend/src/components/Dashboard.tsx'
    component_path = root / 'apps/frontend/src/components/MissionSecurityOperationsDashboard.jsx'
    test_path = root / 'apps/frontend/src/components/MissionSecurityOperationsDashboard.test.jsx'
    translations_path = root / 'apps/frontend/src/lib/translations.js'

    for path in (dashboard_path, component_path, test_path, translations_path):
        if not path.is_file():
            fail(f'missing required file: {path.relative_to(root)}')

    dashboard = dashboard_path.read_text()
    component = component_path.read_text()
    translations = translations_path.read_text()

    required_dashboard_markers = [
        "lazyComponent<any>(() => import('./MissionSecurityOperationsDashboard.jsx'))",
        "const incidentsRead = hasPermission('incidents.read')",
        "const rfRead = hasPermission('rf.read')",
        "const alertsRead = hasPermission('alerts.read')",
        'if (incidentsRead) {',
        'if (rfRead) requests.push(safeFetch(fetchRfAnomalies, setRfAnomalies))',
        'if (connectivityAccess) requests.push(safeFetch(fetchAgentIdentities, setAgentIdentities))',
        'if (alertsRead) {',
        '<MissionSecurityOperationsDashboard',
        'onNavigate={onChangeTab}',
    ]
    for marker in required_dashboard_markers:
        if marker not in dashboard:
            fail(f'Dashboard.tsx is missing permission-aware integration marker: {marker}')

    required_component_markers = [
        'missionSecurity.title',
        'INACTIVE_INCIDENT_STATUSES',
        'AGENT_ONLINE_WINDOW_MS',
        'buildActivity',
        'missionSecurity.queueTitle',
        'missionSecurity.activityTitle',
        'missionSecurity.partialAccess',
    ]
    for marker in required_component_markers:
        if marker not in component:
            fail(f'MissionSecurityOperationsDashboard.jsx is missing operator-view marker: {marker}')

    if "fetchIncidents(" in component or "fetchRfAnomalies(" in component or "fetchAgentIdentities(" in component or "fetchAssetAlerts(" in component:
        fail('the presentation component must not fetch protected resources directly')

    for language in ('en', 'es', 'fr', 'de'):
        if f"{language}: {{" not in translations:
            fail(f'missing {language} translation dictionary')
    if 'missionSecurityTranslations' not in translations or 'translations[language].missionSecurity = extra' not in translations:
        fail('mission security translations are not registered for all supported languages')

    print('mission security dashboard contract: OK')


if __name__ == '__main__':
    main()
