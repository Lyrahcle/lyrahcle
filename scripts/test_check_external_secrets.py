import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / "scripts" / "check-external-secrets.py"


class ExternalSecretsContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        result = subprocess.run(["python3", str(CHECK), str(ROOT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("contract: OK", result.stdout)

    def test_secret_write_permission_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            dst = Path(tmp) / "repo"
            shutil.copytree(ROOT / "deploy", dst / "deploy")
            shutil.copytree(ROOT / "scripts", dst / "scripts")
            shutil.copytree(ROOT / "docs" / "screenshots", dst / "docs" / "screenshots")
            shutil.copy2(ROOT / "README.md", dst / "README.md")
            target = dst / "deploy" / "terraform" / "aws" / "external-secrets.tf"
            target.write_text(target.read_text() + '\nresource "terraform_data" "bad" { input = "secretsmanager:PutSecretValue" }\n')
            result = subprocess.run(["python3", str(dst / "scripts" / CHECK.name), str(dst)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)


if __name__ == "__main__":
    unittest.main()
