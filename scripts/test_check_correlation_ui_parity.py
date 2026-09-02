import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / 'scripts' / 'check-correlation-ui-parity.py'

class CorrelationUIParityContractTest(unittest.TestCase):
    def test_repository_passes_contract(self):
        result = subprocess.run(['python3', str(CHECK), str(ROOT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('correlation UI parity contract: OK', result.stdout)

    def test_missing_incident_integration_fails(self):
        with tempfile.TemporaryDirectory() as temp:
            root = Path(temp)
            for rel in [
                'apps/backend/cmd/server/cross_domain_v94.go',
                'apps/frontend/src/components/CorrelationEvidencePanel.jsx',
                'apps/frontend/src/components/IncidentsPanel.tsx',
                'apps/frontend/src/lib/translations.js',
                'apps/frontend/src/components/CorrelationEvidencePanel.test.jsx',
            ]:
                target = root / rel
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_text((ROOT / rel).read_text())
            p = root / 'apps/frontend/src/components/IncidentsPanel.tsx'
            p.write_text(p.read_text().replace('<CorrelationEvidencePanel evidence={detail.evidence || []} />', ''))
            result = subprocess.run(['python3', str(CHECK), str(root)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('does not render correlation evidence', result.stderr)

if __name__ == '__main__':
    unittest.main()
