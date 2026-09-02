import importlib.util
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("collect-dr-evidence.py")
spec = importlib.util.spec_from_file_location("collect_dr_evidence", MODULE_PATH)
module = importlib.util.module_from_spec(spec)
assert spec.loader
spec.loader.exec_module(module)


class CollectDREvidenceTests(unittest.TestCase):
    def kind(self):
        return {
            "status": "PASS",
            "environment": "local-kind",
            "generated_at": "2026-09-02T00:00:00+00:00",
            "results": [
                {"name": "backend pod recovery", "actual_seconds": 12.4, "target_seconds": 60, "status": "PASS"},
                {"name": "PostgreSQL service recovery", "actual_seconds": 18.1, "target_seconds": 90, "status": "PASS"},
            ],
        }

    def restore(self):
        return {
            "status": "PASS",
            "generated_at": "2026-09-02T00:01:00+00:00",
            "backup_seconds": 1.2,
            "restore_seconds": 2.5,
            "tables_verified": 42,
            "migrations_verified": 42,
        }

    def test_builds_pass_evidence_and_preserves_scope(self):
        evidence = module.build_evidence(self.kind(), self.restore())
        self.assertEqual(evidence["status"], "PASS")
        self.assertEqual(evidence["rpo"]["validated_scope"], "backup boundary")
        self.assertEqual(evidence["restore_integrity"]["tables_verified"], 42)
        self.assertIn("not an AWS production DR certification", evidence["certification_scope"])

    def test_kind_measurement_over_target_fails_closed(self):
        kind = self.kind()
        kind["results"][0]["actual_seconds"] = 61
        with self.assertRaisesRegex(ValueError, "target exceeded"):
            module.build_evidence(kind, self.restore())

    def test_non_pass_restore_fails_closed(self):
        restore = self.restore()
        restore["status"] = "FAIL"
        with self.assertRaisesRegex(ValueError, "status must be PASS"):
            module.build_evidence(self.kind(), restore)

    def test_empty_kind_results_are_rejected(self):
        kind = self.kind()
        kind["results"] = []
        with self.assertRaisesRegex(ValueError, "non-empty results"):
            module.build_evidence(kind, self.restore())


if __name__ == "__main__":
    unittest.main()
