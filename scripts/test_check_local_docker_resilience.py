import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path


class LocalDockerResilienceContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        root = Path(__file__).resolve().parents[1]
        result = subprocess.run(
            ['python3', str(root / 'scripts/check-local-docker-resilience.py'), str(root)],
            capture_output=True,
            text=True,
        )
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn('local docker resilience contract: OK', result.stdout)

    def test_remote_dockerfile_frontend_is_rejected(self):
        root = Path(__file__).resolve().parents[1]
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp)
            for rel in ['docker-compose.yml', 'deploy/docker/Dockerfile.backend', 'deploy/docker/Dockerfile.frontend', 'scripts/docker-local-up.sh']:
                target = clone / rel
                target.parent.mkdir(parents=True, exist_ok=True)
                shutil.copy2(root / rel, target)
            dockerfile = clone / 'deploy/docker/Dockerfile.backend'
            dockerfile.write_text('# syntax=docker/dockerfile:1\n' + dockerfile.read_text())
            result = subprocess.run(
                ['python3', str(root / 'scripts/check-local-docker-resilience.py'), str(clone)],
                capture_output=True,
                text=True,
            )
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('remote dockerfile frontend resolution', result.stdout)


if __name__ == '__main__':
    unittest.main()
