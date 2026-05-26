#!/usr/bin/env python3
"""
submit_benchmarks.py — Submit every benchmark usage scenario YAML in this repo to GMT.

Walks benchmarks/<lang>/*.yml and invokes the external submit_software.py helper
(from gmt-helpers) once per scenario. Runs are named "<prefix>-<lang>-<benchmark>"
so they can be filtered in the GMT UI afterwards. Sleeps SLEEP_SECONDS between
submissions to avoid hammering the API.

Configurable via CLI flags: --machine-id, --token, --repo-url, --branch,
--submit-script, --name-prefix, --include-tests, --only-cluster, --dry-run.
"""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path
from typing import List, Tuple

DEFAULT_REPO_URL = "https://github.com/green-coding-solutions/mthesis.git"
DEFAULT_TOKEN = "xxx"
DEFAULT_BRANCH = "main"
DEFAULT_NAME_PREFIX = "mthesis"
DEFAULT_SCHEDULE_MODE = "one-off"
DEFAULT_SUBMIT_SCRIPT = Path("/Users/didi/code/gmt-helpers/api/submit_software.py")

REPO_ROOT = Path(__file__).resolve().parent.parent
BENCHMARKS_DIR = REPO_ROOT / "benchmarks"

SLEEP_SECONDS = 60
RATE_LIMIT_BACKOFF_SECONDS = 300  # 5 minutes
DEFAULT_MAX_RETRIES = 5

# GMT wraps the upstream GitHub failure as "Repository returned bad status code (403)".
# Treat that pattern (or any non-2xx bad status code) as a transient GitHub rate-limit hit.
RATE_LIMIT_PATTERN = re.compile(r"bad status code \(\d{3}\)", re.IGNORECASE)


@dataclass(frozen=True)
class Scenario:
    """A single usage-scenario YAML to submit."""
    language: str        # e.g. "go"
    benchmark: str       # YAML stem, e.g. "binary-trees" or "gmt-cluster-scenario"
    rel_path: Path       # path relative to repo root, e.g. benchmarks/go/binary-trees.yml


def discover_scenarios(
    benchmarks_dir: Path,
    include_tests: bool,
    only_cluster: bool,
) -> List[Scenario]:
    """
    Find every *.yml directly under benchmarks/<lang>/.

    - include_tests: when False, drops files ending in "_test.yml"
    - only_cluster:  when True, keeps only "gmt-cluster-scenario.yml" files
    Returns a list sorted by (language, benchmark).
    """
    if not benchmarks_dir.is_dir():
        raise FileNotFoundError(f"benchmarks dir not found: {benchmarks_dir}")

    scenarios: List[Scenario] = []
    for yml in benchmarks_dir.glob("*/*.yml"):
        language = yml.parent.name
        stem = yml.stem
        if only_cluster and stem != "gmt-cluster-scenario":
            continue
        if not include_tests and stem.endswith("_test"):
            continue
        scenarios.append(
            Scenario(
                language=language,
                benchmark=stem,
                rel_path=yml.relative_to(REPO_ROOT),
            )
        )
    scenarios.sort(key=lambda s: (s.language, s.benchmark))
    return scenarios


def build_submit_command(
    submit_script: Path,
    api_url: str,
    token: str,
    name: str,
    repo_url: str,
    branch: str,
    machine_id: str,
    schedule_mode: str,
    filename: str,
    email: str | None,
) -> List[str]:
    """Assemble the argv for one invocation of submit_software.py."""
    cmd: List[str] = [
        sys.executable, str(submit_script),
        "--api-url", api_url,
        "--token", token,
        "submit",
        "--name", name,
        "--repo-url", repo_url,
        "--branch", branch,
        "--machine-id", str(machine_id),
        "--schedule-mode", schedule_mode,
        "--filename", filename,
    ]
    if email:
        cmd += ["--email", email]
    return cmd


def submit_one(cmd: List[str], dry_run: bool, progress: str = "") -> Tuple[int, str]:
    """
    Run one submit_software.py invocation.

    Captures stdout/stderr, replays them on this process's streams, and returns
    (exit_code, combined_output). In dry-run mode no process is launched and
    (0, "") is returned. `progress` is a "[n/x]" tag prepended to log lines.
    """
    printable = " ".join(_shquote(a) for a in cmd)
    tag = f"{progress} " if progress else ""
    if dry_run:
        print(f"{tag}[dry-run] {printable}")
        return 0, ""
    print(f"{tag}[submit]  {printable}")
    result = subprocess.run(cmd, capture_output=True, text=True, check=False)
    if result.stdout:
        sys.stdout.write(result.stdout)
        sys.stdout.flush()
    if result.stderr:
        sys.stderr.write(result.stderr)
        sys.stderr.flush()
    return result.returncode, (result.stdout or "") + (result.stderr or "")


def _shquote(s: str) -> str:
    """Minimal shell-quote for printing commands; not used to actually invoke."""
    if not s or any(c in s for c in ' "\'$`\\'):
        return "'" + s.replace("'", "'\\''") + "'"
    return s


