#!/usr/bin/env python3
"""Configure an SDR agent for the next predicted satellite pass."""

import argparse
from datetime import datetime, timedelta, timezone
import getpass
import json
import ssl
import sys
import time
from urllib.error import HTTPError, URLError
from urllib.parse import quote, urlencode, urlparse
from urllib.request import Request, urlopen


def api_json(url: str, token: str, ca: str | None, method: str = "GET", body: dict | None = None):
    parsed = urlparse(url)
    if parsed.scheme != "https" or not parsed.netloc:
        raise ValueError("backend URL must use HTTPS")
    data = json.dumps(body).encode() if body is not None else None
    request = Request(url, data=data, method=method, headers={
        "Authorization": f"Bearer {token}", "Accept": "application/json", "Content-Type": "application/json"
    })
    context = ssl.create_default_context(cafile=ca)
    with urlopen(request, context=context, timeout=15) as response:
        return json.load(response)


def parse_time(value: str) -> datetime:
    return datetime.fromisoformat(value.replace("Z", "+00:00")).astimezone(timezone.utc)


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--backend", required=True)
    parser.add_argument("--agent-id", required=True)
    parser.add_argument("--satellite-id", required=True)
    parser.add_argument("--latitude", required=True, type=float)
    parser.add_argument("--longitude", required=True, type=float)
    parser.add_argument("--altitude-m", default=0, type=float)
    parser.add_argument("--min-elevation", default=10, type=float)
    parser.add_argument("--frequency-hz", required=True, type=int)
    parser.add_argument("--interval-seconds", default=5, type=int)
    parser.add_argument("--lead-seconds", default=60, type=int)
    parser.add_argument("--ca")
    parser.add_argument("--token-stdin", action="store_true")
    parser.add_argument("--apply", action="store_true", help="wait until the pass window and publish the config")
    args = parser.parse_args()
    if not (-90 <= args.latitude <= 90 and -180 <= args.longitude <= 180):
        parser.error("invalid station coordinates")
    if not (100000 <= args.frequency_hz <= 6000000000 and 1 <= args.interval_seconds <= 3600):
        parser.error("frequency or capture interval outside agent bounds")
    token = (sys.stdin.readline() if args.token_stdin else getpass.getpass("Operator token (orbital.read + agents.manage): ")).strip()
    if not token:
        parser.error("operator token is required")
    base = args.backend.rstrip("/")
    query = urlencode({"latitude_deg": args.latitude, "longitude_deg": args.longitude,
                       "altitude_m": args.altitude_m, "hours": 72,
                       "min_elevation_deg": args.min_elevation, "limit": 1})
    try:
        prediction = api_json(f"{base}/api/v1/satellites/{quote(args.satellite_id, safe='')}/passes?{query}", token, args.ca)
        passes = prediction if isinstance(prediction, list) else prediction.get("passes", [])
        if not passes:
            print("No qualifying pass in the next 72 hours.", file=sys.stderr)
            return 3
        next_pass = passes[0]
        activate_at = parse_time(next_pass["aos"]) - timedelta(seconds=max(args.lead_seconds, 0))
        plan = {"agent_id": args.agent_id, "satellite_id": args.satellite_id,
                "activate_at": activate_at.isoformat(), "aos": next_pass["aos"], "los": next_pass["los"],
                "max_elevation_deg": next_pass.get("max_elevation_deg"), "frequency_hz": args.frequency_hz,
                "capture_interval_seconds": args.interval_seconds}
        print(json.dumps(plan, indent=2))
        if not args.apply:
            print("Dry run only. Re-run with --apply to wait and publish the signed configuration.")
            return 0
        delay = (activate_at - datetime.now(timezone.utc)).total_seconds()
        if delay > 72 * 3600:
            raise ValueError("activation lies outside the bounded prediction horizon")
        if delay > 0:
            print(f"Waiting {int(delay)} seconds until the guarded activation time...", file=sys.stderr)
            time.sleep(delay)
        config = {"frequency_hz": args.frequency_hz, "capture_interval_seconds": args.interval_seconds,
                  "satellite_id": args.satellite_id}
        result = api_json(f"{base}/api/v1/agents/{quote(args.agent_id, safe='')}/config", token, args.ca, "PUT", config)
        print(json.dumps({"published": True, "version": result.get("version"), "aos": next_pass["aos"], "los": next_pass["los"]}))
        return 0
    except (HTTPError, URLError, OSError, ValueError, KeyError, json.JSONDecodeError) as exc:
        detail = f"HTTP {exc.code}" if isinstance(exc, HTTPError) else type(exc).__name__
        print(f"Pass scheduling failed safely: {detail}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())

