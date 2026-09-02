#!/usr/bin/env python3
import json
from pathlib import Path
import sys

ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]

FILES = {
    'component': ROOT / 'apps/frontend/src/components/CorrelationGraph.tsx',
    'model': ROOT / 'apps/frontend/src/lib/correlationGraph.ts',
    'layout': ROOT / 'apps/frontend/src/lib/correlationLayout3d.ts',
    'layout_test': ROOT / 'apps/frontend/src/lib/correlationLayout3d.test.ts',
    'scene': ROOT / 'apps/frontend/src/components/correlation/OrbitalScene.tsx',
    'node_mesh': ROOT / 'apps/frontend/src/components/correlation/NodeMesh.tsx',
    'links': ROOT / 'apps/frontend/src/components/correlation/EvidenceLinks.tsx',
    'route': ROOT / 'apps/frontend/src/components/correlation/RoutePath.tsx',
    'labels': ROOT / 'apps/frontend/src/components/correlation/SmartLabels.tsx',
    'clusters': ROOT / 'apps/frontend/src/components/correlation/ClusterMarkers.tsx',
    'rig': ROOT / 'apps/frontend/src/components/correlation/FocusRig.tsx',
    'context_panel': ROOT / 'apps/frontend/src/components/correlation/ContextPanel.tsx',
    'filters': ROOT / 'apps/frontend/src/components/correlation/FilterPopover.tsx',
    'search': ROOT / 'apps/frontend/src/components/correlation/NodeSearch.tsx',
    'node_index': ROOT / 'apps/frontend/src/components/correlation/NodeIndex.tsx',
    'capabilities': ROOT / 'apps/frontend/src/components/correlation/capabilities.ts',
    'theme': ROOT / 'apps/frontend/src/components/correlation/sceneTheme.ts',
    'dashboard': ROOT / 'apps/frontend/src/components/Dashboard.tsx',
    'navigation': ROOT / 'apps/frontend/src/components/dashboard/navigation.ts',
    'routes': ROOT / 'apps/frontend/src/lib/routeTabs.ts',
    'translations': ROOT / 'apps/frontend/src/lib/translations.js',
    'css': ROOT / 'apps/frontend/src/index.css',
    'package': ROOT / 'apps/frontend/package.json',
    'vite': ROOT / 'apps/frontend/vite.config.js',
    'tailwind': ROOT / 'apps/frontend/tailwind.config.js',
    'docs': ROOT / 'docs/correlation-constellation.md',
}

missing = [str(path.relative_to(ROOT)) for path in FILES.values() if not path.is_file()]
if missing:
    print('correlation constellation contract: ERROR')
    print('missing files: ' + ', '.join(missing))
    raise SystemExit(1)

text = {name: path.read_text(encoding='utf-8') for name, path in FILES.items()}
package = json.loads(text['package'])
dependencies = package.get('dependencies', {})

removed_svg_renderer = all(
    token not in text['component']
    for token in ('projectOrbitRing3D', 'orbitLayout3D', 'handleScenePointerDown', 'edgeCurve', '<svg')
)

