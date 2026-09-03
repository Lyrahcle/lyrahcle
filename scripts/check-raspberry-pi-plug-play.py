#!/usr/bin/env python3
"""Static contract for the Raspberry Pi plug-and-play station profile."""

from pathlib import Path
import sys


def main(root: str) -> int:
    base = Path(root)
    required = [
        "deploy/raspberry-pi/install.sh", "deploy/raspberry-pi/build-arm64.sh",
        "deploy/raspberry-pi/prepare-rootfs.sh", "deploy/raspberry-pi/bin/lyrahcle-enroll",
        "deploy/raspberry-pi/bin/lyrahcle-doctor", "deploy/raspberry-pi/bin/lyrahcle-secure-update",
        "deploy/raspberry-pi/systemd/lyrahcle-sdr-agent.service",
        "deploy/raspberry-pi/systemd/lyrahcle-agent-update.timer",
        "deploy/raspberry-pi/systemd/lyrahcle-sdr-agent.sysusers",
        "deploy/raspberry-pi/systemd/lyrahcle-sdr-agent.tmpfiles",
        "deploy/raspberry-pi/config/60-lyrahcle-rtl-sdr.rules",
        "scripts/schedule-agent-pass.py", "docs/raspberry-pi-station.md",
    ]
    missing = [name for name in required if not (base / name).is_file()]
    if missing:
        raise SystemExit("missing Raspberry Pi artifacts: " + ", ".join(missing))
    unit = (base / required[6]).read_text(encoding="utf-8")
    updater = (base / "deploy/raspberry-pi/bin/lyrahcle-secure-update").read_text(encoding="utf-8")
    enroll = (base / "deploy/raspberry-pi/bin/lyrahcle-enroll").read_text(encoding="utf-8")
    scheduler = (base / "scripts/schedule-agent-pass.py").read_text(encoding="utf-8")
    checks = {
        "service is non-root": "User=lyrahcle" in unit,
        "service is hardened": "NoNewPrivileges=true" in unit and "ProtectSystem=strict" in unit,
        "one-time token scrub": "scrub_token()" in enroll,
        "HTTPS enrollment": 'parsed.scheme != "https"' in enroll,
        "signed updates": '"pkeyutl", "-verify"' in updater,
        "update rollback": "os.replace(backup, TARGET)" in updater,
        "pass scheduler dry-run default": "if not args.apply" in scheduler,
        "operator token avoids argv": "--token-stdin" in scheduler,
    }
    failed = [name for name, okay in checks.items() if not okay]
    if failed:
        raise SystemExit("Raspberry Pi contract failed: " + ", ".join(failed))
    print(f"Raspberry Pi plug-and-play contract: {len(checks)} checks passed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1] if len(sys.argv) > 1 else "."))
