import subprocess
import tempfile
import unittest
from pathlib import Path


class BackendReliabilityContractTests(unittest.TestCase):
    def setUp(self):
        self.root = Path(__file__).resolve().parents[1]
        self.check = self.root / 'scripts' / 'check-backend-reliability.py'

    def run_check(self, root):
        return subprocess.run(['python3', str(self.check), str(root)], capture_output=True, text=True)

    def copy_minimal_tree(self, target):
        for rel in [
            'apps/backend/internal/store/assets.go',
            'apps/backend/internal/redisclient/redisclient.go',
            'apps/backend/internal/redisclient/redisclient_test.go',
            'apps/backend/internal/db/db.go',
        ]:
            source = self.root / rel
            destination = target / rel
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_text(source.read_text())

    def test_real_repository_passes(self):
        result = self.run_check(self.root)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_lost_error_chain_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            target = Path(tmp)
            self.copy_minimal_tree(target)
            path = target / 'apps/backend/internal/store/assets.go'
            text = path.read_text().replace('fmt.Errorf("store: list assets: %w", err)', 'fmt.Errorf("store: list assets: %v", err)', 1)
            if text == path.read_text():
                text += '\nfunc lostError(err error) error { return fmt.Errorf("lost: %v", err) }\n'
            path.write_text(text)
            result = self.run_check(target)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('error argument formatted without %w', result.stdout)

    def test_store_background_context_fails(self):
        with tempfile.TemporaryDirectory() as tmp:
            target = Path(tmp)
            self.copy_minimal_tree(target)
            path = target / 'apps/backend/internal/store/assets.go'
            path.write_text(path.read_text() + '\nfunc backgroundContextRegression() { _ = context.Background() }\n')
            result = self.run_check(target)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('store code must not replace caller context', result.stdout)


if __name__ == '__main__':
    unittest.main()
