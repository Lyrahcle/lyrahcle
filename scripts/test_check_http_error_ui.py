import subprocess
import tempfile
import unittest
from pathlib import Path


class HttpErrorUiContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        root = Path(__file__).resolve().parents[1]
        result = subprocess.run(['python3', str(root / 'scripts/check-http-error-ui.py'), str(root)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('HTTP error UI contract: OK', result.stdout)

    def test_raw_response_error_is_rejected(self):
        root = Path(__file__).resolve().parents[1]
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp)
            for rel in [
                'apps/frontend/src/lib/api.ts', 'apps/frontend/src/App.tsx',
                'apps/frontend/src/components/Dashboard.tsx', 'apps/frontend/src/components/ApiStatusBanner.jsx',
                'apps/frontend/src/hooks/useDashboardApiIssue.ts',
                'apps/frontend/src/components/Login.jsx', 'apps/frontend/src/lib/translations.js',
                'scripts/test-local-v1.sh',
            ]:
                target = clone / rel
                target.parent.mkdir(parents=True, exist_ok=True)
                target.write_text((root / rel).read_text())
            api = clone / 'apps/frontend/src/lib/api.ts'
            api.write_text(api.read_text() + "\nif (!res.ok) throw new Error('legacy')\n")
            result = subprocess.run(['python3', str(root / 'scripts/check-http-error-ui.py'), str(clone)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('discard HTTP metadata', result.stderr)


if __name__ == '__main__':
    unittest.main()
