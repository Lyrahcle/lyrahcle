import tempfile
import unittest
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / 'scripts' / 'check-csrf-cookie-auth.py'


class CSRFCookieAuthContractTests(unittest.TestCase):
    def test_current_tree_satisfies_contract(self):
        result = subprocess.run([sys.executable, str(CHECK), str(ROOT)], capture_output=True, text=True)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_checker_rejects_unallowlisted_cookie_setter(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            p = fake / 'apps/backend/cmd/server/session_cookie.go'
            p.parent.mkdir(parents=True, exist_ok=True)
            p.write_text(
                'package main\n\n'
                'func setSessionCookie(w http.ResponseWriter) {\n'
                '\thttp.SetCookie(w, &http.Cookie{Name: "session", Value: "x"})\n'
                '}\n'
            )
            result = subprocess.run([sys.executable, str(CHECK), str(fake)], capture_output=True, text=True)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('session_cookie.go', result.stdout)


if __name__ == '__main__':
    unittest.main()
