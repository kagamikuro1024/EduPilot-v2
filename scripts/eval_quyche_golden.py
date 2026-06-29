#!/usr/bin/env python3
"""
Offline evaluation for regulation RAG (pure Python API).

Reads JSONL golden dataset (default: data/golden_dataset_quyche.jsonl), calls
``src.tools.query_regulations`` — same pipeline as production — and prints a report.

Usage (from repo root):
    python scripts/eval_quyche_golden.py
    python scripts/eval_quyche_golden.py --dataset data/golden_dataset_quyche.jsonl --verbose

Requires: DATABASE_URL, OPENAI_API_KEY (via .env or environment).
"""

from __future__ import annotations

import argparse
import asyncio
import json
import logging
import sys
import time
import re
from pathlib import Path

# Repo root = parent of scripts/
REPO_ROOT = Path(__file__).resolve().parent.parent
if str(REPO_ROOT) not in sys.path:
    sys.path.insert(0, str(REPO_ROOT))


def normalize_vn(text: str) -> str:
    """Chuẩn hóa tiếng Việt: chữ thường, đồng nhất i/y, xóa khoảng trắng thừa và dấu câu."""
    if not text: return ""
    t = text.lower()
    # Đồng nhất i/y
    t = t.replace('kỳ', 'kì').replace('ký', 'kí').replace('lý', 'lí').replace('quy', 'qui')
    # Xóa dấu câu và ký tự đặc biệt
    t = re.sub(r'[^\w\s]', '', t)
    # Xóa khoảng trắng thừa
    return " ".join(t.split())


def _must_contain_pass(answer: str, substrings: list[str] | None) -> tuple[bool, list[str]]:
    if not substrings:
        return True, []
    missing: list[str] = []
    
    # Chuẩn hóa toàn bộ câu trả lời của LLM
    normalized_answer = normalize_vn(answer)
    
    for sub in substrings:
        # Chuẩn hóa từng keyword trước khi so sánh
        normalized_sub = normalize_vn(sub)
        if normalized_sub not in normalized_answer:
            missing.append(sub)
            
    return len(missing) == 0, missing


def _pages_pass(
    citations: list[dict],
    expected_pages: list[int] | None,
) -> tuple[bool, str]:
    if not expected_pages:
        return True, "no expected_pages"
    cited_pages = {int(c.get("page_number", 0) or 0) for c in citations if c.get("page_number")}
    cited_pages.discard(0)
    exp = set(expected_pages)
    if not cited_pages:
        return False, "no citations with page_number"
    inter = cited_pages & exp
    if inter:
        return True, f"cited pages {sorted(cited_pages)} intersect expected {sorted(exp)} -> {sorted(inter)}"
    return False, f"cited pages {sorted(cited_pages)} vs expected {sorted(exp)} (no overlap)"


async def _run_one(query: str) -> tuple[str, list[dict], float]:
    from src.tools import consume_last_rag_citations, query_regulations

    t0 = time.perf_counter()
    answer = await query_regulations(query)
    elapsed = time.perf_counter() - t0
    citations = consume_last_rag_citations()
    return answer, citations, elapsed


async def _async_main(args: argparse.Namespace) -> int:
    logging.basicConfig(level=logging.INFO if args.verbose else logging.WARNING, format="%(message)s")

    from src.database.connection import close_db_pool, init_db_pool
    from src.config import DATABASE_URL, OPENAI_API_KEY

    if not DATABASE_URL:
        print("ERROR: DATABASE_URL is not set.", file=sys.stderr)
        return 1
    if not OPENAI_API_KEY:
        print("ERROR: OPENAI_API_KEY is not set.", file=sys.stderr)
        return 1

    dataset_path = Path(args.dataset)
    if not dataset_path.is_file():
        print(f"ERROR: dataset not found: {dataset_path}", file=sys.stderr)
        return 1

    await init_db_pool(min_size=1, max_size=5)
    try:
        rows: list[dict] = []
        with open(dataset_path, encoding="utf-8") as f:
            for line in f:
                line = line.strip()
                if not line:
                    continue
                rows.append(json.loads(line))

        print(f"Dataset: {dataset_path} ({len(rows)} items)")
        print(f"Pipeline: src.tools.query_regulations (prod-equivalent)")
        print("-" * 72)

        ok_must = 0
        ok_pages = 0
        ok_both = 0
        failures: list[str] = []

        for i, row in enumerate(rows, start=1):
            query = row.get("query", "").strip()
            if not query:
                continue
            exp_pages = row.get("expected_pages")
            must_subs = row.get("must_contain_substrings")

            answer, citations, elapsed = await _run_one(query)

            mc_ok, missing = _must_contain_pass(answer, must_subs if isinstance(must_subs, list) else None)
            pg_ok, pg_detail = _pages_pass(citations if isinstance(citations, list) else [], exp_pages if isinstance(exp_pages, list) else None)

            if mc_ok:
                ok_must += 1
            if pg_ok:
                ok_pages += 1
            if mc_ok and pg_ok:
                ok_both += 1
            else:
                failures.append(f"#{i} query={query[:60]}...")

            status = "PASS" if (mc_ok and pg_ok) else "FAIL"
            print(f"\n[{i}/{len(rows)}] {status}  ({elapsed:.2f}s)")
            print(f"  query: {query[:200]}{'...' if len(query) > 200 else ''}")
            print(f"  must_contain: {'OK' if mc_ok else 'MISSING ' + str(missing)}")
            print(f"  pages: {pg_detail}")
            if args.verbose:
                preview = (answer or "").replace("\n", " ")[:400]
                print(f"  answer_preview: {preview}{'...' if len(answer or '') > 400 else ''}")
                if citations:
                    srcs = [
                        f"{c.get('source_file')}[p.{c.get('page_number')}]"
                        for c in citations[:5]
                        if isinstance(c, dict)
                    ]
                    print(f"  citations: {srcs}")

        print("\n" + "=" * 72)
        print("SUMMARY")
        print(f"  Total rows:           {len(rows)}")
        print(f"  must_contain pass:    {ok_must}/{len(rows)}")
        print(f"  expected_pages pass:  {ok_pages}/{len(rows)}")
        print(f"  both pass:            {ok_both}/{len(rows)}")
        if failures and not args.verbose:
            print(f"  Failed cases: {len(failures)}")
        return 0 if ok_both == len(rows) else 2
    finally:
        await close_db_pool()


def main() -> None:
    parser = argparse.ArgumentParser(description="Evaluate regulation RAG against golden JSONL.")
    parser.add_argument(
        "--dataset",
        default=str(REPO_ROOT / "data" / "golden_dataset_quyche.jsonl"),
        help="Path to JSONL golden file",
    )
    parser.add_argument("--verbose", "-v", action="store_true", help="Print answer preview and citations")
    args = parser.parse_args()
    code = asyncio.run(_async_main(args))
    raise SystemExit(code)


if __name__ == "__main__":
    main()
