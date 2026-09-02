import subprocess
import tempfile
import unittest
from pathlib import Path


class BackendBootstrapArchitectureContractTests(unittest.TestCase):
    def setUp(self):
        self.root = Path(__file__).resolve().parents[1]
        self.check = self.root / 'scripts' / 'check-backend-bootstrap-architecture.py'

    def test_real_repository_passes(self):
        result = subprocess.run(['python3', str(self.check), str(self.root)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_monolithic_main_regression_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            target = Path(tmp)
            for rel in [
                'apps/backend/cmd/server/main.go',
                'apps/backend/cmd/server/agent_config_v91.go',
                'apps/backend/cmd/server/agent_identity_v90.go',
                'apps/backend/cmd/server/auth_v84.go',
                'apps/backend/cmd/server/http_helpers.go',
                'apps/backend/cmd/server/mission_runtime.go',
                'apps/backend/cmd/server/validation_audit.go',
                'apps/backend/cmd/server/startup.go',
                'apps/backend/internal/bootstrap/stores.go',
                'apps/backend/internal/bootstrap/security.go',
                'apps/backend/internal/bootstrap/ratelimit.go',
                'apps/backend/internal/bootstrap/services.go',
                'apps/backend/internal/bootstrap/http.go',
                'apps/backend/internal/bootstrap/bootstrap_test.go',
            ]:
                src = self.root / rel
                dst = target / rel
                dst.parent.mkdir(parents=True, exist_ok=True)
                dst.write_text(src.read_text())
            main = target / 'apps/backend/cmd/server/main.go'
            main.write_text(main.read_text() + '\n' + 'x\n' * 100)
            result = subprocess.run(['python3', str(self.check), str(target)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('main.go remains too large', result.stdout)


if __name__ == '__main__':
    unittest.main()
