import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


class SatelliteSecurityProfileContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        root = Path(__file__).resolve().parents[1]
        result = subprocess.run(
            ['python3', str(root / 'scripts/check-satellite-security-profile.py'), str(root)],
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('satellite security profile contract: OK', result.stdout)

    def test_satellite_scope_removal_is_rejected(self):
        root = Path(__file__).resolve().parents[1]
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp)
            files = [
                'apps/frontend/src/components/SatelliteSecurityProfile.jsx',
                'apps/frontend/src/components/SatellitesPanel.jsx',
                'apps/frontend/src/components/Dashboard.tsx',
                'apps/frontend/src/lib/translations.js',
            ]
            for rel in files:
                target = clone / rel
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(root / rel, target)
            profile = clone / 'apps/frontend/src/components/SatelliteSecurityProfile.jsx'
            profile.write_text(profile.read_text().replace('item.satellite_id === satellite.id', 'true', 1))
            result = subprocess.run(
                ['python3', str(root / 'scripts/check-satellite-security-profile.py'), str(clone)],
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('scope both incident and RF evidence', result.stdout)


if __name__ == '__main__':
    unittest.main()
