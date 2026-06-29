from __future__ import annotations

import argparse
import json
import os
from pathlib import Path

from .common import load_jsonl


def _expected_pages_valid(row: dict, idx: int) -> tuple[bool, str]:
    """Plan §5.2: expected_pages must be [] or a list of integers."""
    if "expected_pages" not in row:
        return False, f"Row {idx} missing expected_pages"
    ep = row["expected_pages"]
    if ep is None:
        return False, f"Row {idx} expected_pages is null"
    if not isinstance(ep, list):
        return False, f"Row {idx} expected_pages must be a JSON array, got {type(ep).__name__}"
    for j, p in enumerate(ep, start=1):
        if not isinstance(p, int):
            return False, f"Row {idx} expected_pages[{j}] must be int, got {type(p).__name__}"
    return True, "expected_pages OK"


def _check_dataset(path: Path) -> tuple[bool, str]:
    try:
        rows = load_jsonl(path)
    except Exception as exc:  # pragma: no cover - safety in CLI tool
        return False, f"Dataset parse failed: {exc}"
    if len(rows) != 100:
        return False, f"Dataset must contain 100 rows, got {len(rows)}."
    required = {"query", "expected_intent", "expected_answer"}
    for idx, row in enumerate(rows, start=1):
        missing = [key for key in required if key not in row]
        if missing:
            return False, f"Row {idx} missing required fields: {', '.join(missing)}"
        ok, msg = _expected_pages_valid(row, idx)
        if not ok:
            return False, msg
    return True, "Dataset parse passed (100 rows, required fields + expected_pages)."


def _default_endpoint_template() -> str:
    return os.getenv("BENCHMARK_ENDPOINT_TEMPLATE", "/api/v1/chat/sessions/{id}/ask-ai").strip()


def _context_id_check() -> tuple[bool, str]:
    """
    BENCHMARK_CONTEXT_ID is only required for fixed-context workflows.
    Default collect creates a new chat session or forum thread per case.
    """
    template = _default_endpoint_template()
    if "{id}" not in template:
        return True, "No {id} in endpoint template."
    if "/api/v1/chat/sessions/" in template:
        return True, "Chat endpoint: collect creates a new session per case unless --context-id is set."
    if "/api/v1/threads/" in template and "ask-ai" in template:
        return True, "Thread ask-ai: collect creates a new thread per case unless --context-id is set."
    if os.getenv("BENCHMARK_CONTEXT_ID", "").strip():
        return True, "BENCHMARK_CONTEXT_ID is set."
    return (
        False,
        "Set BENCHMARK_CONTEXT_ID for this endpoint template, or use a supported collect template.",
    )


def _check_env(require_auth: bool, require_openai: bool) -> list[tuple[str, bool, str]]:
    checks: list[tuple[str, bool, str]] = []
    auth_ok = bool(os.getenv("BENCHMARK_BEARER_TOKEN")) or (
        bool(os.getenv("BENCHMARK_LOGIN_EMAIL")) and bool(os.getenv("BENCHMARK_LOGIN_PASSWORD"))
    )
    checks.append(
        (
            "auth",
            auth_ok if require_auth else True,
            "Auth via token or login envs available." if auth_ok else "Missing auth envs/token.",
        )
    )
    openai_ok = bool(os.getenv("OPENAI_API_KEY", "").strip())
    checks.append(
        (
            "openai",
            openai_ok if require_openai else True,
            "OPENAI_API_KEY set for RAGAS Tier-2." if openai_ok else "Missing OPENAI_API_KEY (required for RAGAS).",
        )
    )
    ctx_ok, ctx_msg = _context_id_check()
    checks.append(("context_id", ctx_ok, ctx_msg))
    return checks


def main() -> None:
    parser = argparse.ArgumentParser(description="Preflight checks for benchmark run readiness.")
    parser.add_argument("--dataset", default="data/benchmark_ground_truth.jsonl")
    parser.add_argument("--require-auth", action="store_true")
    parser.add_argument(
        "--require-openai",
        action="store_true",
        help="Require OPENAI_API_KEY for RAGAS Tier-2.",
    )
    parser.add_argument(
        "--require-judge",
        action="store_true",
        help="Deprecated alias for --require-openai.",
    )
    parser.add_argument("--output", default=None, help="Optional JSON output path.")
    args = parser.parse_args()

    require_openai = args.require_openai or args.require_judge

    dataset_ok, dataset_msg = _check_dataset(Path(args.dataset))
    env_checks = _check_env(args.require_auth, require_openai)
    env_ok = all(ok for _, ok, _ in env_checks)
    all_ok = dataset_ok and env_ok

    report = {
        "dataset": str(args.dataset),
        "dataset_ok": dataset_ok,
        "dataset_message": dataset_msg,
        "env_checks": [
            {"name": name, "ok": ok, "message": message}
            for name, ok, message in env_checks
        ],
        "overall_ok": all_ok,
    }

    if args.output:
        output_path = Path(args.output)
        output_path.parent.mkdir(parents=True, exist_ok=True)
        output_path.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")

    print(json.dumps(report, ensure_ascii=False, indent=2))
    if not all_ok:
        raise SystemExit(1)


if __name__ == "__main__":
    main()
