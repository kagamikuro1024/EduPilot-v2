from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import httpx


def _run_id(explicit: str | None) -> str:
    return explicit or datetime.now(timezone.utc).strftime("current-agent-%Y%m%d-%H%M%S")


def _sha256(path: Path) -> str:
    h = hashlib.sha256()
    with path.open("rb") as handle:
        while True:
            chunk = handle.read(8192)
            if not chunk:
                break
            h.update(chunk)
    return h.hexdigest()


def _resolve_token(host: str, email: str | None, password: str | None) -> str:
    existing = os.getenv("BENCHMARK_BEARER_TOKEN", "").strip()
    if existing:
        return existing
    if not email or not password:
        raise RuntimeError(
            "No BENCHMARK_BEARER_TOKEN. Set BENCHMARK_LOGIN_EMAIL and BENCHMARK_LOGIN_PASSWORD "
            "or pass --login-email/--login-password."
        )
    with httpx.Client(base_url=host, timeout=30.0) as client:
        response = client.post(
            "/api/v1/auth/login",
            json={"email": email, "password": password},
            headers={"Content-Type": "application/json"},
        )
        response.raise_for_status()
        payload = response.json()
        token = str(payload.get("token", "")).strip()
        if not token:
            raise RuntimeError("Login succeeded but token missing in response.")
        return token


def _run_collect(args: argparse.Namespace, predictions_path: Path) -> None:
    cmd = [
        sys.executable,
        "-m",
        "benchmarks.collect_predictions",
        "--dataset",
        args.dataset,
        "--host",
        args.host,
        "--endpoint-template",
        args.endpoint_template,
        "--output",
        str(predictions_path),
        "--timeout",
        str(args.timeout),
        "--sleep-seconds",
        str(args.sleep_seconds),
    ]
    if args.context_id:
        cmd.extend(["--context-id", args.context_id])
    if args.limit:
        cmd.extend(["--limit", str(args.limit)])
    subprocess.run(cmd, check=True)


def _run_eval_ragas(args: argparse.Namespace, predictions_path: Path) -> None:
    cmd = [
        sys.executable,
        "-m",
        "benchmarks.eval_ragas",
        "--dataset",
        args.dataset,
        "--predictions",
        str(predictions_path),
        "--run-id",
        args.run_id,
        "--output-dir",
        args.output_dir,
        "--concurrency",
        str(args.concurrency),
        "--model",
        args.ragas_model,
    ]
    subprocess.run(cmd, check=True)


def _run_postprocess(args: argparse.Namespace) -> None:
    subprocess.run(
        [
            sys.executable,
            "-m",
            "benchmarks.postprocess",
            "--run-id",
            args.run_id,
            "--output-dir",
            args.output_dir,
        ],
        check=True,
    )


