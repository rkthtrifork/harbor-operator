#!/usr/bin/env python3
"""Validate that required GitHub check runs passed for a commit."""

from __future__ import annotations

import argparse
import json
import os
import shlex
import subprocess
import sys
import time
from collections.abc import Callable, Iterable


DEFAULT_REQUIRED_CHECKS = ("docs", "lint", "verify-generated", "test", "test-e2e")


def run_out(args: list[str]) -> str:
    try:
        return subprocess.run(args, check=True, text=True, capture_output=True).stdout.strip()
    except subprocess.CalledProcessError as err:
        print(f"Command failed with exit code {err.returncode}: {shlex.join(args)}", file=sys.stderr)
        if err.stdout:
            print("--- stdout ---", file=sys.stderr)
            print(err.stdout.rstrip(), file=sys.stderr)
        if err.stderr:
            print("--- stderr ---", file=sys.stderr)
            print(err.stderr.rstrip(), file=sys.stderr)
        raise


def list_check_runs(sha: str) -> list[dict[str, str]]:
    repository = os.environ.get("GITHUB_REPOSITORY", "")
    if not repository:
        raise RuntimeError("GITHUB_REPOSITORY is required to inspect check runs")

    output = run_out(
        [
            "gh",
            "api",
            "--paginate",
            f"repos/{repository}/commits/{sha}/check-runs",
            "--jq",
            ".check_runs[] | @json",
        ]
    )

    runs: list[dict[str, str]] = []
    for line in output.splitlines():
        if line:
            runs.append(json.loads(line))
    return runs


def missing_required_checks(sha: str, required_checks: Iterable[str]) -> list[str]:
    check_runs = list_check_runs(sha)
    return [
        check
        for check in required_checks
        if not any(
            run.get("name") == check and run.get("conclusion") == "success"
            for run in check_runs
        )
    ]


def required_checks_green(sha: str, required_checks: Iterable[str]) -> bool:
    return not missing_required_checks(sha, required_checks)


def wait_for_required_checks(
    sha: str,
    required_checks: Iterable[str],
    *,
    timeout_seconds: int,
    interval_seconds: int,
    logger: Callable[[str], None] = print,
) -> bool:
    deadline = time.monotonic() + timeout_seconds
    attempt = 1

    while True:
        missing = missing_required_checks(sha, required_checks)
        if not missing:
            logger(f"Required checks are green on {sha}")
            return True

        remaining = deadline - time.monotonic()
        if remaining <= 0:
            logger(f"Required checks did not become green on {sha}: {', '.join(missing)}")
            return False

        logger(f"Waiting for required checks on {sha} ({attempt}): {', '.join(missing)}")
        time.sleep(min(interval_seconds, max(1, int(remaining))))
        attempt += 1


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("sha", help="Commit SHA to inspect.")
    parser.add_argument(
        "checks",
        nargs="*",
        default=DEFAULT_REQUIRED_CHECKS,
        help="Required check run names. Defaults to the release gate checks.",
    )
    parser.add_argument("--wait", action="store_true", help="Wait until checks pass.")
    parser.add_argument("--timeout", type=int, default=1800, help="Wait timeout in seconds.")
    parser.add_argument("--interval", type=int, default=30, help="Wait interval in seconds.")
    return parser.parse_args()


def main() -> int:
    args = parse_args()

    if args.wait:
        return 0 if wait_for_required_checks(
            args.sha,
            args.checks,
            timeout_seconds=args.timeout,
            interval_seconds=args.interval,
        ) else 1

    missing = missing_required_checks(args.sha, args.checks)
    if missing:
        print(f"Required checks are not green on {args.sha}: {', '.join(missing)}", file=sys.stderr)
        return 1

    print(f"Required checks are green on {args.sha}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
