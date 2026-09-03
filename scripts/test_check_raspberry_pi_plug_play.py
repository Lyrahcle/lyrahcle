import importlib.util
from pathlib import Path
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location("check_rpi", ROOT / "scripts/check-raspberry-pi-plug-play.py")
MODULE = importlib.util.module_from_spec(SPEC)
assert SPEC.loader
SPEC.loader.exec_module(MODULE)


class RaspberryPiContractTest(unittest.TestCase):
    def test_repository_profile_passes(self):
        self.assertEqual(MODULE.main(str(ROOT)), 0)

    def test_missing_profile_fails(self):
        with tempfile.TemporaryDirectory() as directory:
            with self.assertRaises(SystemExit):
                MODULE.main(directory)


if __name__ == "__main__":
    unittest.main()

