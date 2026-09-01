#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
checks = {
    ROOT / 'apps/frontend/src/components/SatelliteSecurityProfile.jsx': [
        'fetchOrbitalEvents(satellite.id, 168, 50)',
        "fetchIncidents(50, 0, { satelliteId: satellite.id })",
        'item.satellite_id === satellite.id',
        "identities.has(String(item.asset_value || '').toLowerCase())",
        'calculateSatelliteRisk(',
        "onNavigate('incidents')",
        "onNavigate('signal')",
    ],
    ROOT / 'apps/frontend/src/components/SatellitesPanel.jsx': [
        'onOpenProfile',
        "satelliteProfile.openProfile",
    ],
    ROOT / 'apps/frontend/src/components/Dashboard.jsx': [
        'selectedSatelliteId',
        '<SatelliteSecurityProfile',
        'onOpenProfile={(satellite) => setSelectedSatelliteId(satellite.id)}',
    ],
    ROOT / 'apps/frontend/src/lib/translations.js': [
        'const satelliteProfileTranslations = {',
        'exposureNote:',
    ],
}
errors = []
for path, tokens in checks.items():
    if not path.exists():
        errors.append(f'missing file: {path.relative_to(ROOT)}')
        continue
    text = path.read_text(encoding='utf-8')
    for token in tokens:
        if token not in text:
            errors.append(f'{path.relative_to(ROOT)} missing contract token: {token}')

profile_path = ROOT / 'apps/frontend/src/components/SatelliteSecurityProfile.jsx'
if profile_path.exists():
    profile_text = profile_path.read_text(encoding='utf-8')
    if profile_text.count('item.satellite_id === satellite.id') < 2:
        errors.append('SatelliteSecurityProfile.jsx must tenant-safely scope both incident and RF evidence to the selected satellite')
if errors:
    print('satellite security profile contract: FAIL')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)
print('satellite security profile contract: OK')
