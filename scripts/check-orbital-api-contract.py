#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1]).resolve() if len(sys.argv) > 1 else Path(__file__).resolve().parents[1]
openapi = (ROOT / 'apps/backend/cmd/server/openapi.yaml').read_text()
api_types = (ROOT / 'apps/frontend/src/lib/apiTypes.ts').read_text()
api = (ROOT / 'apps/frontend/src/lib/api.ts').read_text()
hook = (ROOT / 'apps/frontend/src/hooks/useMissionOrbitalContext.ts').read_text()

required_openapi = [
    '/api/v1/satellites/{satelliteID}/passes:',
    "$ref: '#/components/schemas/SatellitePassPredictionResponse'",
    'SatellitePassPredictionResponse:',
    'OrbitalEvent:',
]
required_types = [
    'export interface SatellitePassPredictionResponse',
    'passes: SatellitePass[]',
    'export interface OrbitalEvent',
    'export interface SatellitePassQuery',
]
required_api = [
    '): Promise<SatellitePassPredictionResponse>',
    '): Promise<OrbitalEvent[]>',
]
required_hook = ['Array.isArray(data?.passes)', 'setPasses(Array.isArray(data?.passes) ? data.passes : [])']

errors = []
for marker in required_openapi:
    if marker not in openapi:
        errors.append(f'OpenAPI missing orbital contract marker: {marker}')
for marker in required_types:
    if marker not in api_types:
        errors.append(f'frontend orbital API types missing marker: {marker}')
for marker in required_api:
    if marker not in api:
        errors.append(f'api.ts missing typed orbital signature: {marker}')
for marker in required_hook:
    if marker not in hook:
        errors.append(f'Mission orbital hook does not unwrap pass response: {marker}')

if errors:
    print('orbital API contract: ERROR')
    print('\n'.join(errors))
    sys.exit(1)
print('orbital API contract: OK')
