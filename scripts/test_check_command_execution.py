#!/usr/bin/env python3
import importlib.util
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("check-command-execution.py")
spec = importlib.util.spec_from_file_location("command_contract", SCRIPT)
module = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(module)


class CommandExecutionContractTests(unittest.TestCase):
    def make_repo(self, relative: str, content: str) -> Path:
        tmp = tempfile.TemporaryDirectory()
        self.addCleanup(tmp.cleanup)
        root = Path(tmp.name)
        target = root / relative
        target.parent.mkdir(parents=True, exist_ok=True)
        target.write_text(content, encoding="utf-8")
        return root

    def test_allows_normal_application_code(self):
        root = self.make_repo("apps/backend/main.go", 'package main\nimport "fmt"\nfunc main(){ fmt.Println("ok") }\n')
        self.assertEqual([], module.scan(root))

    def test_rejects_go_os_exec(self):
        root = self.make_repo(
            "apps/backend/main.go",
            'package main\nimport "os/exec"\nfunc main(){ exec.Command("sh", "-c", "echo x").Run() }\n',
        )
        findings = module.scan(root)
        self.assertTrue(any("Go os/exec import" in item for item in findings))
        self.assertTrue(any("Go process execution" in item for item in findings))

    def test_rejects_node_child_process(self):
        root = self.make_repo(
            "apps/frontend/tool.js",
            'const { spawn } = require("node:child_process");\nspawn("tool", ["arg"]);\n',
        )
        findings = module.scan(root)
        self.assertTrue(any("Node child_process import" in item for item in findings))
        self.assertTrue(any("Node process execution" in item for item in findings))

    def test_does_not_scan_operational_scripts(self):
        root = self.make_repo("scripts/ops.sh", '#!/bin/sh\nsh -c "$SOMETHING"\n')
        self.assertEqual([], module.scan(root))

    def test_skips_node_modules(self):
        root = self.make_repo("apps/frontend/node_modules/pkg/index.js", 'require("child_process").exec("x")\n')
        self.assertEqual([], module.scan(root))


if __name__ == "__main__":
    unittest.main()
