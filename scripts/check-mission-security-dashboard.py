#!/usr/bin/env python3
import sys
from pathlib import Path


def fail(message: str) -> None:
    print(f'mission security dashboard contract: ERROR: {message}', file=sys.stderr)
    raise SystemExit(1)


def main() -> None:
    root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()

    dashboard_path = root / 'apps/frontend/src/components/Dashboard.tsx'
    query_path = root / 'apps/frontend/src/components/dashboard/dashboardQuery.ts'
    component_path = root / 'apps/frontend/src/components/MissionSecurityOperationsDashboard.jsx'
    test_path = root / 'apps/frontend/src/components/MissionSecurityOperationsDashboard.test.jsx'
    translations_path = root / 'apps/frontend/src/lib/translations.js'

    for path in (
        dashboard_path,
        query_path,
        component_path,
        test_path,
        translations_path,
    ):
        if not path.is_file():
            fail(f'missing required file: {path.relative_to(root)}')

    dashboard = dashboard_path.read_text()
    dashboard_query = query_path.read_text()
    component = component_path.read_text()
    translations = translations_path.read_text()

    required_dashboard_markers = [
        "lazyComponent<any>(() => import('./MissionSecurityOperationsDashboard.jsx'))",
        "const incidentsRead = hasPermission('incidents.read')",
        "const rfRead = hasPermission('rf.read')",
        "const alertsRead = hasPermission('alerts.read')",
        '<MissionSecurityOperationsDashboard',
        'onNavigate={onChangeTab}',
    ]

    for marker in required_dashboard_markers:
        if marker not in dashboard:
            fail(
                'Dashboard.tsx is missing permission-aware '
                f'integration marker: {marker}'
            )

    mission_start_marker = "if (activeTab === 'mission') {"
    orbital_start_marker = "\n  if (activeTab === 'orbital') {"

    if mission_start_marker not in dashboard_query:
        fail(
            'dashboardQuery.ts is missing mission loader: '
            f'{mission_start_marker}'
        )

    mission_start = dashboard_query.index(mission_start_marker)

    try:
        mission_end = dashboard_query.index(
            orbital_start_marker,
            mission_start,
        )
    except ValueError:
        fail(
            'dashboardQuery.ts mission loader is not bounded '
            'before orbital loader'
        )

    mission_section = dashboard_query[mission_start:mission_end]

    required_mission_markers = [
        "if (access.rf) [result.rfReadings, result.rfAnomalies] = await Promise.all(",
        "if (access.incidents) result.incidents = await optional(",
        "if (access.connectivity) result.agentIdentities = await optional(fetchAgentIdentities, [])",
        "if (access.alerts) result.assetAlerts = await optional(",
    ]

    for marker in required_mission_markers:
        if marker not in mission_section:
            fail(
                'dashboardQuery.ts mission loader is missing '
                f'permission-aware integration marker: {marker}'
            )

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
            fail(
                'MissionSecurityOperationsDashboard.jsx is missing '
                f'operator-view marker: {marker}'
            )

    direct_fetches = (
        'fetchIncidents(',
        'fetchRfAnomalies(',
        'fetchAgentIdentities(',
        'fetchAssetAlerts(',
    )

    if any(marker in component for marker in direct_fetches):
        fail(
            'the presentation component must not fetch '
            'protected resources directly'
        )

    for language in ('en', 'es', 'fr', 'de'):
        if f"{language}: {{" not in translations:
            fail(f'missing {language} translation dictionary')

    if (
        'missionSecurityTranslations' not in translations
        or 'translations[language].missionSecurity = extra'
        not in translations
    ):
        fail(
            'mission security translations are not registered '
            'for all supported languages'
        )

    print('mission security dashboard contract: OK')


if __name__ == '__main__':
    main()
