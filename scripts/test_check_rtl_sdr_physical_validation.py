import csv
import importlib.util
import json
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / 'scripts' / 'validate-rtl-sdr-physical.py'

spec = importlib.util.spec_from_file_location('rtl_validation', SCRIPT)
mod = importlib.util.module_from_spec(spec)
sys.modules[spec.name] = mod
spec.loader.exec_module(mod)


class RTLSDRPhysicalValidationTests(unittest.TestCase):
    def make_csv(self, path: Path):
        with path.open('w', newline='', encoding='utf-8') as fh:
            writer = csv.writer(fh)
            writer.writerow(['2026-09-02', '06:00:00', 430000000, 430100000, 25000, 4, -101.2, -98.5, -92.0, -99.1])
            writer.writerow(['2026-09-02', '06:00:05', 430000000, 430100000, 25000, 4, -100.0, -96.5, -88.2, -97.0])

    def test_analyse_rtl_power_summary(self):
        with tempfile.TemporaryDirectory() as td:
            capture = Path(td) / 'capture.csv'
            self.make_csv(capture)
            summary = mod.analyse_rtl_power(capture)
            self.assertEqual(summary.rows, 2)
            self.assertEqual(summary.bins, 8)
            self.assertEqual(summary.frequency_min_hz, 430000000)
            self.assertEqual(summary.frequency_max_hz, 430100000)
            self.assertTrue(summary.finite_values)

    def test_offline_report_never_claims_physical_gate_complete(self):
        with tempfile.TemporaryDirectory() as td:
            capture = Path(td) / 'capture.csv'
            report = Path(td) / 'report.json'
            self.make_csv(capture)
            result = subprocess.run([sys.executable, str(SCRIPT), '--input', str(capture), '--output', str(report)], text=True, capture_output=True)
            self.assertEqual(result.returncode, 0, result.stderr)
            payload = json.loads(report.read_text())
            self.assertFalse(payload['physical_hardware_observed'])
            self.assertFalse(payload['gate_complete'])
            self.assertEqual(payload['schema'], 'lyrahcle.rtl-sdr-physical-validation.v1')
            self.assertEqual(len(payload['capture_sha256']), 64)

    def test_rejects_malformed_capture(self):
        with tempfile.TemporaryDirectory() as td:
            capture = Path(td) / 'bad.csv'
            capture.write_text('not,rtl,power\n', encoding='utf-8')
            with self.assertRaises(ValueError):
                mod.analyse_rtl_power(capture)


if __name__ == '__main__':
    unittest.main()
