import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / 'scripts/check-frontend-ux-foundation.py'

class FrontendUXFoundationContractTest(unittest.TestCase):
    def run_check(self, root: Path) -> subprocess.CompletedProcess[str]:
        return subprocess.run(['python3', str(CHECK), str(root)], capture_output=True, text=True)

    def clone(self, destination: Path) -> Path:
        clone = destination / 'repo'
        shutil.copytree(ROOT, clone, ignore=shutil.ignore_patterns('.git', 'node_modules', '.terraform', '__pycache__'))
        return clone

    def test_repository_passes(self):
        result = self.run_check(ROOT)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_persistent_critical_toasts_are_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            clone = self.clone(Path(tmp))
            path = clone / 'apps/frontend/src/components/ToastProvider.tsx'
            path.write_text(path.read_text().replace("tone === 'error' || tone === 'warning' ? 0 : 6000", '6000'))
            result = self.run_check(clone)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('ToastProvider.tsx missing', result.stdout)

    def test_skip_link_target_is_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            clone = self.clone(Path(tmp))
            path = clone / 'apps/frontend/src/components/Dashboard.tsx'
            path.write_text(path.read_text().replace('<main id="main-content"', '<main id="workspace"'))
            result = self.run_check(clone)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('Dashboard.tsx missing', result.stdout)

if __name__ == '__main__':
    unittest.main()