def parse_args(argv: List[str] | None = None) -> argparse.Namespace:
    p = argparse.ArgumentParser(
        description="Submit every benchmark usage scenario YAML in this repo to GMT.",
    )
    p.add_argument("--machine-id", required=True,
                   help="Target GMT machine ID (use submit_software.py list-machines).")
    p.add_argument("--token", default=DEFAULT_TOKEN,
                   help="GMT X-Authentication token.")
    p.add_argument("--repo-url", default=DEFAULT_REPO_URL,
                   help=f"Repository URL (default {DEFAULT_REPO_URL}).")
    p.add_argument("--branch", default=DEFAULT_BRANCH,
                   help=f"Branch to measure (default {DEFAULT_BRANCH}).")
    p.add_argument("--api-url", default="https://api.green-coding.io/",
                   help="GMT API base URL.")
    p.add_argument("--schedule-mode", default=DEFAULT_SCHEDULE_MODE,
                   help=f"Schedule mode passed to submit_software.py (default {DEFAULT_SCHEDULE_MODE}).")
    p.add_argument("--name-prefix", default=DEFAULT_NAME_PREFIX,
                   help=f"Run-name prefix; full name is <prefix>-<lang>-<benchmark> (default {DEFAULT_NAME_PREFIX}).")
    p.add_argument("--email", default=None,
                   help="Optional notification email forwarded to submit_software.py.")
    p.add_argument("--submit-script", default=str(DEFAULT_SUBMIT_SCRIPT), type=Path,
                   help=f"Path to submit_software.py (default {DEFAULT_SUBMIT_SCRIPT}).")
    p.add_argument("--benchmarks-dir", default=str(BENCHMARKS_DIR), type=Path,
                   help="Directory holding benchmarks/<lang>/*.yml.")
    p.add_argument("--include-tests", action="store_true",
                   help="Also submit *_test.yml smoke scenarios (default: skipped).")
    p.add_argument("--only-cluster", action="store_true",
                   help="Submit only gmt-cluster-scenario.yml files.")
    p.add_argument("--language", action="append", default=None,
                   help="Limit to one or more languages (repeatable).")
    p.add_argument("--sleep", type=float, default=SLEEP_SECONDS,
                   help=f"Seconds to wait between submissions (default {SLEEP_SECONDS}).")
    p.add_argument("--rate-limit-backoff", type=float, default=RATE_LIMIT_BACKOFF_SECONDS,
                   help=f"Seconds to wait after a GitHub rate-limit error before retrying "
                        f"(default {RATE_LIMIT_BACKOFF_SECONDS}).")
    p.add_argument("--max-retries", type=int, default=DEFAULT_MAX_RETRIES,
                   help=f"Maximum number of retries per submission when a rate-limit error "
                        f"is detected (default {DEFAULT_MAX_RETRIES}).")
    p.add_argument("--dry-run", action="store_true",
                   help="Print the commands that would be run without executing them.")
    return p.parse_args(argv)


def main(argv: List[str] | None = None) -> int:
    """Entry point: discover scenarios, submit each with a sleep gap, report a summary."""
    args = parse_args(argv)

    submit_script: Path = args.submit_script
    if not submit_script.is_file():
        print(f"error: submit script not found: {submit_script}", file=sys.stderr)
        return 2

    scenarios = discover_scenarios(
        benchmarks_dir=args.benchmarks_dir,
        include_tests=args.include_tests,
        only_cluster=args.only_cluster,
    )
    if args.language:
        wanted = {lang.lower() for lang in args.language}
        scenarios = [s for s in scenarios if s.language.lower() in wanted]

    if not scenarios:
        print("No scenarios matched the given filters.", file=sys.stderr)
        return 1

    print(f"Found {len(scenarios)} scenario(s) to submit:")
    for s in scenarios:
        print(f"  - {s.language}/{s.benchmark}  ({s.rel_path})")
    print()

    total = len(scenarios)
    max_attempts = max(1, args.max_retries + 1)
    for idx, scenario in enumerate(scenarios):
        progress = f"[{idx + 1}/{total}]"
        if idx > 0 and not args.dry_run:
            print(f"{progress} [wait]    sleeping {args.sleep:.0f}s before next submission...")
            time.sleep(args.sleep)

        run_name = f"{scenario.language}-{scenario.benchmark.replace('-', '_')}"
        cmd = build_submit_command(
            submit_script=submit_script,
            api_url=args.api_url,
            token=args.token,
            name=run_name,
            repo_url=args.repo_url,
            branch=args.branch,
            machine_id=args.machine_id,
            schedule_mode=args.schedule_mode,
            filename=str(scenario.rel_path),
            email=args.email,
        )

        for attempt in range(1, max_attempts + 1):
            attempt_tag = progress if attempt == 1 else f"{progress} (retry {attempt - 1}/{args.max_retries})"
            rc, output = submit_one(cmd, dry_run=args.dry_run, progress=attempt_tag)
            if rc == 0:
                break
            rate_limited = bool(RATE_LIMIT_PATTERN.search(output))
            if rate_limited and attempt < max_attempts and not args.dry_run:
                print(
                    f"{progress} [rate-limit] GitHub rate-limit detected for {run_name}; "
                    f"sleeping {args.rate_limit_backoff:.0f}s before retry "
                    f"{attempt}/{args.max_retries}...",
                    file=sys.stderr,
                )
                time.sleep(args.rate_limit_backoff)
                continue
            reason = "rate-limit retries exhausted" if rate_limited else f"exit {rc}"
            print(
                f"{progress} [error]   submission failed for {run_name} ({reason}); aborting.",
                file=sys.stderr,
            )
            return rc

    print()
    print(f"Done. {total} submission(s) succeeded.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
