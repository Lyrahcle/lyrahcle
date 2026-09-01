#!/usr/bin/env python3
from pathlib import Path
import sys

from check_source_utils import read_source_tree

ROOT = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()

SERVER_REQUIRED = [
    '"assets.manage", isTokenRevoked)).Put("/{assetID}"',
    '"assets.manage", isTokenRevoked)).Delete("/{assetID}"',
    '"programs.manage", isTokenRevoked)).Put("/{programID}"',
    '"programs.manage", isTokenRevoked)).Delete("/{programID}"',
    'audit.ActionAssetUpdated',
    'audit.ActionAssetDeleted',
    'audit.ActionProgramUpdated',
    'audit.ActionProgramDeleted',
]

CHECKS = {
    ROOT / 'apps/backend/internal/store/assets.go': [
        'p.organization_id=$1',
        'UpdateInOrganization',
        'DeleteInOrganization',
    ],
    ROOT / 'apps/backend/internal/store/programs.go': [
        'UpdateInOrganization',
        'DeleteInOrganization',
        'organization_id=$1',
    ],
    ROOT / 'apps/frontend/src/lib/api.ts': [
        'export async function updateAsset',
        'export async function deleteAsset',
        'export async function updateProgram',
        'export async function deleteProgram',
    ],
    ROOT / 'apps/frontend/src/components/AssetsTable.jsx': [
        "hasPermission('assets.manage')",
        'updateAsset(',
        'deleteAsset(',
    ],
    ROOT / 'apps/frontend/src/components/ProgramManager.jsx': [
        "hasPermission('programs.manage')",
        'updateProgram(',
        'deleteProgram(',
    ],
}

errors = []
server_dir = ROOT / 'apps/backend/cmd/server'
server_text = read_source_tree(server_dir)
if not server_text:
    errors.append('missing server source directory: apps/backend/cmd/server')
else:
    for token in SERVER_REQUIRED:
        if token not in server_text:
            errors.append(f'backend server sources missing contract token: {token}')
for path, required in CHECKS.items():
    if not path.exists():
        errors.append(f'missing file: {path.relative_to(ROOT)}')
        continue
    text = path.read_text(encoding='utf-8')
    for token in required:
        if token not in text:
            errors.append(f'{path.relative_to(ROOT)} missing contract token: {token}')

if errors:
    print('asset/program UI parity contract: FAIL')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)

print('asset/program UI parity contract: OK')
