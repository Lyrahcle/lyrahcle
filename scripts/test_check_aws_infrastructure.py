import subprocess
import tempfile
import unittest
from pathlib import Path
import shutil

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / "scripts" / "check-aws-infrastructure.py"

class AwsInfrastructureContractTest(unittest.TestCase):
    def test_current_tree_passes(self):
        proc = subprocess.run(["python3", str(CHECK), str(ROOT)], text=True, capture_output=True)
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertIn("contract: OK", proc.stdout)

    def test_world_open_tfvars_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            dst = Path(tmp) / "repo"
            shutil.copytree(ROOT / "deploy", dst / "deploy")
            (dst / "scripts").mkdir(parents=True)
            shutil.copy2(CHECK, dst / "scripts" / CHECK.name)
            tfvars = dst / "deploy" / "terraform" / "aws" / "terraform.tfvars.example"
            tfvars.write_text(tfvars.read_text().replace('203.0.113.10/32', '0.0.0.0/0'))
            proc = subprocess.run(["python3", str(dst / "scripts" / CHECK.name), str(dst)], text=True, capture_output=True)
            self.assertNotEqual(proc.returncode, 0)

if __name__ == "__main__":
    unittest.main()
