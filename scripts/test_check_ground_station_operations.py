import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


class GroundStationOperationsContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        root = Path(__file__).resolve().parents[1]
        result = subprocess.run(
            ['python3', str(root / 'scripts/check-ground-station-operations.py'), str(root)],
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('ground station operations contract: OK', result.stdout)

    def test_station_scope_removal_is_rejected(self):
        root = Path(__file__).resolve().parents[1]
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp)
            files = [
                'apps/frontend/src/components/Dashboard.jsx',
                'apps/frontend/src/components/GroundStationOperations.jsx',
                'apps/frontend/src/components/WorldMap.jsx',
                'apps/frontend/src/lib/risk.ts',
                'apps/frontend/src/lib/groundStations.ts',
                'apps/frontend/src/lib/routeTabs.ts',
                'apps/frontend/src/lib/translations.js',
            ]
            for rel in files:
                target = clone / rel
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(root / rel, target)
            component = clone / 'apps/frontend/src/components/GroundStationOperations.jsx'
            component.write_text(component.read_text().replace('item.agent_id === agent.agent_id', 'true'))
            result = subprocess.run(
                ['python3', str(root / 'scripts/check-ground-station-operations.py'), str(clone)],
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('scope evidence to station agent', result.stdout)


if __name__ == '__main__':
    unittest.main()
