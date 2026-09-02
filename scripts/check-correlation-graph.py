#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]

FILES = {
    'component': ROOT / 'apps/frontend/src/components/CorrelationGraph.tsx',
    'model': ROOT / 'apps/frontend/src/lib/correlationGraph.ts',
    'dashboard': ROOT / 'apps/frontend/src/components/Dashboard.tsx',
    'navigation': ROOT / 'apps/frontend/src/components/dashboard/navigation.ts',
    'routes': ROOT / 'apps/frontend/src/lib/routeTabs.ts',
    'translations': ROOT / 'apps/frontend/src/lib/translations.js',
    'css': ROOT / 'apps/frontend/src/index.css',
    'package': ROOT / 'apps/frontend/package.json',
    'tailwind': ROOT / 'apps/frontend/tailwind.config.js',
    'docs': ROOT / 'docs/correlation-constellation.md',
}

missing = [str(path.relative_to(ROOT)) for path in FILES.values() if not path.is_file()]
if missing:
    print('correlation graph flagship contract: ERROR')
    print('missing files: ' + ', '.join(missing))
    raise SystemExit(1)

text = {name: path.read_text(encoding='utf-8') for name, path in FILES.items()}
checks = [
    ('typed graph model', "export type CorrelationNodeKind = 'satellite' | 'station' | 'agent' | 'incident' | 'rf' | 'asset' | 'cve' | 'program'" in text['model']),
    ('real identifier relationships', "reading.satellite_id" in text['model'] and "incident.satellite_id" in text['model'] and "incident.agent_id" in text['model'] and "alert.asset_value" in text['model']),
    ('exact asset exposure matching', "assetByValue.get(normalized(assetValue))" in text['model'] and "relation: 'observed'" in text['model']),
    ('explainable shortest path', 'shortestCorrelationPath' in text['model'] and 'routeOriginId' in text['component'] and 'routeTargetId' in text['component']),
    ('orbital focus layout', 'correlationNeighborhood' in text['component'] and 'orbitLayout3D' in text['component'] and 'correlationDistances' in text['model']),
    ('minimal 3D projection', 'projectOrbitRing3D' in text['component'] and 'CorrelationCamera' in text['model'] and 'cameraDistance = 980' in text['model']),
    ('orbital camera interaction', 'onPointerDown={handleScenePointerDown}' in text['component'] and 'onPointerMove={handleScenePointerMove}' in text['component'] and 'setPointerCapture' in text['component']),
    ('viewport-first graph stage', 'xl:min-h-[calc(100vh-118px)]' in text['component'] and 'absolute inset-0 xl:left-[164px] xl:right-[344px]' in text['component']),
    ('Tailwind scans TSX graph classes', './src/**/*.{js,jsx,ts,tsx}' in text['tailwind']),
    ('disconnected constellation context', 'contextTarget = Math.min(30' in text['component'] and "t('correlationGraph.awaitingEvidence'" in text['component']),
    ('depth ordering', 'points.sort((a, b) => b.z - a.z' in text['model'] and 'scale: perspective' in text['model']),
    ('time filtering', "type WindowKey = '24h' | '7d' | '30d' | 'all'" in text['component']),
    ('domain filtering', 'KIND_ORDER' in text['component'] and 'enabledKinds' in text['component']),
    ('severity filtering', 'SeverityFilter' in text['component'] and 'severityMatches' in text['component']),
    ('keyboard accessible nodes', 'role="button"' in text['component'] and 'tabIndex={0}' in text['component'] and "event.key !== 'Enter'" in text['component']),
    ('reduced motion support', '@media (prefers-reduced-motion: reduce)' in text['css'] and '.lyra-correlation-flow' in text['css']),
    ('distinctive visual fabric', 'lyra-correlation-stage' in text['css'] and 'projectOrbitRing3D' in text['component'] and 'edgeCurve' in text['component']),
    ('dedicated route', "'graph'," in text['routes'] and "id: 'graph'" in text['navigation']),
    ('lazy workspace', "lazyComponent(() => import('./CorrelationGraph.tsx'))" in text['dashboard'] and "activeTab === 'graph'" in text['dashboard']),
    ('permission-aware source loading', 'assetsAccess' in text['dashboard'] and 'incidentsRead' in text['dashboard'] and 'rfRead' in text['dashboard'] and 'alertsRead' in text['dashboard']),
    ('satellite profile navigation', 'onOpenSatellite' in text['component'] and 'setSelectedSatelliteId(id)' in text['dashboard']),
    ('multilingual flagship copy', 'correlationGraphTranslations' in text['translations'] and "translations[language].navigation.items.correlationGraph" in text['translations']),
    ('no graph runtime dependency', all(name not in text['package'].lower() for name in ['cytoscape', 'react-flow', 'reactflow', 'd3-force', 'vis-network'])),
    ('documented correlation limits', 'does not infer undocumented relationships' in text['docs'] and 'loaded evidence' in text['docs']),
]

failed = [name for name, ok in checks if not ok]
if failed:
    print('correlation graph flagship contract: ERROR')
    for name in failed:
        print(name)
    raise SystemExit(1)

print('correlation graph flagship contract: OK')
