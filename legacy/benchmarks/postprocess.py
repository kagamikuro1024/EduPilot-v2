from __future__ import annotations

import argparse
import csv
import json
from pathlib import Path
from statistics import mean
from typing import Any

TARGETS = {
    "ttft_p95_ms": 3000,
    "context_precision": 0.90,
    "faithfulness": 0.95,
    "cache_hit_rate": 0.30,
    "stress_ccu": 50,
}


def _load_json(path: Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def _load_jsonl(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    if not path.exists():
        return rows
    for line in path.read_text(encoding="utf-8").splitlines():
        line = line.strip()
        if line:
            rows.append(json.loads(line))
    return rows


def _read_locust_stats(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {}
    with path.open("r", encoding="utf-8-sig", newline="") as handle:
        rows = list(csv.DictReader(handle))
    ask_rows = [r for r in rows if (r.get("Name", "").startswith("ask_ai:"))]
    ttft_rows = [r for r in rows if (r.get("Name", "").startswith("ttft:"))]
    response_time_p95 = max((float(r.get("95%", 0.0)) for r in ask_rows), default=0.0)
    reqs = sum(float(r.get("Request Count", 0.0)) for r in ask_rows)
    failures = sum(float(r.get("Failure Count", 0.0)) for r in ask_rows)
    ttft_p95 = max((float(r.get("95%", 0.0)) for r in ttft_rows), default=0.0)
    return {
        "request_count": int(reqs),
        "failure_count": int(failures),
        "failure_rate": (failures / reqs) if reqs else 0.0,
        "response_time_p95_ms": response_time_p95,
        "ttft_p95_ms": ttft_p95,
    }


def _tier1_summary(run_dir: Path) -> dict[str, Any]:
    load_dir = run_dir / "load"
    profiles: dict[str, Any] = {}
    for profile in ("baseline", "stress", "spike"):
        stats = _read_locust_stats(load_dir / f"{profile}_stats.csv")
        if stats:
            profiles[profile] = stats
    if not profiles:
        return {
            "source": "MISSING",
            "profiles": {},
            "ttft_p95_ms": None,
            "stress_users_configured": TARGETS["stress_ccu"],
        }
    ttft_max = max(float(p.get("ttft_p95_ms", 0.0)) for p in profiles.values())
    return {
        "source": "locust_csv",
        "profiles": profiles,
        "ttft_p95_ms": ttft_max,
        "stress_users_configured": TARGETS["stress_ccu"],
    }


def _tier3_from_predictions(pred_rows: list[dict[str, Any]]) -> dict[str, Any]:
    cache_hits = 0
    cache_total = 0
    in_tok: list[int] = []
    out_tok: list[int] = []
    for row in pred_rows:
        meta = row.get("metadata", {})
        if isinstance(meta, dict):
            if isinstance(meta.get("cache_hit"), bool):
                cache_total += 1
                if meta["cache_hit"]:
                    cache_hits += 1
            usage = meta.get("usage")
            if isinstance(usage, dict):
                if usage.get("input_tokens") is not None:
                    in_tok.append(int(usage["input_tokens"]))
                if usage.get("output_tokens") is not None:
                    out_tok.append(int(usage["output_tokens"]))
    return {
        "cache_hit_rate": (cache_hits / cache_total) if cache_total else None,
        "cache_observation_count": cache_total,
        "avg_input_tokens": mean(in_tok) if in_tok else None,
        "avg_output_tokens": mean(out_tok) if out_tok else None,
    }


def _write_target_vs_actual(
    run_dir: Path,
    ragas: dict[str, Any],
    tier1: dict[str, Any],
    tier3: dict[str, Any],
) -> None:
    ttft = tier1.get("ttft_p95_ms")
    cp = ragas.get("context_precision_mean")
    fh = ragas.get("faithfulness_mean")
    ch = tier3.get("cache_hit_rate")

    lines = [
        f"# Target vs Actual - {run_dir.name}",
        "",
        "## Tier 2 (RAGAS)",
        f"- context_precision_mean: {cp}",
        f"- faithfulness_mean: {fh}",
        f"- context_recall_mean: {ragas.get('context_recall_mean')}",
        f"- intent_accuracy: {ragas.get('intent_accuracy')}",
        f"- must_contain_pass_rate: {ragas.get('must_contain_pass_rate')}",
        f"- uncertain_behavior_pass_rate: {ragas.get('uncertain_behavior_pass_rate')}",
        "",
        "## Tier 1 (Locust)",
        f"- ttft_p95_ms (max across profiles): {ttft if ttft is not None else 'MISSING'}",
        f"- Locust source: {tier1.get('source')}",
        "",
        "## Tier 3 (instrumentation)",
        f"- cache_hit_rate: {ch if ch is not None else 'MISSING (no cache_hit in prediction metadata)'}",
        f"- avg_input_tokens: {tier3.get('avg_input_tokens')}",
        f"- avg_output_tokens: {tier3.get('avg_output_tokens')}",
        "",
        "## Target check (Kế hoạch §4.1)",
        f"- TTFT p95 < {TARGETS['ttft_p95_ms']}ms: {'PASS' if isinstance(ttft, (int, float)) and ttft < TARGETS['ttft_p95_ms'] else 'MISSING_OR_FAIL'}",
        f"- context_precision >= {TARGETS['context_precision']}: {'PASS' if isinstance(cp, (int, float)) and cp >= TARGETS['context_precision'] else 'MISSING_OR_FAIL'}",
        f"- faithfulness >= {TARGETS['faithfulness']}: {'PASS' if isinstance(fh, (int, float)) and fh >= TARGETS['faithfulness'] else 'MISSING_OR_FAIL'}",
        f"- cache_hit_rate >= {TARGETS['cache_hit_rate']}: {'PASS' if isinstance(ch, float) and ch >= TARGETS['cache_hit_rate'] else 'MISSING_OR_FAIL'}",
    ]
    (run_dir / "target_vs_actual.md").write_text("\n".join(lines), encoding="utf-8")


def _write_kpi_gate(
    run_dir: Path,
    ragas: dict[str, Any],
    tier1: dict[str, Any],
    tier3: dict[str, Any],
) -> None:
    ttft = tier1.get("ttft_p95_ms")
    cp = ragas.get("context_precision_mean")
    fh = ragas.get("faithfulness_mean")
    ch = tier3.get("cache_hit_rate")

    gate = {
        "targets": TARGETS,
        "actual": {
            "ttft_p95_ms": ttft,
            "context_precision_mean": cp,
            "faithfulness_mean": fh,
            "context_recall_mean": ragas.get("context_recall_mean"),
            "intent_accuracy": ragas.get("intent_accuracy"),
            "must_contain_pass_rate": ragas.get("must_contain_pass_rate"),
            "uncertain_behavior_pass_rate": ragas.get("uncertain_behavior_pass_rate"),
            "cache_hit_rate": ch,
            "avg_input_tokens": tier3.get("avg_input_tokens"),
            "avg_output_tokens": tier3.get("avg_output_tokens"),
        },
        "status": {
            "tier1_locust_artifacts": tier1.get("source") == "locust_csv",
            "tier2_ragas_summary": (run_dir / "ragas_summary.json").exists(),
            "tier3_cache_instrumented": ch is not None,
            "tier3_token_instrumented": tier3.get("avg_input_tokens") is not None,
        },
    }
    (run_dir / "kpi_gate.json").write_text(json.dumps(gate, ensure_ascii=False, indent=2), encoding="utf-8")


def _write_final_report(run_dir: Path, gate: dict[str, Any]) -> None:
    st = gate["status"]
    lines = [
        f"# Final benchmark report - {run_dir.name}",
        "",
        "## Summary",
        f"- Tier 1 Locust CSV present: {st.get('tier1_locust_artifacts')}",
        f"- Tier 2 ragas_summary.json present: {st.get('tier2_ragas_summary')}",
        f"- Tier 3 cache metadata present: {st.get('tier3_cache_instrumented')}",
        f"- Tier 3 token usage present: {st.get('tier3_token_instrumented')}",
        "",
        "## Next actions",
        "- P0: Run `python -m benchmarks.run_locust_profiles --run-id <id>` if Tier 1 CSV is missing.",
        "- P1: Emit `metadata.cache_hit` and `metadata.usage` from Java SSE for Tier 3 KPIs.",
        "",
        "See `target_vs_actual.md` and `kpi_gate.json` for numeric comparison vs plan targets.",
    ]
    (run_dir / "final_benchmark_report.md").write_text("\n".join(lines), encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser(description="Postprocess benchmark run: KPI gate + reports.")
    parser.add_argument("--run-id", required=True)
    parser.add_argument("--output-dir", default="reports/benchmark")
    args = parser.parse_args()

    run_dir = Path(args.output_dir) / args.run_id
    ragas_path = run_dir / "ragas_summary.json"
    if not ragas_path.exists():
        raise SystemExit(f"Missing {ragas_path}. Run benchmarks.eval_ragas first.")

    ragas = _load_json(ragas_path)
    tier1 = _tier1_summary(run_dir)
    preds = _load_jsonl(run_dir / "predictions.jsonl")
    tier3 = _tier3_from_predictions(preds)

    (run_dir / "tier1_summary.json").write_text(json.dumps(tier1, ensure_ascii=False, indent=2), encoding="utf-8")
    (run_dir / "tier3_summary.json").write_text(json.dumps(tier3, ensure_ascii=False, indent=2), encoding="utf-8")

    _write_target_vs_actual(run_dir, ragas, tier1, tier3)
    _write_kpi_gate(run_dir, ragas, tier1, tier3)
    gate = _load_json(run_dir / "kpi_gate.json")
    _write_final_report(run_dir, gate)
    print(f"Postprocess done for run: {args.run_id}")


if __name__ == "__main__":
    main()
