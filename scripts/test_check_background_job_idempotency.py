import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / "scripts/check-background-job-idempotency.py"


class BackgroundJobIdempotencyContractTest(unittest.TestCase):
    def test_current_tree_passes(self):
        result = subprocess.run(["python3", str(CHECK), str(ROOT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_destructive_pop_is_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            clone = Path(tmp) / "repo"
            subprocess.run(["cp", "-a", str(ROOT), str(clone)], check=True)
            worker = clone / "apps/backend/internal/alerts/worker.go"
            worker.write_text(worker.read_text() + '\n// regression: w.redis.BLPop(ctx, time.Second, QueueKey)\n')
            result = subprocess.run(["python3", str(CHECK), str(clone)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn("destructive blocking list pop", result.stdout + result.stderr)


if __name__ == "__main__":
    unittest.main()
