#!/usr/bin/env python3
"""E4 — so sánh model (US-P10-10 AC6): chất lượng (E2, E3), độ trễ, chi phí / 100 yêu cầu. Chỉ đọc raw/ của E2, E3 (không gọi mạng).

  e4.py report   → e4.json / e4.md / e4.svg
Chi phí là QUY ĐỔI: token đo được × giá niêm yết của model tương ứng ngày đo; 9router là thuê bao nên chi phí biên thực = 0.
"""
import json

import lib
from lib import Fail

# Giá niêm yết USD / 1 triệu token (vào, ra), tra ngày 2026-11-15:
#   gemini-3.8-flash: ai.google.dev/gemini-api/docs/pricing ($0,75 / $3,75 tới 31/12/2026)
#   claude-sonnet-4-6: anthropic.com/news/claude-sonnet-4-6 ($3 / $15)
PRICE = {"ag/gemini-3.8-flash": (0.75, 3.75), "ag/claude-sonnet-4-6": (3.0, 15.0)}
ROUTER_OVERHEAD_IN = 2200  # token vào bị đội thêm do lời nhắc hệ thống của đường `ag/` của 9router (PoC Jev mục 4) — ước tính khi gọi thẳng


def main():
    prov = lib.load_raw("providers.json") or {}
    e2 = {t: lib.load_raw(f"e2-{t}.json") for t in ("P1", "P2")}
    e2r = json.loads((lib.OUT / "e2.json").read_text()) if (lib.OUT / "e2.json").exists() else None
    e3r = json.loads((lib.OUT / "e3.json").read_text()) if (lib.OUT / "e3.json").exists() else None
    if not any(e2.values()):
        raise Fail("chưa có raw/e2-P1.json, raw/e2-P2.json")
    rows = {}
    for t, raw in e2.items():
        model = prov.get(t, {}).get("model")
        if not raw:
            rows[t] = {"model": model, "status": "UNAVAILABLE", "reason": "chưa chạy E2 với model này"}
            continue
        chat = [a for a in raw.get("audit") or [] if a["task"] == "CHAT"]
        ok = [a for a in chat if a["status"] == "ok"]
        tin = [a["tokens_in"] for a in ok]
        tout = [a["tokens_out"] for a in ok]
        pin, pout = PRICE.get(model, (None, None))
        avg_in, avg_out = (sum(tin) / len(tin), sum(tout) / len(tout)) if ok else (None, None)
        cost = round(100 * (avg_in * pin + avg_out * pout) / 1e6, 4) if ok and pin is not None else None
        cost_direct = round(100 * (max(0, avg_in - ROUTER_OVERHEAD_IN) * pin + avg_out * pout) / 1e6, 4) if ok and pin is not None else None
        q = (e2r or {}).get("by_provider", {}).get(t, {})
        stab = next((s for s in (e3r or {}).get("stability", []) if s["b"].startswith(t)), None) if t != "P1" else next((s for s in (e3r or {}).get("stability", []) if s["b"] == "P1r2"), None)
        rows[t] = {"model": model, "status": "OK", "via": "9router (openai_compatible)",
                   "quality": {"e2_accuracy_answerable": q.get("accuracy_answerable"), "e2_fabrication_unanswerable": q.get("fabrication_unanswerable"),
                               "e2_at_080": q.get("current_at_t0"), "e3_vs_human": "WAITING_HUMAN" if (e3r or {}).get("status") != "OK" else next((h for h in e3r["human_vs_ai"] if h["a"].startswith(t)), None),
                               "e3_stability": stab and {k: stab[k] for k in ("a", "b", "qwk", "mae")}},
                   "latency_llm_ms": lib.pct([a["latency_ms"] for a in ok]), "latency_client_to_answer_ms": lib.pct([r["client_ms"] for r in raw["items"].values() if r.get("state") == "ANSWERED" and r.get("client_ms")]),
                   "calls": len(chat), "errors": len(chat) - len(ok), "tokens_in_avg": avg_in and round(avg_in), "tokens_out_avg": avg_out and round(avg_out),
                   "price_usd_per_1m": {"in": pin, "out": pout}, "cost_usd_per_100_measured": cost, "cost_usd_per_100_direct_est": cost_direct}
    obj = {"meta": lib.meta({"stack": "ep-qc", "providers": prov, "price_date": "2026-11-15"}), "status": "OK" if all(r["status"] == "OK" for r in rows.values()) else "PARTIAL", "by_provider": rows,
           "notes": ["Chi phí quy đổi: 9router là thuê bao, chi phí biên thực 0.", f"Token vào qua 9router đội ≈ {ROUTER_OVERHEAD_IN} do lời nhắc hệ thống đường ag/ (ước tính, PoC Jev mục 4); cột «gọi thẳng» trừ phần này."]}
    lib.svg_bars("e4.svg", [(t, [r.get("latency_llm_ms", {}).get("p50"), r.get("latency_llm_ms", {}).get("p95")]) for t, r in rows.items() if r["status"] == "OK"], ["p50", "p95"],
                 "E4 — độ trễ lời gọi sinh chữ (ms, llm_audit, qua 9router)", "ms", fmt="{:.0f}")
    f = lambda x, d=3: "–" if x is None else f"{x:.{d}f}".replace(".", ",")  # noqa: E731
    L = ["# E4 — so sánh model", "", f"Trạng thái: **{obj['status']}**. Stack `ep-qc` @ `{obj['meta']['commit']}`, {obj['meta']['timestamp'][:10]}. Cả hai model gọi qua 9router (openai_compatible), cùng stack, cùng seed, cùng bộ câu E2.", "",
         "| | " + " | ".join(f"{t}: `{r['model']}`" for t, r in rows.items()) + " |", "| --- |" + " --- |" * len(rows)]
    def row(name, fn):
        L.append(f"| {name} | " + " | ".join(fn(r) if r["status"] == "OK" else r["status"] for r in rows.values()) + " |")
    row("E2 đúng (câu trả lời được)", lambda r: f(r["quality"]["e2_accuracy_answerable"]))
    row("E2 bịa (câu không trả lời được)", lambda r: f(r["quality"]["e2_fabrication_unanswerable"]))
    row("E2 ở ngưỡng 0,80: escalate / lỗi lọt", lambda r: f"{f((r['quality']['e2_at_080'] or {}).get('escalate_rate'))} / {f((r['quality']['e2_at_080'] or {}).get('error_rate'))}")
    row("E3 so với người", lambda r: "WAITING_HUMAN" if r["quality"]["e3_vs_human"] == "WAITING_HUMAN" else f(r["quality"]["e3_vs_human"]["qwk"]))
    row("E3 ổn định (QWK)", lambda r: f(r["quality"]["e3_stability"]["qwk"]) + f" ({r['quality']['e3_stability']['a']}–{r['quality']['e3_stability']['b']})" if r["quality"]["e3_stability"] else "–")
    row("Lời gọi CHAT (lỗi)", lambda r: f"{r['calls']} ({r['errors']})")
    row("Độ trễ LLM p50 / p95 (ms)", lambda r: f"{r['latency_llm_ms'].get('p50')} / {r['latency_llm_ms'].get('p95')}")
    row("Đăng câu → có trả lời p50 / p95 (ms)", lambda r: f"{r['latency_client_to_answer_ms'].get('p50')} / {r['latency_client_to_answer_ms'].get('p95')}")
    row("Token TB vào / ra", lambda r: f"{r['tokens_in_avg']} / {r['tokens_out_avg']}")
    row("Giá niêm yết USD / 1M (vào / ra)", lambda r: f"{r['price_usd_per_1m']['in']} / {r['price_usd_per_1m']['out']}")
    row("Chi phí quy đổi / 100 yêu cầu (USD, token đo qua 9router)", lambda r: f(r["cost_usd_per_100_measured"], 4))
    row("Chi phí ước tính / 100 yêu cầu khi gọi thẳng (USD)", lambda r: f(r["cost_usd_per_100_direct_est"], 4))
    L += ["", "![](e4.svg)", "", "## Giới hạn", "", "- Dữ liệu mô phỏng (D44), cùng bộ 140 câu của E2; E3 chưa có điểm người (`WAITING_HUMAN`), nên chất lượng chấm chỉ có độ ổn định.",
          "- Độ trễ đo **qua 9router** (gồm cả proxy và hàng đợi của 9router), làn NEAR_REALTIME của Threads; không phải độ trễ API gốc của nhà cung cấp.",
          "- Chi phí là quy đổi theo giá niêm yết ngày 2026-11-15 (Gemini 3.8 Flash: giá giới thiệu tới 31/12/2026, từ 01/01/2027 gấp đôi); 9router là thuê bao nên chi phí biên thực bằng 0.",
          "- Nhà cung cấp: Google (gemini-3.8-flash) và Anthropic (claude-sonnet-4-6), đều qua 9router `ag/`; ngày đo ghi ở `meta`."]
    lib.write_report("e4", obj, L)


if __name__ == "__main__":
    try:
        main()
    except Fail as e:
        lib.die(f"LỖI: {e}", 2)
