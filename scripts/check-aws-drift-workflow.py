#!/usr/bin/env python3
"""Regression contract for the scheduled AWS drift-check workflow.

A workflow that runs unattended on a cron and touches production AWS
infrastructure is high-risk if it silently grows an `apply` step or starts
using long-lived static credentials. This check keeps both guarantees
enforced in CI rather than relying on review discipline alone.
"""
from __future__ import annotations
import pathlib
import re
import sys

APPLY_COMMAND_RE = re.compile(r"(?:terraform|tofu|\$IAC)\s+apply\b", re.IGNORECASE)

ROOT = pathlib.Path(sys.argv[1] if len(sys.argv) > 1 else ".").resolve()
WORKFLOW = ROOT / ".github" / "workflows" / "aws-drift-check.yml"
DRIFT_SCRIPT = ROOT / "scripts" / "aws-drift-check.sh"

errors = []

if not WORKFLOW.exists():
    print(f"aws drift workflow contract: FAIL - missing {WORKFLOW}")
    raise SystemExit(1)
if not DRIFT_SCRIPT.exists():
    print(f"aws drift workflow contract: FAIL - missing {DRIFT_SCRIPT}")
    raise SystemExit(1)

workflow_text = WORKFLOW.read_text()
script_text = DRIFT_SCRIPT.read_text()

if "schedule:" not in workflow_text:
    errors.append("workflow must run on a schedule (cron) trigger")
if "id-token: write" not in workflow_text:
    errors.append("workflow must request id-token: write for OIDC role assumption")
if "aws-actions/configure-aws-credentials" not in workflow_text:
    errors.append("workflow must assume an AWS role via aws-actions/configure-aws-credentials")
if "role-to-assume" not in workflow_text:
    errors.append("workflow must use role-to-assume (OIDC), not static keys")
if "aws-access-key-id" in workflow_text.lower() or "AWS_SECRET_ACCESS_KEY" in workflow_text:
    errors.append("workflow must not reference static AWS access keys")
if APPLY_COMMAND_RE.search(workflow_text):
    errors.append("workflow must never call `apply` - drift checks are read-only")
if APPLY_COMMAND_RE.search(script_text):
    errors.append("aws-drift-check.sh must never call `apply` - it is a read-only plan gate")
if "-lock=false" not in script_text:
    errors.append("aws-drift-check.sh should plan with -lock=false so it cannot block a real deploy's state lock")
if "detailed-exitcode" not in script_text:
    errors.append("aws-drift-check.sh must use `plan -detailed-exitcode` to distinguish drift (2) from no-op (0)")

if errors:
    print("aws drift workflow contract: FAIL")
    for item in errors:
        print(" -", item)
    raise SystemExit(1)

print("aws drift workflow contract: OK")
