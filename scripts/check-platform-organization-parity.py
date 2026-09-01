#!/usr/bin/env python3
from pathlib import Path
import sys

ROOT = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()

CHECKS = {
    ROOT / 'apps/backend/cmd/server/organizations_v100.go': [
        'requirePlatformAdmin).Put("/{organizationID}"',
        'organizations.Update(',
        'audit.ActionOrganizationUpdated',
        'store.ErrOrganizationSlugTaken',
    ],
    ROOT / 'apps/backend/internal/store/organizations.go': [
        'func (s *OrganizationStore) Update(',
        'UPDATE organizations',
        'updated_at=now()',
        'ErrOrganizationSlugTaken',
    ],
    ROOT / 'apps/backend/internal/audit/audit.go': [
        'ActionOrganizationUpdated',
        'organization.updated',
    ],
    ROOT / 'apps/frontend/src/lib/api.ts': [
        'export async function updateOrganization',
        '/api/v1/organizations/${encodeURIComponent(organizationId)}',
    ],
    ROOT / 'apps/frontend/src/components/PlatformOrganizationsPanel.jsx': [
        'updateOrganization(',
        "t('platform.edit')",
        "t('platform.status')",
        'organizationById',
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
    print('platform organization UI parity contract: FAIL')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)

print('platform organization UI parity contract: OK')