checks = [
    ('typed graph model', "export type CorrelationNodeKind = 'satellite' | 'station' | 'agent' | 'incident' | 'rf' | 'asset' | 'cve' | 'program'" in text['model']),
    ('real identifier relationships', "reading.satellite_id" in text['model'] and "incident.satellite_id" in text['model'] and "incident.agent_id" in text['model'] and "alert.asset_value" in text['model']),
    ('exact asset exposure matching', "assetByValue.get(normalized(assetValue))" in text['model'] and "relation: 'observed'" in text['model']),
    ('explainable shortest path', 'shortestCorrelationPath' in text['model'] and 'routeOriginId' in text['component'] and 'routeTargetId' in text['component']),
    ('declared 3D runtime', all(name in dependencies for name in ('three', '@react-three/fiber', '@react-three/drei'))),
    ('React 18 compatible fiber major', dependencies.get('@react-three/fiber', '').lstrip('^~').startswith('8.')),
    ('legacy SVG renderer removed', removed_svg_renderer),
    ('real WebGL scene', '<Canvas' in text['scene'] and "from '@react-three/fiber'" in text['scene'] and 'OrbitalScene' in text['component']),
    ('scene is code split', 'lazyComponent(' in text['component'] and "import('./correlation/OrbitalScene')" in text['component']),
    ('three isolated in its own chunk', "name: 'three-vendor'" in text['vite'] and 'advancedChunks' in text['vite']),
    ('dedicated layout engine', 'buildOrbitalLayout' in text['layout'] and 'buildOrbitalLayout' in text['component']),
    ('layout independent of the renderer', 'three' not in text['layout'].split('\n')[0] and "from 'three'" not in text['layout']),
    ('deterministic layout without randomness', 'Math.random' not in text['layout'] and 'hashUnit' in text['layout']),
    ('physical relaxation for dense graphs', 'function relax(' in text['layout'] and 'separation' in text['layout']),
    ('layout survives zero links', 'linkCount' in text['layout'] and "contextTarget = Math.min(30" in text['component'] and "t('correlationGraph.awaitingEvidence'" in text['component']),
    ('kind clustering', 'sectorCenter' in text['layout'] and 'clusters' in text['layout'] and 'LAYOUT_KIND_ORDER' in text['layout']),
    ('layout unit coverage', 'buildOrbitalLayout' in text['layout_test'] and 'deterministic' in text['layout_test']),
    ('distinct geometry per node kind', all(token in text['node_mesh'] for token in ('torusGeometry', 'octahedronGeometry', 'cylinderGeometry', 'boxGeometry'))),
    ('thin luminous evidence links', 'lineSegments' in text['links'] and 'quadraticPoints' in text['links'] and 'AdditiveBlending' in text['links']),
    ('explainable route is the visual protagonist', 'RoutePath' in text['scene'] and 'pathEdgeKeys' in text['component'] and 'lineWidth' in text['route']),
    ('visual clustering by node kind', 'ClusterMarkers' in text['scene'] and 'showClusters' in text['component'] and 'minimumSize' in text['clusters']),
    ('numbered route hops', 'hopLabel' in text['route'] and "t('correlationGraph.hop'" in text['component'] and 'lyra-route-hop' in text['css']),
    ('scene geometry is disposed', 'dispose()' in text['links']),
    ('focus is announced to assistive technology', "aria-live=\"polite\"" in text['component'] and "t('correlationGraph.selectionAnnouncement'" in text['component']),
    ('smart labels only where they add value', 'selectedId' in text['labels'] and 'hoveredId' in text['labels'] and 'keyNodeLimit' in text['labels']),
    ('smooth focus camera', 'OrbitControls' in text['rig'] and 'lerp' in text['rig'] and 'focusToken' in text['component']),
    ('focus mode isolates the neighbourhood', 'focusMode' in text['component'] and 'neighborIds' in text['component'] and 'isolate' in text['scene']),
    ('premium search with autocomplete', "role=\"combobox\"" in text['search'] and "role=\"listbox\"" in text['search'] and 'ArrowDown' in text['search']),
    ('discreet HUD instead of large blocks', 'HudRail' in text['component'] and 'FilterPopover' in text['component'] and 'lyra-node-index' in text['node_index']),
    ('viewport-first stage', 'xl:min-h-[calc(100vh-118px)]' in text['component'] and 'xl:left-[186px] xl:right-[324px]' in text['component']),
    ('compact context panel', 'ContextPanel' in text['component'] and 'w-[300px]' in text['component']),
    ('time filtering', "export type WindowKey = '24h' | '7d' | '30d' | 'all'" in text['filters']),
    ('domain filtering', 'KIND_ORDER' in text['component'] and 'enabledKinds' in text['component']),
    ('severity filtering', 'SeverityFilter' in text['filters'] and 'severityMatches' in text['component']),
    ('depth filtering', "t('correlationGraph.depth')" in text['filters'] and 'onDepth' in text['component']),
    ('keyboard reachable nodes outside the canvas', 'data-correlation-node' in text['node_index'] and 'tabIndex={0}' in text['node_index']),
    ('graceful non-WebGL fallback', 'supportsWebGl' in text['capabilities'] and "t('correlationGraph.webglFallback')" in text['component']),
    ('reduced motion support', 'prefersReducedMotion' in text['capabilities'] and 'reducedMotion' in text['scene'] and '@media (prefers-reduced-motion: reduce)' in text['css']),
    ('theme-aware scene palette', 'readScenePalette' in text['theme'] and '--lyra-gold' in text['theme']),
    ('Tailwind scans TSX graph classes', './src/**/*.{js,jsx,ts,tsx}' in text['tailwind']),
    ('dedicated route', "'graph'," in text['routes'] and "id: 'graph'" in text['navigation']),
    ('lazy workspace', "lazyComponent(() => import('./CorrelationGraph.tsx'))" in text['dashboard'] and "activeTab === 'graph'" in text['dashboard']),
    ('permission-aware source loading', 'assetsAccess' in text['dashboard'] and 'incidentsRead' in text['dashboard'] and 'rfRead' in text['dashboard'] and 'alertsRead' in text['dashboard']),
    ('satellite profile navigation', 'onOpenSatellite' in text['component'] and 'setSelectedSatelliteId(id)' in text['dashboard']),
    ('multilingual flagship copy', 'correlationGraphTranslations' in text['translations'] and "translations[language].navigation.items.correlationGraph" in text['translations']),
    ('new HUD copy is localized', all(text['translations'].count(key) >= 4 for key in ('focusMode:', 'webglFallback:', 'nodeIndex:', 'filters:', 'selectionAnnouncement:'))),
    ('no graph runtime dependency', all(name not in text['package'].lower() for name in ['cytoscape', 'react-flow', 'reactflow', 'd3-force', 'vis-network'])),
    ('documented correlation limits', 'does not infer undocumented relationships' in text['docs'] and 'loaded evidence' in text['docs']),
    ('documented renderer rebuild', 'WebGL' in text['docs'] and 'three.js' in text['docs']),
]

failed = [name for name, ok in checks if not ok]
if failed:
    print('correlation constellation contract: ERROR')
    for name in failed:
        print(name)
    raise SystemExit(1)

print(f'correlation constellation contract: OK ({len(checks)} checks)')
