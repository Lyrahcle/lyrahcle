import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name('check-code-cleanliness.py')


class CodeCleanlinessContractTests(unittest.TestCase):
    def run_check(self, root):
        return subprocess.run(
            [sys.executable, str(SCRIPT), str(root)],
            capture_output=True,
            text=True,
            check=False,
        )

    def test_generated_terraform_modules_are_ignored(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            module = root / 'deploy' / 'terraform' / 'aws' / '.terraform' / 'modules' / 'vpc'
            module.mkdir(parents=True)
            (module / 'main.tf').write_text('# downloaded module comment\nresource "null_resource" "x" {}\n')
            result = self.run_check(root)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
            self.assertIn('code cleanliness contract: OK', result.stdout)

    def test_owned_source_comment_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            source = root / 'apps' / 'frontend' / 'src'
            source.mkdir(parents=True)
            (source / 'example.ts').write_text('// ordinary source comment\nexport const value = 1\n')
            result = self.run_check(root)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('apps/frontend/src/example.ts:1', result.stdout)


if __name__ == '__main__':
    unittest.main()
