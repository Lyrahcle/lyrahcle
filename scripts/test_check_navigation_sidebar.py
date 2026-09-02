import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


class NavigationSidebarContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        root = Path(__file__).resolve().parents[1]
        result = subprocess.run(
            ['python3', str(root / 'scripts/check-navigation-sidebar.py'), str(root)],
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn('navigation sidebar contract: OK', result.stdout)

    def test_permission_group_removal_is_rejected(self):
        root = Path(__file__).resolve().parents[1]
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp)
            files = [
                'apps/frontend/src/components/Dashboard.tsx',
                'apps/frontend/src/components/Header.jsx',
                'apps/frontend/src/components/NavigationMenu.jsx',
                'apps/frontend/src/components/dashboard/navigation.ts',
                'apps/frontend/src/lib/translations.js',
                'apps/frontend/src/lib/routeTabs.ts',
            ]
            for rel in files:
                target = clone / rel
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(root / rel, target)
            navigation_model = clone / 'apps/frontend/src/components/dashboard/navigation.ts'
            navigation_model.write_text(navigation_model.read_text().replace('access.administration ?', 'true ?'))
            result = subprocess.run(
                ['python3', str(root / 'scripts/check-navigation-sidebar.py'), str(clone)],
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('preserve permission-aware destinations', result.stdout)


if __name__ == '__main__':
    unittest.main()
