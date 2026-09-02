import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


class PerformanceObservabilityCheckerTest(unittest.TestCase):
    def test_repository_contract_passes(self):
        root = Path(__file__).resolve().parents[1]
        result = subprocess.run(
            [sys.executable, str(root / 'scripts/check-performance-observability.py'), str(root)],
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('performance observability contract: OK', result.stdout)


if __name__ == '__main__':
    unittest.main()
