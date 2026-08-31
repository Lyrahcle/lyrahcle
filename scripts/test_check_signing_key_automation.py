import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECKER = ROOT / 'scripts' / 'check-signing-key-automation.py'


class SigningKeyAutomationContractTest(unittest.TestCase):
    def test_repository_satisfies_contract(self):
        result = subprocess.run(['python3', str(CHECKER), str(ROOT)], text=True, capture_output=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_checker_rejects_missing_automation(self):
        with tempfile.TemporaryDirectory() as tmp:
            result = subprocess.run(['python3', str(CHECKER), tmp], text=True, capture_output=True)
            self.assertNotEqual(result.returncode, 0)


if __name__ == '__main__':
    unittest.main()
