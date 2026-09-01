import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / "scripts" / "check-aws-production-readiness.py"


class AwsProductionReadinessContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        result = subprocess.run(["python3", str(CHECK), str(ROOT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("contract: OK", result.stdout)

    def test_mutable_ecr_regression_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            dst = Path(tmp) / "repo"
            shutil.copytree(ROOT / "deploy", dst / "deploy")
            shutil.copytree(ROOT / "docs", dst / "docs")
            (dst / "scripts").mkdir(parents=True)
            for name in ["aws-build-push-images.sh", "aws-render-release-values.py", "aws-production-readiness.sh", CHECK.name]:
                shutil.copy2(ROOT / "scripts" / name, dst / "scripts" / name)
            ecr = dst / "deploy" / "terraform" / "aws" / "ecr.tf"
            ecr.write_text(ecr.read_text().replace('image_tag_mutability = "IMMUTABLE"', 'image_tag_mutability = "MUTABLE"'))
            result = subprocess.run(["python3", str(dst / "scripts" / CHECK.name), str(dst)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)

    def test_invalid_hcl_sentinels_are_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            dst = Path(tmp) / "repo"
            shutil.copytree(ROOT / "deploy", dst / "deploy")
            shutil.copytree(ROOT / "docs", dst / "docs")
            (dst / "scripts").mkdir(parents=True)
            for name in ["aws-build-push-images.sh", "aws-render-release-values.py", "aws-production-readiness.sh", CHECK.name]:
                shutil.copy2(ROOT / "scripts" / name, dst / "scripts" / name)
            guardrails = dst / "deploy" / "terraform" / "aws" / "guardrails.tf"
            guardrails.write_text(guardrails.read_text().replace("    }\n  }\n}", "    }  }\n}", 1))
            result = subprocess.run(["python3", str(dst / "scripts" / CHECK.name), str(dst)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)

        with tempfile.TemporaryDirectory() as tmp:
            dst = Path(tmp) / "repo"
            shutil.copytree(ROOT / "deploy", dst / "deploy")
            shutil.copytree(ROOT / "docs", dst / "docs")
            (dst / "scripts").mkdir(parents=True)
            for name in ["aws-build-push-images.sh", "aws-render-release-values.py", "aws-production-readiness.sh", CHECK.name]:
                shutil.copy2(ROOT / "scripts" / name, dst / "scripts" / name)
            variables = dst / "deploy" / "terraform" / "aws" / "variables.tf"
            variables.write_text(variables.read_text().replace("[.]", "\\.", 1))
            result = subprocess.run(["python3", str(dst / "scripts" / CHECK.name), str(dst)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)


if __name__ == "__main__":
    unittest.main()
