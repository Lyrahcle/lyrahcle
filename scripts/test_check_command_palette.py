import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / 'scripts/check-command-palette.py'


class CommandPaletteContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        result = subprocess.run(['python3', str(CHECK), str(ROOT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr + result.stdout)

    def test_missing_direct_satellite_navigation_fails(self):
        with tempfile.TemporaryDirectory() as temp:
            target = Path(temp)
            for relative in [
                'apps/frontend/src/components/CommandPalette.jsx',
                'apps/frontend/src/components/Header.jsx',
                'apps/frontend/src/components/Dashboard.jsx',
                'apps/frontend/src/lib/translations.js',
            ]:
                source = ROOT / relative
                destination = target / relative
                destination.parent.mkdir(parents=True, exist_ok=True)
                text = source.read_text()
                if relative.endswith('Dashboard.jsx'):
                    text = text.replace("result?.type === 'satellites'", "result?.type === 'removed'")
                destination.write_text(text)
            result = subprocess.run(['python3', str(CHECK), str(target)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('open satellite profile directly', result.stdout)


if __name__ == '__main__':
    unittest.main()
