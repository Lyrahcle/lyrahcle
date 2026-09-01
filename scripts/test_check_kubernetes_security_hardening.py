import importlib.util
import shutil
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name('check-kubernetes-security-hardening.py')
spec = importlib.util.spec_from_file_location('check_kubernetes_security_hardening', SCRIPT)
module = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(module)


class KubernetesSecurityHardeningContractTests(unittest.TestCase):
    def copy_repo(self):
        source = Path(__file__).resolve().parents[1]
        temp = tempfile.TemporaryDirectory()
        root = Path(temp.name) / 'repo'
        for rel in ('deploy/helm/lyrahcle', 'deploy/k8s/security'):
            shutil.copytree(source / rel, root / rel)
        (root / 'scripts').mkdir(parents=True, exist_ok=True)
        for name in ('k8s-local-up.sh', 'k8s-enforce-restricted.sh'):
            shutil.copy2(source / 'scripts' / name, root / 'scripts' / name)
        return temp, root

    def test_current_repository_passes(self):
        root = Path(__file__).resolve().parents[1]
        self.assertEqual(module.check(root), [])

    def test_rejects_missing_init_capability_drop(self):
        temp, root = self.copy_repo()
        self.addCleanup(temp.cleanup)
        path = root / 'deploy/helm/lyrahcle/templates/backend-deployment.yaml'
        content = path.read_text()
        path.write_text(content.replace('drop: ["ALL"]', 'drop: ["NET_RAW"]', 1))
        errors = module.check(root)
        self.assertTrue(any('init and app capability drop' in item for item in errors), errors)

    def test_rejects_disabled_production_network_policy(self):
        temp, root = self.copy_repo()
        self.addCleanup(temp.cleanup)
        path = root / 'deploy/helm/lyrahcle/values-production.yaml'
        path.write_text(path.read_text().replace('networkPolicy:\n  enabled: true', 'networkPolicy:\n  enabled: false', 1))
        errors = module.check(root)
        self.assertTrue(any('production network policy enforcement' in item for item in errors), errors)


if __name__ == '__main__':
    unittest.main()
