from __future__ import annotations

import argparse
import os
import subprocess
from pathlib import Path


def _locust_cmd(
    profile: str,
    host: str,
    output_prefix: Path,
) -> list[str]:
    # Plan §5.1: baseline = 1 user, 10 sequential questions (see locustfile); allow ~5m to finish.
    presets = {
        "baseline": {"users": 1, "spawn_rate": 1, "run_time": "5m"},
        "stress": {"users": 20, "spawn_rate": 2, "run_time": "5m"},
        "spike": {"users": 200, "spawn_rate": 100, "run_time": "10s"},
    }
    conf = presets[profile]
    return [
        "locust",
        "-f",
        "benchmarks/locustfile.py",
        "--host",
        host,
        "--users",
        str(conf["users"]),
        "--spawn-rate",
        str(conf["spawn_rate"]),
        "--run-time",
        str(conf["run_time"]),
        "--headless",
        "--csv",
        str(output_prefix),
        "--only-summary",
    ]


def main() -> None:
    parser = argparse.ArgumentParser(description="Run Locust profiles for benchmark Tier 1.")
    parser.add_argument("--host", default=os.getenv("BENCHMARK_HOST", "http://localhost:8080"))
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--output-dir", default="reports/benchmark")
    parser.add_argument(
        "--profiles",
        default="baseline,stress,spike",
        help="Comma-separated profiles from: baseline,stress,spike",
    )
    args = parser.parse_args()

    run_dir = Path(args.output_dir) / args.run_id / "load"
    run_dir.mkdir(parents=True, exist_ok=True)
    selected = [item.strip() for item in args.profiles.split(",") if item.strip()]

    for profile in selected:
        if profile not in {"baseline", "stress", "spike"}:
            raise SystemExit(f"Unknown profile: {profile}")
        env = os.environ.copy()
        env["BENCHMARK_SCENARIO"] = profile
        prefix = run_dir / f"{profile}"
        cmd = _locust_cmd(profile, args.host, prefix)

        is_windows = os.name == "nt"
        subprocess.run(cmd, env=env, check=True, shell=is_windows)
        print(f"Completed Locust profile: {profile}")


if __name__ == "__main__":
    main()
