import pathlib
import subprocess
import tempfile
import unittest

SCRIPT = pathlib.Path(__file__).with_name("check-frontend-xss.py")

class FrontendXSSContractTests(unittest.TestCase):
    def run_scan(self, files):
        with tempfile.TemporaryDirectory() as td:
            root = pathlib.Path(td)
            base = root / "apps/frontend/src"
            base.mkdir(parents=True)
            for rel, content in files.items():
                path = base / rel
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_text(content)
            return subprocess.run(["python3", str(SCRIPT), str(root)], capture_output=True, text=True)

    def test_safe_blank_link_passes(self):
        r = self.run_scan({"A.jsx": '<a href="https://example.com" target="_blank" rel="noopener noreferrer">x</a>'})
        self.assertEqual(r.returncode, 0, r.stderr)

    def test_blank_link_without_rel_fails(self):
        r = self.run_scan({"A.jsx": '<a href="https://example.com" target="_blank">x</a>'})
        self.assertNotEqual(r.returncode, 0)

    def test_duplicate_rel_attribute_fails(self):
        # Regression test: a duplicate `rel` prop is valid JS (the object
        # literal's last key wins), so only the last value is ever applied
        # at runtime -- this previously let a `noopener`-bearing but
        # shadowed first `rel` mask a real missing-noopener bug.
        r = self.run_scan({
            "A.jsx": '<a href="https://example.com" target="_blank" rel="noopener noreferrer" rel="noreferrer">x</a>'
        })
        self.assertNotEqual(r.returncode, 0)
        self.assertIn('rel attributes', r.stderr)

    def test_unreviewed_raw_html_fails(self):
        r = self.run_scan({"A.jsx": '<div dangerouslySetInnerHTML={{ __html: html }} />'})
        self.assertNotEqual(r.returncode, 0)

    def test_inner_html_assignment_fails(self):
        r = self.run_scan({"A.js": 'node.innerHTML = input'})
        self.assertNotEqual(r.returncode, 0)

if __name__ == '__main__':
    unittest.main()
