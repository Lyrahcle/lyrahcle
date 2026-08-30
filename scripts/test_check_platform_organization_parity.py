import subprocess
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name('check-platform-organization-parity.py')


class PlatformOrganizationParityContractTest(unittest.TestCase):
    def test_current_repository_passes(self):
        root = Path(__file__).resolve().parents[1]
        result = subprocess.run(['python3', str(SCRIPT), str(root)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_missing_surface_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            result = subprocess.run(['python3', str(SCRIPT), tmp], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('missing file', result.stdout)


if __name__ == '__main__':
    unittest.main()
