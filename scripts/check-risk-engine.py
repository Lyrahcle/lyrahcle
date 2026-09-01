#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(__file__).resolve().parents[1]
risk = (ROOT / 'apps/frontend/src/lib/risk.ts').read_text()
mission = (ROOT / 'apps/frontend/src/components/MissionSecurityOperationsDashboard.jsx').read_text()
profile = (ROOT / 'apps/frontend/src/components/SatelliteSecurityProfile.jsx').read_text()
satellites = (ROOT / 'apps/frontend/src/components/SatellitesPanel.jsx').read_text()
stations = (ROOT / 'apps/frontend/src/components/GroundStationOperations.jsx').read_text()
required = [
    ('shared mission risk', 'calculateMissionRisk' in risk and 'calculateMissionRisk' in mission),
    ('shared satellite risk', 'calculateSatelliteRisk' in risk and 'calculateSatelliteRisk' in profile and 'calculateSatelliteRisk' in satellites),
    ('shared ground station risk', 'calculateGroundStationRisk' in risk and 'calculateGroundStationRisk' in stations),
    ('bounded score', 'Math.min(100' in risk),
    ('explainable factors', 'factors' in risk and 'domain' in risk),
    ('mission risk UI', "missionSecurity.metric.riskScore" in mission),
    ('satellite table risk', "satelliteManagement.risk" in satellites),
]
failed = [name for name, ok in required if not ok]
if failed:
    print('risk engine contract: ERROR: ' + ', '.join(failed))
    sys.exit(1)
print('risk engine contract: OK')
