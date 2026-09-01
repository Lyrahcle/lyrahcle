#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]
dashboard = (ROOT / 'apps/frontend/src/components/Dashboard.jsx').read_text()
component = (ROOT / 'apps/frontend/src/components/GroundStationOperations.jsx').read_text()
world_map = (ROOT / 'apps/frontend/src/components/WorldMap.jsx').read_text()
risk = (ROOT / 'apps/frontend/src/lib/risk.js').read_text()
helpers = (ROOT / 'apps/frontend/src/lib/groundStations.js').read_text()
routes = (ROOT / 'apps/frontend/src/lib/routeTabs.js').read_text()
translations = (ROOT / 'apps/frontend/src/lib/translations.js').read_text()
required = [
    ('expose stations route', "'stations'" in routes and "id: 'stations'" in dashboard),
    ('permission-aware station loading', "activeTab === 'stations' && connectivityAccess" in dashboard and 'if (orbitalAccess)' in dashboard and 'if (rfRead)' in dashboard and 'if (incidentsRead)' in dashboard),
    ('render ground station operations', 'GroundStationOperations' in dashboard and 'GroundStationOperations' in component),
    ('require trusted station coordinates', 'stationCoordinates(agent)' in component and 'stationCoordinates(agent)' in helpers),
    ('calculate current satellite visibility', 'visibleSatellites' in component and 'satelliteElevationDeg' in helpers),
    ('scope evidence to station agent', 'item.agent_id === agent.agent_id' in component),
    ('use shared station risk', 'calculateGroundStationRisk' in component and 'calculateGroundStationRisk' in risk),
    ('keep risk permission-aware', 'incidents: access.incidents ? evidence.incidents : []' in component and 'anomalies: access.rf ? evidence.anomalies : []' in component),
    ('show stations on reviewed world map', 'stationMarkers' in world_map and 'onStationSelect' in world_map),
    ('provide multilingual copy', 'groundStationTranslations' in translations and "tabs.stations = extra.tab" in translations),
]
failed = [name for name, ok in required if not ok]
if failed:
    print('ground station operations contract: ERROR: ' + ', '.join(failed))
    sys.exit(1)
print('ground station operations contract: OK')
