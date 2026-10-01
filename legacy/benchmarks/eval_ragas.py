from __future__ import annotations

import argparse
import asyncio
import json
import os
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

import numpy as np
from openai import AsyncOpenAI
from ragas.llms import llm_factory
from ragas.metrics.collections import ContextPrecision, ContextRecall, Faithfulness

from benchmarks.common import load_jsonl, load_predictions, must_contain_pass, normalize_text


def _nanmean(values: list[float]) -> float:
    arr = np.array(values, dtype=float)
    if arr.size == 0:
        return float("nan")
    return float(np.nanmean(arr))


def _intent_match(expected: str, predicted: str) -> bool:
    exp = (expected or "").strip().upper()
    pred = (predicted or "").strip().upper()
    if not pred:
        return False
    return exp == pred


def _uncertain_behavior_pass(case: dict[str, Any], answer: str) -> bool:
    """Plan §4.2 / §5.4: UNCERTAIN cases should clarify scope, not fabricate."""
    if str(case.get("expected_intent", "")).upper() != "UNCERTAIN":
        return True
    low = normalize_text(answer)
    markers = (
        "?",
        "lam ro",
        "cu the",
        "ban muon",
        "em muon",
        "ngu canh",
        "ro hon",
        "y dinh",
        "hieu la",
    )
    return any(m in low for m in markers)


async def _score_one(
    sem: asyncio.Semaphore,
    faith: Faithfulness,
    c_recall: ContextRecall,
    c_prec: ContextPrecision,
    case: dict[str, Any],
    pred: dict[str, Any],
) -> dict[str, Any]:
    async with sem:
        q = str(case.get("query", ""))
        ref = str(case.get("expected_answer", ""))
        ans = str(pred.get("predicted_answer", ""))
        ctxs = pred.get("retrieved_contexts")
        if not isinstance(ctxs, list):
            meta = pred.get("metadata", {})
            if isinstance(meta, dict) and isinstance(meta.get("citations"), list):
                ctxs = [
                    str(c.get("snippet", "")).strip()
                    for c in meta["citations"]
                    if isinstance(c, dict) and str(c.get("snippet", "")).strip()
                ]
            else:
                ctxs = []
        ctxs = [str(c).strip() for c in ctxs if str(c).strip()]
        if not ctxs:
            ctxs = ["(Không có đoạn ngữ cảnh trích xuất — đánh giá RAGAS có thể bị hạn chế.)"]

        row: dict[str, Any] = {
            "case_id": str(case.get("case_id", "")),
            "faithfulness": float("nan"),
            "context_recall": float("nan"),
            "context_precision": float("nan"),
            "intent_match": _intent_match(str(case.get("expected_intent", "")), str(pred.get("predicted_intent", ""))),
            "must_contain_pass": must_contain_pass(case, ans),
            "uncertain_behavior_pass": _uncertain_behavior_pass(case, ans),
        }

        if not q.strip() or not ref.strip() or not ans.strip():
            row["error"] = "missing_query_reference_or_answer"
            return row

        try:
            fr = await faith.ascore(user_input=q, response=ans, retrieved_contexts=ctxs)
            row["faithfulness"] = float(fr.value)
        except Exception as exc:  # pragma: no cover
            row["faithfulness_error"] = str(exc)

        try:
            rr = await c_recall.ascore(user_input=q, retrieved_contexts=ctxs, reference=ref)
            row["context_recall"] = float(rr.value)
        except Exception as exc:  # pragma: no cover
            row["context_recall_error"] = str(exc)

        try:
            pr = await c_prec.ascore(user_input=q, reference=ref, retrieved_contexts=ctxs)
            row["context_precision"] = float(pr.value)
        except Exception as exc:  # pragma: no cover
            row["context_precision_error"] = str(exc)

        return row


