import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

class TypeScriptMigrationContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        result = subprocess.run(['python3', str(ROOT / 'scripts/check-typescript-migration.py'), str(ROOT)], cwd=ROOT, capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

if __name__ == '__main__':
    unittest.main()
