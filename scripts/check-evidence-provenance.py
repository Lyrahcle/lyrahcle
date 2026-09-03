#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
checks = {
    'apps/backend/migrations/0043_evidence_provenance.sql': ['applicability_status', 'match_method', 'confidence', 'context_only'],
    'apps/backend/internal/alerts/worker.go': ['CreateWithAssessment', '"thematic_asset_kind"', '"unknown"', '20, true'],
    'apps/backend/internal/store/asset_alerts.go': ['json:"applicability_status"', 'json:"context_only"'],
    'apps/backend/cmd/server/mission_runtime.go': ['PositionEvidenceKind', '"inferred"', '"celestrak_or_operator"'],
    'apps/frontend/src/components/EvidenceBadge.tsx': ["'external'", "'inferred'", "'context'"],
    'apps/frontend/src/components/AssetAlertsPanel.jsx': ['contextWarningTitle', 'applicability_status', 'context_only'],
    'apps/frontend/src/lib/risk.ts': ['item.context_only === false', "'probable_exposure', 'confirmed'"],
    'docs/evidence-provenance.md': ['Context only', 'applicability_status=unknown'],
}

errors = []
for relative, tokens in checks.items():
    path = root / relative
    if not path.exists():
        errors.append(f'missing {relative}')
        continue
    content = path.read_text(encoding='utf-8')
    for token in tokens:
        if token not in content:
            errors.append(f'{relative} missing {token!r}')

if errors:
    print('evidence provenance contract: FAIL')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)
print('evidence provenance contract: OK')

