import subprocess
import tempfile
import unittest
from pathlib import Path


class RedisTLSAuthContractTests(unittest.TestCase):
    def setUp(self):
        self.root = Path(__file__).resolve().parents[1]
        self.check = self.root / 'scripts' / 'check-redis-tls-auth.py'

    def test_real_repository_passes(self):
        result = subprocess.run(['python3', str(self.check), str(self.root)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_plaintext_terraform_regression_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            target = Path(tmp)
            for source in [
                'apps/backend/internal/redisclient/redisclient.go',
                'apps/backend/internal/config/config.go',
                'apps/backend/cmd/server/main.go',
                'apps/cve-sync/internal/redisclient/redisclient.go',
                'apps/cve-sync/internal/config/config.go',
                'docker-compose.yml',
                'deploy/helm/lyrahcle/values.yaml',
                'deploy/helm/lyrahcle/templates/backend-deployment.yaml',
                'deploy/helm/lyrahcle/templates/cve-sync-deployment.yaml',
                'deploy/terraform/aws/redis.tf',
                'deploy/terraform/aws/variables.tf',
                'deploy/terraform/aws/outputs.tf',
                'docs/configuration.md',
            ]:
                src = self.root / source
                dst = target / source
                dst.parent.mkdir(parents=True, exist_ok=True)
                dst.write_text(src.read_text())
            redis_tf = target / 'deploy/terraform/aws/redis.tf'
            redis_tf.write_text(redis_tf.read_text().replace('transit_encryption_mode    = "required"', 'transit_encryption_mode    = "preferred"'))
            result = subprocess.run(['python3', str(self.check), str(target)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('transit_encryption_mode', result.stdout)


if __name__ == '__main__':
    unittest.main()
