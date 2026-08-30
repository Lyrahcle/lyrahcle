#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()

CHECKS = {
    ROOT / 'apps/backend/cmd/server/main.go': [
        'Put("/{satelliteID}", handleUpdateSatellite',
        'cache.Delete(r.Context(), satellitesCacheKey)',
    ],
    ROOT / 'apps/backend/cmd/server/satellite_management_v132.go': [
        'func handleUpdateSatellite',
        'satellites.Update(',
        'audit.ActionSatelliteUpdated',
        'http.StatusConflict',
        'http.StatusNotFound',
    ],
    ROOT / 'apps/backend/internal/store/satellites.go': [
        'func (s *SatelliteStore) Update(',
        'UPDATE satellites',
        'RETURNING id, norad_id, name, tle_line1, tle_line2, tle_epoch, updated_at',
    ],
    ROOT / 'apps/frontend/src/lib/api.js': [
        'export async function updateSatellite',
        '/api/v1/satellites/${encodeURIComponent(satelliteId)}',
        "method: 'PUT'",
    ],
    ROOT / 'apps/frontend/src/components/SatellitesPanel.jsx': [
        'updateSatellite(',
        'canManage',
        "t('satelliteManagement.saveChanges')",
        'tle_line1',
        'tle_line2',
    ],
    ROOT / 'apps/frontend/src/components/Dashboard.jsx': [
        "canManage={hasPermission('orbital.manage')}",
        'onChanged={reloadSatellites}',
    ],
    ROOT / 'apps/backend/internal/audit/audit.go': [
        'ActionSatelliteUpdated',
        'satellite.updated',
    ],
    ROOT / 'apps/backend/cmd/server/openapi.yaml': [
        '/api/v1/satellites/{satelliteID}:',
        'summary: Update a tracked satellite and its TLE',
    ],
}

errors = []
for path, required in CHECKS.items():
    if not path.exists():
        errors.append(f'missing file: {path.relative_to(ROOT)}')
        continue
    text = path.read_text(encoding='utf-8')
    for token in required:
        if token not in text:
            errors.append(f'{path.relative_to(ROOT)} missing contract token: {token}')

if errors:
    print('satellite UI parity contract: FAIL')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)

print('satellite UI parity contract: OK')
