#!/usr/bin/env python3
from pathlib import Path
import sys

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.').resolve()
errors = []

def text(path):
    p = root / path
    if not p.exists():
        errors.append(f'missing required file: {path}')
        return ''
    return p.read_text(encoding='utf-8')

backend = text('apps/backend/cmd/server/cross_domain_v94.go')
component = text('apps/frontend/src/components/CorrelationEvidencePanel.jsx')
incidents = text('apps/frontend/src/components/IncidentsPanel.jsx')
translations = text('apps/frontend/src/lib/translations.js')
component_test = text('apps/frontend/src/components/CorrelationEvidencePanel.test.jsx')

for token in (
    '"confidence":            result.Confidence',
    '"candidate_severity":    result.CandidateSeverity',
    '"corroborating_domains": result.CorroboratingDomains',
    '"context_only_domains":  result.ContextOnlyDomains',
    '"reasons":               result.Reasons',
    '"severity_applied":      false',
    '"policy":                "explainable_candidate_only"',
):
    if token not in backend:
        errors.append(f'backend correlation evidence contract missing {token}')

for token in (
    'cross-domain-v9.4', 'candidate_severity', 'corroborating_domains',
    'context_only_domains', 'severity_applied', 'orbital-operations-v9.5',
    'unacknowledged-7d', 'Recent agent audit activity',
):
    if token not in component:
        errors.append(f'correlation UI does not surface {token}')

if "import CorrelationEvidencePanel from './CorrelationEvidencePanel.jsx'" not in incidents:
    errors.append('IncidentsPanel does not import CorrelationEvidencePanel')
if '<CorrelationEvidencePanel evidence={detail.evidence || []} />' not in incidents:
    errors.append('IncidentsPanel does not render correlation evidence from incident detail')
if 'correlationEvidenceTranslations' not in translations:
    errors.append('correlation UI translations are missing')
if 'not applied to severity' not in component_test.lower():
    errors.append('frontend test does not assert candidate severity remains unapplied')

if errors:
    for error in errors:
        print(f'ERROR: {error}', file=sys.stderr)
    raise SystemExit(1)
print('correlation UI parity contract: OK')
