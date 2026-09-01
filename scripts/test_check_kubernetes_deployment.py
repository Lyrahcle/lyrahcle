import importlib.util
import shutil
import tempfile
import unittest
from pathlib import Path

SCRIPT = Path(__file__).with_name("check-kubernetes-deployment.py")
spec = importlib.util.spec_from_file_location("check_kubernetes_deployment", SCRIPT)
module = importlib.util.module_from_spec(spec)
assert spec.loader is not None
spec.loader.exec_module(module)


class KubernetesDeploymentContractTests(unittest.TestCase):
    def copy_repo(self):
        source = Path(__file__).resolve().parents[1]
        temp = tempfile.TemporaryDirectory()
        root = Path(temp.name) / "repo"
        shutil.copytree(source / "deploy/helm", root / "deploy/helm")
        docker_src = source / "deploy/docker/Dockerfile.frontend"
        docker_dst = root / "deploy/docker/Dockerfile.frontend"
        docker_dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(docker_src, docker_dst)
        db_src = source / "apps/backend/internal/db/db.go"
        db_dst = root / "apps/backend/internal/db/db.go"
        db_dst.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(db_src, db_dst)
        return temp, root

    def test_current_repository_passes(self):
        root = Path(__file__).resolve().parents[1]
        self.assertEqual(module.check(root), [])

    def test_rejects_missing_advisory_lock(self):
        temp, root = self.copy_repo()
        self.addCleanup(temp.cleanup)
        path = root / "apps/backend/internal/db/db.go"
        path.write_text(path.read_text().replace("pg_advisory_lock", "pg_removed_lock"))
        self.assertTrue(any("serialized database migrations" in err for err in module.check(root)))

    def test_rejects_non_writable_nginx_runtime(self):
        temp, root = self.copy_repo()
        self.addCleanup(temp.cleanup)
        dockerfile = root / "deploy/docker/Dockerfile.frontend"
        dockerfile.write_text(dockerfile.read_text().replace(" /run /usr/share/nginx/html", " /var/run /usr/share/nginx/html"))
        deployment = root / "deploy/helm/lyrahcle/templates/frontend-deployment.yaml"
        deployment.write_text(deployment.read_text().replace("mountPath: /run", "mountPath: /var/run"))
        errors = module.check(root)
        self.assertTrue(any("nginx runtime" in err for err in errors), errors)

    def test_rejects_mutable_production_tag(self):
        temp, root = self.copy_repo()
        self.addCleanup(temp.cleanup)
        path = root / "deploy/helm/lyrahcle/values-production.yaml"
        path.write_text(path.read_text().replace("tag: 14.6.8", "tag: latest", 1))
        self.assertTrue(any("mutable production image tag" in err for err in module.check(root)))


if __name__ == "__main__":
    unittest.main()
