import importlib.util
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("check-go-toolchain.py")
spec = importlib.util.spec_from_file_location("check_go_toolchain", SCRIPT)
module = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(module)


class GoToolchainContractTests(unittest.TestCase):
    def make_repo(self, *, module_toolchain="1.26.7", docker="1.26.7", workflow="1.26.7"):
        temp = tempfile.TemporaryDirectory()
        root = Path(temp.name)
        for rel in module.MODULES:
            p = root / rel
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(f"module example.test/x\n\ngo 1.26.0\n\ntoolchain go{module_toolchain}\n")
        for rel in module.DOCKERFILES:
            p = root / rel
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(f"FROM golang:{docker}-alpine AS build\n")
        for rel in module.WORKFLOWS:
            p = root / rel
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(f"steps:\n  - uses: actions/setup-go@v7\n    with:\n      go-version: '{workflow}'\n")
        return temp, root

    def test_accepts_consistent_pin(self):
        temp, root = self.make_repo()
        self.addCleanup(temp.cleanup)
        self.assertEqual(module.check(root), [])

    def test_rejects_old_module_toolchain(self):
        temp, root = self.make_repo(module_toolchain="1.25.0")
        self.addCleanup(temp.cleanup)
        self.assertTrue(any("toolchain" in err for err in module.check(root)))

    def test_rejects_floating_docker_builder(self):
        temp, root = self.make_repo(docker="1")
        self.addCleanup(temp.cleanup)
        self.assertTrue(any("Dockerfile" in err for err in module.check(root)))

    def test_rejects_floating_ci_version(self):
        temp, root = self.make_repo(workflow="stable")
        self.addCleanup(temp.cleanup)
        self.assertTrue(any("setup-go" in err or "floating" in err for err in module.check(root)))


if __name__ == "__main__":
    unittest.main()
