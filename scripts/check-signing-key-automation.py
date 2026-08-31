#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
checks = {
    'apps/backend/internal/keyprovider/rotation.go': [
        'type RotationPolicy struct', 'AutomaticPrepare', 'IntervalSeconds', 'StagedAlertAfterSeconds',
        'GraceAlertBeforeSeconds', 'type RotationHealth struct', 'AssessRotation', 'rotationBase := active.CreatedAt',
        'active.ActivatedAt',
    ],
    'apps/backend/internal/keyprovider/provider.go': [
        'ActivatedAt', 'RotationPolicy(context.Context)', 'UpdateRotationPolicy(context.Context, RotationPolicy)',
        'RotationPolicy RotationPolicy',
    ],
    'apps/backend/cmd/server/signing_key_automation_v137.go': [
        'runSigningKeyRotationCheck', 'policy.AutomaticPrepare', 'health.RotationDue', 'health.StagedPending',
        'lifecycle.PrepareRotation', 'ActionSigningKeyAutoPrepared', 'RecordSigningKeyLifecycle',
    ],
    'apps/backend/cmd/server/signing_keys_v136.go': [
        'r.Put("/policy"', 'ActionSigningKeyPolicyUpdated', 'RotationPolicy', 'AssessRotation',
    ],
    'apps/backend/internal/metrics/metrics.go': [
        'lyrahcle_signing_keys', 'lyrahcle_signing_key_rotation_due', 'lyrahcle_signing_key_staged_pending',
        'lyrahcle_signing_key_warnings', 'lyrahcle_signing_key_automation_total', 'RecordSigningKeyLifecycle',
    ],
    'apps/frontend/src/components/SigningKeysPanel.jsx': [
        'updateAgentConfigSigningRotationPolicy', "t('signingKeys.rotationPolicy')", 'automatic_prepare',
        'state?.health?.warnings', "t('signingKeys.automationSafety')",
    ],
    'deploy/prometheus/alerts.yml': [
        'LyrahcleSigningKeyNoActiveKey', 'LyrahcleSigningKeyRotationDue', 'LyrahcleSigningKeyStagedPending',
        'LyrahcleSigningKeyGraceExpiring', 'LyrahcleSigningKeyAutomationFailure',
    ],
}
errors = []
for rel, needles in checks.items():
    path = root / rel
    if not path.is_file():
        errors.append(f'missing {rel}')
        continue
    text = path.read_text(encoding='utf-8')
    for needle in needles:
        if needle not in text:
            errors.append(f'{rel}: missing contract token {needle!r}')

automation = root / 'apps/backend/cmd/server/signing_key_automation_v137.go'
if automation.is_file():
    text = automation.read_text(encoding='utf-8')
    if 'lifecycle.Activate(' in text or '.Activate(ctx' in text:
        errors.append('scheduled signing-key automation must never activate a staged key automatically')

if errors:
    print('signing-key automation contract: FAILED')
    for error in errors:
        print(f' - {error}')
    raise SystemExit(1)
print('signing-key automation contract: OK')
