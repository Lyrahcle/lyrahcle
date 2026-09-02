import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECKER = ROOT / 'scripts/check-correlation-graph.py'
PATHS = [
    'apps/frontend/src/components/CorrelationGraph.tsx',
    'apps/frontend/src/lib/correlationGraph.ts',
    'apps/frontend/src/components/Dashboard.tsx',
    'apps/frontend/src/components/dashboard/navigation.ts',
    'apps/frontend/src/lib/routeTabs.ts',
    'apps/frontend/src/lib/translations.js',
    'apps/frontend/src/index.css',
    'apps/frontend/package.json',
    'apps/frontend/tailwind.config.js',
    'docs/correlation-constellation.md',
]

class CorrelationGraphContractTest(unittest.TestCase):
    def run_checker(self, root: Path):
        return subprocess.run(['python3', str(CHECKER), str(root)], capture_output=True, text=True)

    def fixture(self, target: Path):
        for relative in PATHS:
            source = ROOT / relative
            destination = target / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(source, destination)

    def test_repository_passes(self):
        result = self.run_checker(ROOT)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_shortest_path_is_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            model = root / 'apps/frontend/src/lib/correlationGraph.ts'
            model.write_text(model.read_text().replace('shortestCorrelationPath', 'removedCorrelationPath'))
            result = self.run_checker(root)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('explainable shortest path', result.stdout)

    def test_external_graph_engines_are_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            package = root / 'apps/frontend/package.json'
            package.write_text(package.read_text().replace('"react":', '"cytoscape": "1.0.0",\n    "react":'))
            result = self.run_checker(root)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('no graph runtime dependency', result.stdout)


    def test_tailwind_typescript_scan_is_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            tailwind = root / 'apps/frontend/tailwind.config.js'
            tailwind.write_text(tailwind.read_text().replace('{js,jsx,ts,tsx}', '{js,jsx}'))
            result = self.run_checker(root)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('Tailwind scans TSX graph classes', result.stdout)

    def test_3d_projection_is_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            component = root / 'apps/frontend/src/components/CorrelationGraph.tsx'
            component.write_text(component.read_text().replace('projectOrbitRing3D', 'removedOrbitProjection'))
            result = self.run_checker(root)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('minimal 3D projection', result.stdout)

if __name__ == '__main__':
    unittest.main()
