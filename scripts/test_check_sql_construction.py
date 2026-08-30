import pathlib
import subprocess
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).with_name("check-sql-construction.py")

class SQLConstructionContractTests(unittest.TestCase):
    def run_scan(self, files):
        with tempfile.TemporaryDirectory() as td:
            root = pathlib.Path(td)
            store = root / "apps/backend/internal/store"
            store.mkdir(parents=True)
            for name, content in files.items():
                (store / name).write_text(content)
            return subprocess.run(["python3", str(SCRIPT), str(root)], capture_output=True, text=True)

    def test_static_parameterized_sql_passes(self):
        r = self.run_scan({"assets.go": 'package store\nfunc f(){ q := `SELECT * FROM assets WHERE id=$1`; _ = q }'})
        self.assertEqual(r.returncode, 0, r.stderr)

    def test_unreviewed_sprintf_fails(self):
        r = self.run_scan({"assets.go": 'package store\nimport "fmt"\nfunc f(s string){ _ = fmt.Sprintf("SELECT * FROM x ORDER BY %s", s) }'})
        self.assertNotEqual(r.returncode, 0)

if __name__ == '__main__':
    unittest.main()