async def _run_all(
    cases: list[dict[str, Any]],
    preds: dict[str, dict[str, Any]],
    model: str,
    concurrency: int,
) -> list[dict[str, Any]]:
    api_key = os.getenv("OPENAI_API_KEY", "").strip()
    if not api_key:
        raise RuntimeError("OPENAI_API_KEY is required for RAGAS Tier-2.")

    client = AsyncOpenAI(api_key=api_key)
    llm = llm_factory(model, client=client)
    faith = Faithfulness(llm=llm)
    c_recall = ContextRecall(llm=llm)
    c_prec = ContextPrecision(llm=llm)

    sem = asyncio.Semaphore(max(1, concurrency))

    async def _one(case: dict[str, Any]) -> dict[str, Any]:
        cid = str(case["case_id"])
        p = preds.get(cid)
        if not p:
            return {
                "case_id": cid,
                "faithfulness": float("nan"),
                "context_recall": float("nan"),
                "context_precision": float("nan"),
                "intent_match": False,
                "must_contain_pass": False,
                "uncertain_behavior_pass": False,
                "error": "missing_prediction",
            }
        return await _score_one(sem, faith, c_recall, c_prec, case, p)

    return await asyncio.gather(*[_one(c) for c in cases])


def main() -> None:
    parser = argparse.ArgumentParser(description="Tier-2 RAGAS evaluation (faithfulness, context precision/recall).")
    parser.add_argument("--dataset", required=True)
    parser.add_argument("--predictions", required=True)
    parser.add_argument("--run-id", default=None)
    parser.add_argument("--output-dir", default="reports/benchmark")
    parser.add_argument("--concurrency", type=int, default=3)
    parser.add_argument(
        "--model",
        default=os.getenv("BENCHMARK_RAGAS_MODEL", os.getenv("BENCHMARK_JUDGE_MODEL", "gpt-5.4-mini")),
    )
    args = parser.parse_args()

    rows = load_jsonl(args.dataset)
    for i, row in enumerate(rows, start=1):
        row.setdefault("case_id", f"case_{i:03d}")

    preds = load_predictions(args.predictions)
    run_id = args.run_id or datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    run_dir = Path(args.output_dir) / run_id
    run_dir.mkdir(parents=True, exist_ok=True)

    results = asyncio.run(_run_all(rows, preds, args.model, args.concurrency))

    per_path = run_dir / "ragas_per_row.jsonl"
    with per_path.open("w", encoding="utf-8") as handle:
        for r in results:
            handle.write(json.dumps(r, ensure_ascii=False) + "\n")

    intent_ok = sum(1 for r in results if r.get("intent_match")) / len(results) if results else 0.0
    must_ok = sum(1 for r in results if r.get("must_contain_pass")) / len(results) if results else 0.0

    uncertain_subset = [c for c in rows if str(c.get("expected_intent", "")).upper() == "UNCERTAIN"]
    uncertain_pass = 0
    uncertain_total = len(uncertain_subset)
    for c in uncertain_subset:
        cid = str(c["case_id"])
        r = next((x for x in results if x.get("case_id") == cid), {})
        if r.get("uncertain_behavior_pass"):
            uncertain_pass += 1
    uncertain_rate = (uncertain_pass / uncertain_total) if uncertain_total else float("nan")

    summary = {
        "run_id": run_id,
        "dataset": str(args.dataset),
        "predictions": str(args.predictions),
        "ragas_model": args.model,
        "faithfulness_mean": _nanmean([r.get("faithfulness", float("nan")) for r in results]),
        "context_precision_mean": _nanmean([r.get("context_precision", float("nan")) for r in results]),
        "context_recall_mean": _nanmean([r.get("context_recall", float("nan")) for r in results]),
        "intent_accuracy": intent_ok,
        "must_contain_pass_rate": must_ok,
        "uncertain_behavior_pass_rate": uncertain_rate,
        "rows": len(results),
        "generated_at": datetime.now(timezone.utc).isoformat(),
    }
    (run_dir / "ragas_summary.json").write_text(json.dumps(summary, ensure_ascii=False, indent=2), encoding="utf-8")

    print(f"Run id: {run_id}")
    print(f"Per-row: {per_path}")
    print(f"Summary: {run_dir / 'ragas_summary.json'}")


if __name__ == "__main__":
    main()
