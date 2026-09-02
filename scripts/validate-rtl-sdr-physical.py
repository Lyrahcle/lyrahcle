#!/usr/bin/env python3
"""Run or analyse an RTL-SDR physical validation capture.

The command is deliberately an operator/lab tool under scripts/: application runtime
never spawns SDR utilities. A successful report proves only the checks represented
in that report; it does not mark the whole Lyrahcle physical-hardware gate complete.
"""
from __future__ import annotations

import argparse
import csv
import hashlib
import json
import math
import shutil
import subprocess
import sys
import tempfile
from dataclasses import asdict, dataclass
from datetime import datetime, timezone
from pathlib import Path


@dataclass
class SweepSummary:
    rows: int
    bins: int
    frequency_min_hz: int
    frequency_max_hz: int
    power_min_dbm: float
    power_max_dbm: float
    power_mean_dbm: float
    finite_values: bool


def sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open('rb') as fh:
        for chunk in iter(lambda: fh.read(1024 * 1024), b''):
            h.update(chunk)
    return h.hexdigest()


def analyse_rtl_power(path: Path) -> SweepSummary:
    rows = bins = 0
    min_freq: int | None = None
    max_freq: int | None = None
    powers: list[float] = []
    with path.open(newline='', encoding='utf-8') as fh:
        reader = csv.reader(fh)
        for line_no, row in enumerate(reader, 1):
            if not row or all(not cell.strip() for cell in row):
                continue
            if len(row) < 7:
                raise ValueError(f'line {line_no}: expected rtl_power CSV with at least 7 columns')
            try:
                start = int(float(row[2].strip()))
                stop = int(float(row[3].strip()))
                step = float(row[4].strip())
                values = [float(value.strip()) for value in row[6:] if value.strip()]
            except ValueError as exc:
                raise ValueError(f'line {line_no}: invalid numeric rtl_power field') from exc
            if start <= 0 or stop <= start or step <= 0 or not values:
                raise ValueError(f'line {line_no}: invalid frequency range or empty power bins')
            rows += 1
            bins += len(values)
            min_freq = start if min_freq is None else min(min_freq, start)
            max_freq = stop if max_freq is None else max(max_freq, stop)
            powers.extend(values)
    if not rows or min_freq is None or max_freq is None or not powers:
        raise ValueError('rtl_power CSV contains no usable sweep data')
    finite = all(math.isfinite(value) for value in powers)
    if not finite:
        raise ValueError('rtl_power CSV contains non-finite power values')
    return SweepSummary(
        rows=rows,
        bins=bins,
        frequency_min_hz=min_freq,
        frequency_max_hz=max_freq,
        power_min_dbm=round(min(powers), 3),
        power_max_dbm=round(max(powers), 3),
        power_mean_dbm=round(sum(powers) / len(powers), 3),
        finite_values=True,
    )


def run(command: list[str], timeout: int) -> subprocess.CompletedProcess[str]:
    return subprocess.run(command, text=True, capture_output=True, timeout=timeout, check=False)


