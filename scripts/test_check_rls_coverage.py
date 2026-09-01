import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / 'scripts' / 'check-rls-coverage.py'

# The check's filename uses a hyphen, which is not an importable module
# identifier; exec its source into a namespace to reuse its exception set
# instead of duplicating that list in this test file.
_ns: dict = {}
exec(compile((ROOT / 'scripts' / 'check-rls-coverage.py').read_text(), 'check-rls-coverage.py', 'exec'), _ns)
APP_LEVEL_TENANT_EXCEPTIONS = _ns['APP_LEVEL_TENANT_EXCEPTIONS']


def run(target):
    return subprocess.run([sys.executable, str(CHECK), str(target)], capture_output=True, text=True)


def baseline_migration_sql() -> str:
    """CREATE TABLE statements for every documented app-level exception, so
    fixture repos don't spuriously trip the 'stale exception' guard just
    because they only model the table under test."""
    lines = []
    for table in APP_LEVEL_TENANT_EXCEPTIONS:
        lines.append(f"CREATE TABLE {table} (\n    id UUID PRIMARY KEY,\n    organization_id UUID NOT NULL\n);\n")
    return "\n".join(lines)


class RlsCoverageContractTests(unittest.TestCase):
    def test_current_tree_satisfies_contract(self):
        result = run(ROOT)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_missing_migrations_directory_fails(self):
        with tempfile.TemporaryDirectory() as td:
            result = run(Path(td))
            self.assertNotEqual(result.returncode, 0)

    def test_new_tenant_table_without_rls_or_exception_fails(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            migrations = fake / 'apps' / 'backend' / 'migrations'
            migrations.mkdir(parents=True)
            (migrations / '0000_baseline.sql').write_text(baseline_migration_sql())
            (migrations / '0001_new_table.sql').write_text(
                "CREATE TABLE widgets (\n"
                "    id UUID PRIMARY KEY,\n"
                "    organization_id UUID NOT NULL\n"
                ");\n"
            )
            result = run(fake)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('widgets', result.stdout)

    def test_literal_enable_rls_satisfies_contract(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            migrations = fake / 'apps' / 'backend' / 'migrations'
            migrations.mkdir(parents=True)
            (migrations / '0000_baseline.sql').write_text(baseline_migration_sql())
            (migrations / '0001_new_table.sql').write_text(
                "CREATE TABLE widgets (\n"
                "    id UUID PRIMARY KEY,\n"
                "    organization_id UUID NOT NULL\n"
                ");\n"
                "ALTER TABLE widgets ENABLE ROW LEVEL SECURITY;\n"
            )
            result = run(fake)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_dynamic_array_enable_rls_satisfies_contract(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            migrations = fake / 'apps' / 'backend' / 'migrations'
            migrations.mkdir(parents=True)
            (migrations / '0000_baseline.sql').write_text(baseline_migration_sql())
            (migrations / '0001_new_table.sql').write_text(
                "CREATE TABLE widgets (\n"
                "    id UUID PRIMARY KEY,\n"
                "    organization_id UUID NOT NULL\n"
                ");\n"
                "DO $$\n"
                "DECLARE target_table text;\n"
                "BEGIN\n"
                "    FOREACH target_table IN ARRAY ARRAY['widgets']\n"
                "    LOOP\n"
                "        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', target_table);\n"
                "    END LOOP;\n"
                "END $$;\n"
            )
            result = run(fake)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_stale_exception_for_now_covered_table_fails(self):
        # A table hardcoded into APP_LEVEL_TENANT_EXCEPTIONS (e.g. "users")
        # that gains real RLS coverage should force the exception to be
        # removed, not silently accepted twice.
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            migrations = fake / 'apps' / 'backend' / 'migrations'
            migrations.mkdir(parents=True)
            other_exceptions = {t: r for t, r in APP_LEVEL_TENANT_EXCEPTIONS.items() if t != 'users'}
            baseline = "\n".join(
                f"CREATE TABLE {t} (\n    id UUID PRIMARY KEY,\n    organization_id UUID NOT NULL\n);\n"
                for t in other_exceptions
            )
            (migrations / '0000_baseline.sql').write_text(baseline)
            (migrations / '0001_users.sql').write_text(
                "CREATE TABLE users (\n"
                "    id UUID PRIMARY KEY,\n"
                "    organization_id UUID NOT NULL\n"
                ");\n"
                "ALTER TABLE users ENABLE ROW LEVEL SECURITY;\n"
            )
            result = run(fake)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('stale', result.stdout.lower())


if __name__ == '__main__':
    unittest.main()
