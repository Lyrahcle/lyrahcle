import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / "scripts/check-kubernetes-resilience.py"


class KubernetesResilienceContractTests(unittest.TestCase):
    def run_check(self, root):
        return subprocess.run(["python3", str(CHECK), str(root)], capture_output=True, text=True)

    def test_repository_passes(self):
        result = self.run_check(ROOT)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_missing_fault_cleanup_is_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            target = Path(temp) / "repo"
            shutil.copytree(ROOT, target, ignore=shutil.ignore_patterns(".git", "node_modules", ".terraform", "__pycache__", "dist"))
            path = target / "scripts/k8s-resilience-test.sh"
            path.write_text(path.read_text().replace("trap restore_faults EXIT", "trap true EXIT", 1))
            result = self.run_check(target)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("trap restore_faults EXIT", result.stdout)

    def test_weak_production_spread_is_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            target = Path(temp) / "repo"
            shutil.copytree(ROOT, target, ignore=shutil.ignore_patterns(".git", "node_modules", ".terraform", "__pycache__", "dist"))
            path = target / "deploy/helm/lyrahcle/values-production.yaml"
            path.write_text(path.read_text().replace("zoneWhenUnsatisfiable: DoNotSchedule", "zoneWhenUnsatisfiable: ScheduleAnyway"))
            result = self.run_check(target)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("production topology spread", result.stdout)


    def test_weak_aws_spread_is_rejected(self):
        with tempfile.TemporaryDirectory() as temp:
            target = Path(temp) / "repo"
            shutil.copytree(ROOT, target, ignore=shutil.ignore_patterns(".git", "node_modules", ".terraform", "__pycache__", "dist"))
            path = target / "deploy/helm/lyrahcle/values-aws.example.yaml"
            path.write_text(path.read_text().replace("zoneWhenUnsatisfiable: DoNotSchedule", "zoneWhenUnsatisfiable: ScheduleAnyway", 1))
            result = self.run_check(target)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("AWS production topology spread", result.stdout)


    def test_fault_injection_must_break_existing_connections(self):
        with tempfile.TemporaryDirectory() as temp:
            target = Path(temp) / "repo"
            shutil.copytree(ROOT, target, ignore=shutil.ignore_patterns(".git", "node_modules", ".terraform", "__pycache__", "dist"))
            path = target / "scripts/k8s-resilience-test.sh"
            path.write_text(path.read_text().replace("redis-cli CLIENT KILL TYPE normal", "redis-cli PING", 1))
            result = self.run_check(target)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("redis-cli CLIENT KILL TYPE normal", result.stdout)

    def test_postgres_fault_must_terminate_existing_sessions(self):
        with tempfile.TemporaryDirectory() as temp:
            target = Path(temp) / "repo"
            shutil.copytree(ROOT, target, ignore=shutil.ignore_patterns(".git", "node_modules", ".terraform", "__pycache__", "dist"))
            path = target / "scripts/k8s-resilience-test.sh"
            path.write_text(path.read_text().replace("pg_terminate_backend(pid)", "pg_backend_pid()", 1))
            result = self.run_check(target)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("pg_terminate_backend(pid)", result.stdout)

    def test_restore_drill_must_use_test_database(self):
        with tempfile.TemporaryDirectory() as temp:
            target = Path(temp) / "repo"
            shutil.copytree(ROOT, target, ignore=shutil.ignore_patterns(".git", "node_modules", ".terraform", "__pycache__", "dist"))
            path = target / "scripts/dr-local-restore-test.sh"
            path.write_text(path.read_text().replace('SOURCE_DATABASE="lyrahcle_test"', 'SOURCE_DATABASE="lyrahcle"', 1))
            result = self.run_check(target)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('SOURCE_DATABASE="lyrahcle_test"', result.stdout)


if __name__ == "__main__":
    unittest.main()
