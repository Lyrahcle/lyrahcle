import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / "scripts" / "check-aws-load-balancer-controller.py"


class AwsLoadBalancerControllerContractTests(unittest.TestCase):
    def test_real_repository_passes(self):
        result = subprocess.run(["python3", str(CHECK), str(ROOT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertIn("contract: OK", result.stdout)

    def test_node_role_permission_regression_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            dst = Path(tmp) / "repo"
            shutil.copytree(ROOT / "deploy", dst / "deploy")
            (dst / "scripts").mkdir(parents=True)
            shutil.copy2(CHECK, dst / "scripts" / CHECK.name)
            target = dst / "deploy" / "terraform" / "aws" / "aws-load-balancer-controller.tf"
            target.write_text(target.read_text() + '\nresource "aws_iam_role_policy_attachment" "bad" {\n  role = module.eks.eks_managed_node_groups["primary"].iam_role_name\n  policy_arn = aws_iam_policy.aws_load_balancer_controller[0].arn\n}\n')
            result = subprocess.run(["python3", str(dst / "scripts" / CHECK.name), str(dst)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)


if __name__ == "__main__":
    unittest.main()