def live_capture(args: argparse.Namespace, work: Path) -> tuple[Path, dict]:
    rtl_test = shutil.which('rtl_test')
    rtl_power = shutil.which('rtl_power')
    if not rtl_test or not rtl_power:
        missing = ', '.join(name for name, value in [('rtl_test', rtl_test), ('rtl_power', rtl_power)] if not value)
        raise RuntimeError(f'missing required RTL-SDR utilities: {missing}')

    probe = run([rtl_test, '-t'], args.probe_timeout)
    if probe.returncode != 0:
        detail = (probe.stderr or probe.stdout).strip().splitlines()
        raise RuntimeError(f'rtl_test probe failed: {detail[-1] if detail else "unknown error"}')

    output = work / 'rtl-power.csv'
    band = f'{args.start_hz}:{args.stop_hz}:{args.bin_hz}'
    sweep = run([rtl_power, '-f', band, '-i', str(args.interval_seconds), '-e', f'{args.duration_seconds}s', str(output)], args.duration_seconds + 20)
    if sweep.returncode != 0 or not output.exists():
        detail = (sweep.stderr or sweep.stdout).strip().splitlines()
        raise RuntimeError(f'rtl_power capture failed: {detail[-1] if detail else "no output produced"}')
    return output, {
        'probe_command': 'rtl_test -t',
        'sweep_band': band,
        'duration_seconds': args.duration_seconds,
        'interval_seconds': args.interval_seconds,
        'probe_excerpt': '\n'.join((probe.stderr or probe.stdout).strip().splitlines()[-8:]),
    }


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--input', type=Path, help='analyse an existing rtl_power CSV instead of accessing hardware')
    parser.add_argument('--output', type=Path, required=True, help='write the JSON evidence report here')
    parser.add_argument('--start-hz', type=int, default=430_000_000)
    parser.add_argument('--stop-hz', type=int, default=440_000_000)
    parser.add_argument('--bin-hz', type=int, default=25_000)
    parser.add_argument('--duration-seconds', type=int, default=30)
    parser.add_argument('--interval-seconds', type=int, default=5)
    parser.add_argument('--probe-timeout', type=int, default=15)
    parser.add_argument('--operator-note', default='')
    args = parser.parse_args(argv)

    if args.start_hz <= 0 or args.stop_hz <= args.start_hz or args.bin_hz <= 0:
        parser.error('frequency range must satisfy 0 < start-hz < stop-hz and bin-hz > 0')
    if not (5 <= args.duration_seconds <= 3600):
        parser.error('duration-seconds must be between 5 and 3600')
    if not (1 <= args.interval_seconds <= args.duration_seconds):
        parser.error('interval-seconds must be between 1 and duration-seconds')

    args.output.parent.mkdir(parents=True, exist_ok=True)
    temp_ctx = tempfile.TemporaryDirectory(prefix='lyrahcle-rtl-sdr-') if args.input is None else None
    try:
        live_meta: dict = {}
        if args.input is None:
            capture, live_meta = live_capture(args, Path(temp_ctx.name))
            source = 'physical-rtl-sdr'
        else:
            capture = args.input.resolve()
            if not capture.is_file():
                raise RuntimeError(f'input capture not found: {capture}')
            source = 'offline-rtl-power-csv'

        summary = analyse_rtl_power(capture)
        report = {
            'schema': 'lyrahcle.rtl-sdr-physical-validation.v1',
            'generated_at': datetime.now(timezone.utc).isoformat(),
            'source': source,
            'physical_hardware_observed': source == 'physical-rtl-sdr',
            'gate_complete': False,
            'gate_note': 'This report is evidence for one capture only; RTL-SDR/HackRF physical validation remains a separate multi-run v1 gate.',
            'capture_sha256': sha256(capture),
            'summary': asdict(summary),
            'requested': {
                'start_hz': args.start_hz,
                'stop_hz': args.stop_hz,
                'bin_hz': args.bin_hz,
                'duration_seconds': args.duration_seconds,
                'interval_seconds': args.interval_seconds,
            },
            'operator_note': args.operator_note,
            'live': live_meta,
        }
        args.output.write_text(json.dumps(report, indent=2, sort_keys=True) + '\n', encoding='utf-8')
        print(f'RTL-SDR validation evidence written to {args.output}')
        print(f"capture: {summary.rows} rows, {summary.bins} bins, {summary.frequency_min_hz}-{summary.frequency_max_hz} Hz")
        return 0
    except (RuntimeError, ValueError, subprocess.TimeoutExpired) as exc:
        print(f'ERROR: {exc}', file=sys.stderr)
        return 1
    finally:
        if temp_ctx is not None:
            temp_ctx.cleanup()


if __name__ == '__main__':
    raise SystemExit(main())
