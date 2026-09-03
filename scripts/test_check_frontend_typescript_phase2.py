import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / 'scripts/check-frontend-typescript-phase2.py'

class FrontendTypeScriptPhase2ContractTest(unittest.TestCase):
    def run_check(self, root: Path) -> subprocess.CompletedProcess[str]:
        return subprocess.run(['python3', str(CHECK), str(root)], capture_output=True, text=True)

    def test_repository_passes(self):
        result = self.run_check(ROOT)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_legacy_dashboard_path_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp) / 'repo'
            shutil.copytree(ROOT, clone, ignore=shutil.ignore_patterns('.git', 'node_modules', '.terraform', '__pycache__'))
            dashboard = clone / 'apps/frontend/src/components/Dashboard.tsx'
            dashboard.rename(clone / 'apps/frontend/src/components/Dashboard.jsx')
            result = self.run_check(clone)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('legacy frontend path still present', result.stdout)

    def test_hash_routing_in_app_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp) / 'repo'
            shutil.copytree(ROOT, clone, ignore=shutil.ignore_patterns('.git', 'node_modules', '.terraform', '__pycache__'))
            app = clone / 'apps/frontend/src/App.tsx'
            app.write_text(app.read_text() + '\nconst routeRegression = subscribeTabChanges\n')
            result = self.run_check(clone)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('still owns hash-routing mechanics', result.stdout)

    def test_untyped_js_boundary_regression_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp) / 'repo'
            shutil.copytree(ROOT, clone, ignore=shutil.ignore_patterns('.git', 'node_modules', '.terraform', '__pycache__'))
            dashboard = clone / 'apps/frontend/src/components/Dashboard.tsx'
            source = dashboard.read_text().replace("lazyComponent<any>(() => import('./SatellitesPanel.jsx'))", "lazyComponent(() => import('./SatellitesPanel.jsx'))")
            dashboard.write_text(source)
            result = self.run_check(clone)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("lazyComponent<any>(() => import('./SatellitesPanel.jsx'))", result.stdout)

    def test_async_request_array_regression_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp) / 'repo'
            shutil.copytree(ROOT, clone, ignore=shutil.ignore_patterns('.git', 'node_modules', '.terraform', '__pycache__'))
            dashboard_query = clone / 'apps/frontend/src/components/dashboard/dashboardQuery.ts'
            dashboard_query.write_text(
                dashboard_query.read_text().replace(
                    'const tasks: Promise<void>[] = []',
                    'const tasks = []',
                )
            )
            result = self.run_check(clone)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('const tasks: Promise<void>[] = []', result.stdout)

if __name__ == '__main__':
    unittest.main()
