import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


class EvidenceProvenanceContractTests(unittest.TestCase):
    def run_check(self, target: Path):
        root = Path(__file__).resolve().parents[1]
        return subprocess.run(
            ['python3', str(root / 'scripts/check-evidence-provenance.py'), str(target)],
            capture_output=True,
            text=True,
        )

    def test_repository_passes(self):
        root = Path(__file__).resolve().parents[1]
        result = self.run_check(root)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_context_only_risk_guard_is_required(self):
        root = Path(__file__).resolve().parents[1]
        with tempfile.TemporaryDirectory() as temp:
            clone = Path(temp)
            for relative in [
                'apps/backend/migrations/0043_evidence_provenance.sql',
                'apps/backend/internal/alerts/worker.go',
                'apps/backend/internal/store/asset_alerts.go',
                'apps/backend/cmd/server/mission_runtime.go',
                'apps/frontend/src/components/EvidenceBadge.tsx',
                'apps/frontend/src/components/AssetAlertsPanel.jsx',
                'apps/frontend/src/lib/risk.ts',
                'docs/evidence-provenance.md',
            ]:
                destination = clone / relative
                destination.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(root / relative, destination)
            risk = clone / 'apps/frontend/src/lib/risk.ts'
            risk.write_text(risk.read_text().replace('item.context_only === false', 'true'), encoding='utf-8')
            result = self.run_check(clone)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('risk.ts', result.stdout)


if __name__ == '__main__':
    unittest.main()