def _write_run_meta(run_dir: Path, meta: dict[str, Any]) -> None:
    run_dir.mkdir(parents=True, exist_ok=True)
    (run_dir / "run_meta.json").write_text(json.dumps(meta, ensure_ascii=False, indent=2), encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Benchmark: preflight (optional) → collect → eval_ragas → postprocess; optional Locust Tier 1."
    )
    parser.add_argument("--host", default=os.getenv("BENCHMARK_HOST", "http://localhost:8080"))
    parser.add_argument("--dataset", default="data/benchmark_ground_truth.jsonl")
    parser.add_argument("--endpoint-template", default=os.getenv("BENCHMARK_ENDPOINT_TEMPLATE", "/api/v1/chat/sessions/{id}/ask-ai"))
    parser.add_argument("--context-id", default=os.getenv("BENCHMARK_CONTEXT_ID"))
    parser.add_argument("--run-id", default=None)
    parser.add_argument("--output-dir", default="reports/benchmark")
    parser.add_argument("--timeout", type=float, default=90.0)
    parser.add_argument("--sleep-seconds", type=float, default=0.0)
    parser.add_argument("--limit", type=int, default=None)
    parser.add_argument("--predictions-input", default=None, help="Skip collect; use existing predictions.jsonl")
    parser.add_argument("--skip-postprocess", action="store_true")
    parser.add_argument("--skip-preflight", action="store_true")
    parser.add_argument("--run-locust", action="store_true", help="Run Locust profiles into reports/benchmark/<run_id>/load/")
    parser.add_argument("--locust-profiles", default="baseline,stress,spike")
    parser.add_argument("--login-email", default=os.getenv("BENCHMARK_LOGIN_EMAIL"))
    parser.add_argument("--login-password", default=os.getenv("BENCHMARK_LOGIN_PASSWORD"))
    parser.add_argument("--ragas-model", default=os.getenv("BENCHMARK_RAGAS_MODEL", "gpt-4o-mini"))
    parser.add_argument("--concurrency", type=int, default=3)
    args = parser.parse_args()

    args.run_id = _run_id(args.run_id)
    run_dir = Path(args.output_dir) / args.run_id
    predictions_path = (
        Path(args.predictions_input) if args.predictions_input else run_dir / "predictions.jsonl"
    )

    meta: dict[str, Any] = {
        "run_id": args.run_id,
        "host": args.host,
        "dataset": args.dataset,
        "endpoint_template": args.endpoint_template,
        "predictions": str(predictions_path),
        "ragas_model": args.ragas_model,
        "dataset_sha256": _sha256(Path(args.dataset)),
    }

    if not args.skip_preflight:
        pre = [
            sys.executable,
            "-m",
            "benchmarks.preflight_check",
            "--dataset",
            args.dataset,
            "--require-auth",
            "--require-openai",
        ]
        try:
            subprocess.run(pre, check=True)
        except subprocess.CalledProcessError as exc:
            _write_run_meta(run_dir, {**meta, "status": "preflight_failed", "error": str(exc)})
            raise SystemExit("[preflight] failed") from exc

    if not args.predictions_input:
        try:
            token = _resolve_token(args.host, args.login_email, args.login_password)
            os.environ["BENCHMARK_BEARER_TOKEN"] = token
            meta["auth_mode"] = "token_or_login"
        except Exception as exc:
            raise SystemExit(f"[auth] {exc}") from exc
        try:
            _run_collect(args, predictions_path)
        except subprocess.CalledProcessError as exc:
            _write_run_meta(run_dir, {**meta, "status": "collect_failed", "error": str(exc)})
            raise SystemExit("[collect] failed") from exc
    elif not predictions_path.exists():
        raise SystemExit(f"Missing predictions: {predictions_path}")
    else:
        meta["auth_mode"] = "skipped_collect"
        run_dir.mkdir(parents=True, exist_ok=True)
        target_pred = run_dir / "predictions.jsonl"
        if predictions_path.resolve() != target_pred.resolve():
            shutil.copy2(predictions_path, target_pred)
            predictions_path = target_pred

    if args.run_locust:
        try:
            env = os.environ.copy()
            env["BENCHMARK_ENDPOINT_TEMPLATE"] = args.endpoint_template
            subprocess.run(
                [
                    sys.executable,
                    "-m",
                    "benchmarks.run_locust_profiles",
                    "--host",
                    args.host,
                    "--run-id",
                    args.run_id,
                    "--output-dir",
                    args.output_dir,
                    "--profiles",
                    args.locust_profiles,
                ],
                env=env,
                check=True,
            )
            meta["locust_profiles"] = args.locust_profiles
        except subprocess.CalledProcessError as exc:
            _write_run_meta(run_dir, {**meta, "status": "locust_failed", "error": str(exc)})
            raise SystemExit("[locust] failed") from exc

    try:
        _run_eval_ragas(args, predictions_path)
    except subprocess.CalledProcessError as exc:
        _write_run_meta(run_dir, {**meta, "status": "eval_ragas_failed", "error": str(exc)})
        raise SystemExit("[eval_ragas] failed") from exc

    if not args.skip_postprocess:
        try:
            _run_postprocess(args)
        except subprocess.CalledProcessError as exc:
            _write_run_meta(run_dir, {**meta, "status": "postprocess_failed", "error": str(exc)})
            raise SystemExit("[postprocess] failed") from exc

    _write_run_meta(run_dir, {**meta, "status": "completed"})
    print(f"Done. Run dir: {run_dir}")


if __name__ == "__main__":
    main()
