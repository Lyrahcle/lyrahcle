#!/usr/bin/env python3
from __future__ import annotations

import contextlib
import importlib.util
import io
import tempfile
import unittest
from pathlib import Path

MODULE_PATH = Path(__file__).with_name("check-frontend-test-warnings.py")
spec = importlib.util.spec_from_file_location("warning_checker", MODULE_PATH)
mod = importlib.util.module_from_spec(spec)
assert spec and spec.loader
spec.loader.exec_module(mod)


class WarningCheckerTests(unittest.TestCase):
    def test_clean_output_passes(self):
        self.assertEqual(mod.check("Test Files 1 passed\nTests 4 passed\n"), [])

    def test_act_warning_fails(self):
        out = "Warning: An update to CopyButton inside a test was not wrapped in act(...)."
        self.assertIn("react act warning", mod.check(out))

    def test_expected_error_boundary_fixture_is_not_misclassified(self):
        self.assertEqual(mod.check("Error: boom\nErrorBoundary caught expected fixture"), [])

    def test_directory_input_is_skipped_for_contract_runner(self):
        with tempfile.TemporaryDirectory() as td:
            text, reason = mod.read_input(td)
        self.assertIsNone(text)
        self.assertIn("directory", reason)

    def test_main_directory_mode_returns_successful_skip(self):
        with tempfile.TemporaryDirectory() as td:
            stdout = io.StringIO()
            with contextlib.redirect_stdout(stdout):
                rc = mod.main([td])
        self.assertEqual(rc, 0)
        self.assertIn("frontend test warnings: SKIP:", stdout.getvalue())

    def test_file_mode_remains_strict(self):
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "vitest.log"
            path.write_text("Warning: update was not wrapped in act(...).\n", encoding="utf-8")
            stderr = io.StringIO()
            with contextlib.redirect_stderr(stderr):
                rc = mod.main([str(path)])
        self.assertEqual(rc, 1)
        self.assertIn("react act warning", stderr.getvalue())

    def test_missing_file_is_an_input_error(self):
        stderr = io.StringIO()
        with contextlib.redirect_stderr(stderr):
            rc = mod.main(["/definitely/not/present/vitest.log"])
        self.assertEqual(rc, 2)
        self.assertIn("cannot read input", stderr.getvalue())


if __name__ == "__main__":
    unittest.main()
