import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECKER = ROOT / 'scripts/check-correlation-graph.py'
PATHS = [
    'apps/frontend/src/components/CorrelationGraph.tsx',
    'apps/frontend/src/components/correlation/OrbitalScene.tsx',
    'apps/frontend/src/components/correlation/NodeMesh.tsx',
    'apps/frontend/src/components/correlation/EvidenceLinks.tsx',
    'apps/frontend/src/components/correlation/RoutePath.tsx',
    'apps/frontend/src/components/correlation/SmartLabels.tsx',
    'apps/frontend/src/components/correlation/ClusterMarkers.tsx',
    'apps/frontend/src/components/correlation/FocusRig.tsx',
    'apps/frontend/src/components/correlation/ContextPanel.tsx',
    'apps/frontend/src/components/correlation/FilterPopover.tsx',
    'apps/frontend/src/components/correlation/NodeSearch.tsx',
    'apps/frontend/src/components/correlation/NodeIndex.tsx',
    'apps/frontend/src/components/correlation/capabilities.ts',
    'apps/frontend/src/components/correlation/sceneTheme.ts',
    'apps/frontend/src/lib/correlationGraph.ts',
    'apps/frontend/src/lib/correlationLayout3d.ts',
    'apps/frontend/src/lib/correlationLayout3d.test.ts',
    'apps/frontend/src/components/Dashboard.tsx',
    'apps/frontend/src/components/dashboard/navigation.ts',
    'apps/frontend/src/lib/routeTabs.ts',
    'apps/frontend/src/lib/translations.js',
    'apps/frontend/src/index.css',
    'apps/frontend/package.json',
    'apps/frontend/vite.config.js',
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

    def mutate(self, root: Path, relative: str, old: str, new: str):
        path = root / relative
        text = path.read_text()
        self.assertIn(old, text, f'fixture for {relative} no longer contains {old!r}')
        path.write_text(text.replace(old, new))

    def assert_fails_with(self, root: Path, check_name: str):
        result = self.run_checker(root)
        self.assertNotEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertIn(check_name, result.stdout)

    def test_repository_passes(self):
        result = self.run_checker(ROOT)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_shortest_path_is_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/src/lib/correlationGraph.ts', 'shortestCorrelationPath', 'removedCorrelationPath')
            self.assert_fails_with(root, 'explainable shortest path')

    def test_external_graph_engines_are_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/package.json', '"react":', '"cytoscape": "1.0.0",\n    "react":')
            self.assert_fails_with(root, 'no graph runtime dependency')

    def test_three_runtime_is_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/package.json', '"three":', '"unused-three":')
            self.assert_fails_with(root, 'declared 3D runtime')

    def test_react_18_compatible_fiber_major_is_enforced(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/package.json', '"@react-three/fiber": "^8.', '"@react-three/fiber": "^9.')
            self.assert_fails_with(root, 'React 18 compatible fiber major')

    def test_webgl_scene_is_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/src/components/correlation/OrbitalScene.tsx', '<Canvas', '<NotACanvas')
            self.assert_fails_with(root, 'real WebGL scene')

    def test_scene_must_stay_code_split(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(
                root,
                'apps/frontend/src/components/CorrelationGraph.tsx',
                "import('./correlation/OrbitalScene')",
                "eagerImport('./correlation/OrbitalScene')",
            )
            self.assert_fails_with(root, 'scene is code split')

    def test_three_must_stay_in_its_own_chunk(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/vite.config.js', "name: 'three-vendor'", "name: 'app-vendor'")
            self.assert_fails_with(root, 'three isolated in its own chunk')

    def test_layout_must_not_import_the_renderer(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            path = root / 'apps/frontend/src/lib/correlationLayout3d.ts'
            path.write_text("import { Vector3 } from 'three'\n" + path.read_text())
            self.assert_fails_with(root, 'layout independent of the renderer')

    def test_layout_must_stay_deterministic(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/src/lib/correlationLayout3d.ts', 'hashUnit(node.id, ring * 31 + 7)', 'Math.random()')
            self.assert_fails_with(root, 'deterministic layout without randomness')

    def test_zero_link_layout_is_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/src/components/CorrelationGraph.tsx', 'contextTarget = Math.min(30', 'contextTarget = Math.min(0')
            self.assert_fails_with(root, 'layout survives zero links')

    def test_keyboard_reachable_nodes_are_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/src/components/correlation/NodeIndex.tsx', 'tabIndex={0}', 'tabIndex={-1}')
            self.assert_fails_with(root, 'keyboard reachable nodes outside the canvas')

    def test_non_webgl_fallback_is_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(
                root,
                'apps/frontend/src/components/CorrelationGraph.tsx',
                "t('correlationGraph.webglFallback')",
                "'unavailable'",
            )
            self.assert_fails_with(root, 'graceful non-WebGL fallback')

    def test_legacy_svg_renderer_cannot_return(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            path = root / 'apps/frontend/src/components/CorrelationGraph.tsx'
            path.write_text(path.read_text() + '\nconst legacy = projectOrbitRing3D\n')
            self.assert_fails_with(root, 'legacy SVG renderer removed')

    def test_cluster_markers_are_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/src/components/CorrelationGraph.tsx', 'showClusters', 'hiddenClusters')
            self.assert_fails_with(root, 'visual clustering by node kind')

    def test_route_hops_must_stay_numbered(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/src/index.css', 'lyra-route-hop', 'lyra-route-step')
            self.assert_fails_with(root, 'numbered route hops')

    def test_focus_announcement_is_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(
                root,
                'apps/frontend/src/components/CorrelationGraph.tsx',
                "t('correlationGraph.selectionAnnouncement'",
                "String(",
            )
            self.assert_fails_with(root, 'focus is announced to assistive technology')

    def test_tailwind_typescript_scan_is_required(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            self.mutate(root, 'apps/frontend/tailwind.config.js', '{js,jsx,ts,tsx}', '{js,jsx}')
            self.assert_fails_with(root, 'Tailwind scans TSX graph classes')

    def test_new_hud_copy_must_be_localized(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            self.fixture(root)
            path = root / 'apps/frontend/src/lib/translations.js'
            path.write_text(path.read_text().replace("focusMode:", "focusModeDe:", 1))
            self.assert_fails_with(root, 'new HUD copy is localized')


if __name__ == '__main__':
    unittest.main()
