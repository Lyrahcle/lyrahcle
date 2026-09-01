#!/usr/bin/env python3
from __future__ import annotations

import gzip
import importlib.util
import sys
import tempfile
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("check-frontend-bundle.py")
SPEC = importlib.util.spec_from_file_location("check_frontend_bundle_v2", MODULE_PATH)
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC and SPEC.loader
sys.modules[SPEC.name] = MODULE
SPEC.loader.exec_module(MODULE)


def asset(name: str, raw_kib: int, gzip_kib: int = 100):
    return MODULE.AssetSize(
        Path(name), raw_bytes=raw_kib * 1024, gzip_bytes=gzip_kib * 1024
    )


class FrontendBundleBudgetV2Tests(unittest.TestCase):
    def evaluate(self, sizes):
        return MODULE.evaluate(
            sizes,
            max_js_kib=500,
            max_gzip_kib=180,
            warn_js_kib=450,
            min_js_chunks=5,
        )

    def test_defaults_are_strict_code_splitting_budgets(self):
        self.assertEqual(MODULE.DEFAULT_MAX_JS_KIB, 500)
        self.assertEqual(MODULE.DEFAULT_MAX_GZIP_KIB, 180)
        self.assertEqual(MODULE.DEFAULT_MIN_JS_CHUNKS, 5)

    def test_accepts_split_bundle_below_limits(self):
        errors, warnings = self.evaluate(
            [asset(f"chunk-{i}.js", 100 + i) for i in range(5)]
        )
        self.assertEqual(errors, [])
        self.assertEqual(warnings, [])

    def test_rejects_chunk_over_500_kib(self):
        sizes = [asset("large.js", 501)] + [
            asset(f"chunk-{i}.js", 100) for i in range(4)
        ]
        errors, _ = self.evaluate(sizes)
        self.assertTrue(any("raw JS budget 500 KiB" in e for e in errors))

    def test_rejects_gzip_regression(self):
        sizes = [asset("large.js", 400, 181)] + [
            asset(f"chunk-{i}.js", 100) for i in range(4)
        ]
        errors, _ = self.evaluate(sizes)
        self.assertTrue(any("gzip budget 180 KiB" in e for e in errors))

    def test_rejects_collapsed_code_splitting(self):
        errors, _ = self.evaluate([asset("index.js", 400)])
        self.assertTrue(any("expected at least 5" in e for e in errors))

    def test_warns_near_limit_without_failing(self):
        sizes = [asset("near.js", 475)] + [
            asset(f"chunk-{i}.js", 100) for i in range(4)
        ]
        errors, warnings = self.evaluate(sizes)
        self.assertEqual(errors, [])
        self.assertEqual(len(warnings), 1)

    def test_measure_is_deterministic(self):
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "a.js"
            payload = b"const a = 1;\n" * 100
            path.write_bytes(payload)
            measured = MODULE.measure(path)
            self.assertEqual(measured.raw_bytes, len(payload))
            self.assertEqual(
                measured.gzip_bytes, len(gzip.compress(payload, mtime=0))
            )

    def test_missing_build_skips_by_default(self):
        with tempfile.TemporaryDirectory() as tmp:
            self.assertEqual(MODULE.main([tmp]), 0)

    def test_missing_build_fails_when_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            self.assertEqual(MODULE.main(["--require-build", tmp]), 2)


if __name__ == "__main__":
    unittest.main()
