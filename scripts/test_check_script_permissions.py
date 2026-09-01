import os
import subprocess
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name('check-script-permissions.py')
REQUIRED = [
    'test-local-v1.sh', 'smoke-test.sh', 'generate-sbom.sh', 'backup-db.sh',
    'restore-db.sh', 'verify-backup.sh', 'load-test.sh', 'generate-agent-ca.sh',
    'generate-agent-config-key.sh', 'generate-dev-certs.sh',
    'aws-drift-check.sh',
]

class ScriptPermissionContractTests(unittest.TestCase):
    def make_repo(self):
        td = tempfile.TemporaryDirectory()
        root = Path(td.name)
        scripts = root / 'scripts'
        scripts.mkdir()
        for name in REQUIRED:
            p = scripts / name
            p.write_text('#!/usr/bin/env bash\n')
            p.chmod(0o755)
        return td, root

    def test_accepts_executable_ci_scripts(self):
        td, root = self.make_repo()
        self.addCleanup(td.cleanup)
        result = subprocess.run(['python3', str(SCRIPT), str(root)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_rejects_non_executable_script(self):
        td, root = self.make_repo()
        self.addCleanup(td.cleanup)
        (root / 'scripts' / 'smoke-test.sh').chmod(0o644)
        result = subprocess.run(['python3', str(SCRIPT), str(root)], capture_output=True, text=True)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('smoke-test.sh: not executable', result.stderr)

if __name__ == '__main__':
    unittest.main()
