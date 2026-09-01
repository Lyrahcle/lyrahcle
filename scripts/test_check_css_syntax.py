import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / 'scripts' / 'check-css-syntax.py'


class CSSSyntaxContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        result = subprocess.run(['python3', str(CHECK), str(ROOT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_unexpected_closing_brace_fails(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            css = root / 'apps' / 'frontend' / 'src' / 'index.css'
            css.parent.mkdir(parents=True)
            css.write_text('body { color: red; }\n}\n')
            result = subprocess.run(['python3', str(CHECK), str(root)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('unexpected closing brace', result.stdout)


if __name__ == '__main__':
    unittest.main()
