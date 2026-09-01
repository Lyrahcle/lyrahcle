import subprocess
import tempfile
import unittest
from pathlib import Path


class MissionSecurityDashboardContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        root = Path(__file__).resolve().parents[1]
        result = subprocess.run(
            ['python3', str(root / 'scripts/check-mission-security-dashboard.py'), str(root)],
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('mission security dashboard contract: OK', result.stdout)

    def test_permission_guard_removal_is_rejected(self):
        root = Path(__file__).resolve().parents[1]
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp)
            files = [
                'apps/frontend/src/components/Dashboard.jsx',
                'apps/frontend/src/components/MissionSecurityOperationsDashboard.jsx',
                'apps/frontend/src/components/MissionSecurityOperationsDashboard.test.jsx',
                'apps/frontend/src/lib/translations.js',
            ]
            for rel in files:
                target = clone / rel
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_text((root / rel).read_text())
            dashboard = clone / 'apps/frontend/src/components/Dashboard.jsx'
            dashboard.write_text(dashboard.read_text().replace('if (rfRead) requests.push(safeFetch(fetchRfAnomalies, setRfAnomalies))', 'requests.push(safeFetch(fetchRfAnomalies, setRfAnomalies))'))
            result = subprocess.run(
                ['python3', str(root / 'scripts/check-mission-security-dashboard.py'), str(clone)],
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('permission-aware integration marker', result.stderr)


if __name__ == '__main__':
    unittest.main()
