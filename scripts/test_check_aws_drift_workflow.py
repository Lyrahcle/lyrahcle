import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
CHECK = ROOT / 'scripts' / 'check-aws-drift-workflow.py'

GOOD_WORKFLOW = """
name: AWS Drift Check
on:
  schedule:
    - cron: '17 6 * * *'
permissions:
  id-token: write
jobs:
  drift:
    steps:
      - uses: aws-actions/configure-aws-credentials@v4
        with:
          role-to-assume: arn:aws:iam::123456789012:role/drift
      - run: ./scripts/aws-drift-check.sh
"""

GOOD_SCRIPT = """
#!/usr/bin/env bash
terraform plan -lock=false -detailed-exitcode
"""


def run(target):
    return subprocess.run([sys.executable, str(CHECK), str(target)], capture_output=True, text=True)


def write(fake: Path, workflow: str, script: str):
    wf_dir = fake / '.github' / 'workflows'
    wf_dir.mkdir(parents=True, exist_ok=True)
    (wf_dir / 'aws-drift-check.yml').write_text(workflow)
    scripts_dir = fake / 'scripts'
    scripts_dir.mkdir(parents=True, exist_ok=True)
    (scripts_dir / 'aws-drift-check.sh').write_text(script)


class AwsDriftWorkflowContractTests(unittest.TestCase):
    def test_current_tree_satisfies_contract(self):
        result = run(ROOT)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_missing_files_fail(self):
        with tempfile.TemporaryDirectory() as td:
            result = run(Path(td))
            self.assertNotEqual(result.returncode, 0)

    def test_well_formed_workflow_passes(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            write(fake, GOOD_WORKFLOW, GOOD_SCRIPT)
            result = run(fake)
            self.assertEqual(result.returncode, 0, result.stdout + result.stderr)

    def test_apply_in_workflow_fails(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            write(fake, GOOD_WORKFLOW + "\n      - run: terraform apply -auto-approve\n", GOOD_SCRIPT)
            result = run(fake)
            self.assertNotEqual(result.returncode, 0)
            self.assertIn('apply', result.stdout.lower())

    def test_static_credentials_fail(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            bad = GOOD_WORKFLOW + "\n      - env:\n          AWS_SECRET_ACCESS_KEY: dummy\n"
            write(fake, bad, GOOD_SCRIPT)
            result = run(fake)
            self.assertNotEqual(result.returncode, 0)

    def test_missing_schedule_trigger_fails(self):
        with tempfile.TemporaryDirectory() as td:
            fake = Path(td)
            bad = GOOD_WORKFLOW.replace("schedule:\n    - cron: '17 6 * * *'", "workflow_dispatch: {}")
            write(fake, bad, GOOD_SCRIPT)
            result = run(fake)
            self.assertNotEqual(result.returncode, 0)


if __name__ == '__main__':
    unittest.main()
