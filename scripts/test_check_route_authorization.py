import tempfile
import unittest
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / 'scripts' / 'check-route-authorization.py'


class RouteAuthorizationContractTests(unittest.TestCase):
    def test_current_tree_satisfies_contract(self):
        result = subprocess.run([sys.executable, str(CHECK), str(ROOT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_checker_rejects_bare_unprotected_route(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            p = fake / 'apps/backend/cmd/server/main.go'
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(
                'package main\n\n'
                'func registerRoutes(r chi.Router) {\n'
                '\tr.Route("/api/v1", func(r chi.Router) {\n'
                '\t\tr.Post("/admin/delete-everything", func(w http.ResponseWriter, r *http.Request) {\n'
                '\t\t\tdoSomethingDangerous()\n'
                '\t\t})\n'
                '\t})\n'
                '}\n'
            )
            result = subprocess.run([sys.executable, str(CHECK), str(fake)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('admin/delete-everything', result.stdout)

    def test_checker_accepts_route_gated_by_enclosing_use(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            p = fake / 'apps/backend/cmd/server/main.go'
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(
                'package main\n\n'
                'func registerRoutes(r chi.Router) {\n'
                '\tr.Route("/api/v1", func(r chi.Router) {\n'
                '\t\tr.Use(auth.RequireRoleWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, auth.RoleViewer, isTokenRevoked))\n'
                '\t\tr.Post("/admin/delete-everything", func(w http.ResponseWriter, r *http.Request) {\n'
                '\t\t\tdoSomethingDangerous()\n'
                '\t\t})\n'
                '\t})\n'
                '}\n'
            )
            result = subprocess.run([sys.executable, str(CHECK), str(fake)], capture_output=True, text=True)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_checker_does_not_leak_sibling_route_guard(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            p = fake / 'apps/backend/cmd/server/main.go'
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(
                'package main\n\n'
                'func registerRoutes(r chi.Router) {\n'
                '\tr.Route("/api/v1", func(r chi.Router) {\n'
                '\t\tr.With(auth.RequirePermissionWithFallback(cfg.JWTSecret, cfg.JWTPreviousSecret, rbac, "assets.read", isTokenRevoked)).Get("/assets", handleAssets)\n'
                '\t\tr.Post("/admin/delete-everything", func(w http.ResponseWriter, r *http.Request) {\n'
                '\t\t\tdoSomethingDangerous()\n'
                '\t\t})\n'
                '\t})\n'
                '}\n'
            )
            result = subprocess.run([sys.executable, str(CHECK), str(fake)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('admin/delete-everything', result.stdout)


if __name__ == '__main__':
    unittest.main()
