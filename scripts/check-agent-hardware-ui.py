#!/usr/bin/env python3
from pathlib import Path
import sys

from check_source_utils import read_source_tree

root = Path(sys.argv[1] if len(sys.argv) > 1 else '.')
checks = {
    'migration': ('apps/backend/migrations/0038_agent_hardware_capabilities.sql', ['device_capabilities', 'agent_identities', 'agent_keys']),
    'store': ('apps/backend/internal/store/agent_keys.go', ['AgentDeviceCapabilities', 'compatibility_reason', 'finalizeAgentDeviceInfo']),
    'tenant-list': ('apps/backend/internal/store/agent_identity.go', ['ListByOrganization', 'ReportDeviceInOrganization', 'device_capabilities', 'organization_id = $2']),
    'agent-report': ('apps/sdr-agent/internal/uplink/uplink.go', ['frequency_min_hz', 'sample_rates_hz', 'gain_elements', 'sample_drops']),
    'ui': ('apps/frontend/src/components/AgentHardwarePanel.jsx', ['hardwareSummary', 'compatibility_reason', 'sample_rates_hz', 'hardwareValidationNote']),
    'ui-wiring': ('apps/frontend/src/components/ConnectivityPanel.jsx', ['AgentHardwarePanel', 'hardwareInventory']),
    'openapi': ('apps/backend/cmd/server/openapi.yaml', ['AgentDeviceCapabilities:', 'compatibility_reason:', 'sample_rates_hz:']),
}
errors = []
for label, (path, needles) in checks.items():
    p = root / path
    if not p.exists():
        errors.append(f'{label}: missing {path}')
        continue
    text = p.read_text(errors='replace')
    for needle in needles:
        if needle not in text:
            errors.append(f'{label}: {path} missing {needle!r}')


server_dir = root / 'apps/backend/cmd/server'
server_text = read_source_tree(server_dir)
for needle in ['Post("/agents/me/device"', 'validateAgentDeviceReport', 'ActionAgentHardwareReported', 'logAuditInOrganization']:
    if needle not in server_text:
        errors.append(f'machine-route: backend server sources missing {needle!r}')

translations = (root / 'apps/frontend/src/lib/translations.js').read_text(errors='replace')
if translations.count('hardwareInventory:') < 4 or translations.count('hardwareValidationNote:') < 4:
    errors.append('translations: hardware inventory strings must exist in EN/ES/FR/DE')

if errors:
    print('\n'.join(errors), file=sys.stderr)
    raise SystemExit(1)
print('agent hardware/UI parity contract: OK')
