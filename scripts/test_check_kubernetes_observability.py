import importlib.util
import shutil
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("check-kubernetes-observability.py")
spec = importlib.util.spec_from_file_location("check_kubernetes_observability", SCRIPT)
module = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(module)


class KubernetesObservabilityContractTests(unittest.TestCase):
    def copy_repo(self):
        source = Path(__file__).resolve().parents[1]
        temp = tempfile.TemporaryDirectory()
        root = Path(temp.name) / "repo"
        for relative in ("deploy/helm", "deploy/kind", "deploy/k8s/local"):
            src = source / relative
            shutil.copytree(src, root / relative)
        (root / "scripts").mkdir(parents=True)
        for name in ("k8s-local-up.sh", "k8s-local-down.sh", "k8s-smoke-test.sh"):
            shutil.copy2(source / "scripts" / name, root / "scripts" / name)
        return temp, root

    def test_current_repository_passes(self):
        root = Path(__file__).resolve().parents[1]
        self.assertEqual(module.check(root), [])

    def test_rejects_missing_service_monitor(self):
        temp, root = self.copy_repo()
        self.addCleanup(temp.cleanup)
        (root / "deploy/helm/lyrahcle/templates/servicemonitor.yaml").unlink()
        self.assertTrue(any("service_monitor" in error for error in module.check(root)))

    def test_rejects_single_replica_staging(self):
        temp, root = self.copy_repo()
        self.addCleanup(temp.cleanup)
        path = root / "deploy/helm/lyrahcle/values-staging.yaml"
        path.write_text(path.read_text().replace("replicaCount: 2", "replicaCount: 1"))
        self.assertTrue(any("multi-replica staging" in error for error in module.check(root)))

    def test_rejects_missing_tls_guard(self):
        temp, root = self.copy_repo()
        self.addCleanup(temp.cleanup)
        path = root / "deploy/helm/lyrahcle/templates/validate.yaml"
        path.write_text(path.read_text().replace("backend TLS ServiceMonitor requires a CA secret", "removed guard"))
        self.assertTrue(any("fail-closed ServiceMonitor TLS" in error for error in module.check(root)))


if __name__ == "__main__":
    unittest.main()
