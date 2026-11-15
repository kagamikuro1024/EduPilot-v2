#!/usr/bin/env python3
"""Mô phỏng CASCADE theo độ tin cậy trên kết quả thật của poc.py (results.json), không gọi model nào.

Luật: decision model (jev | gdec) trả lời trước; nếu confidence ≥ τ thì dùng luôn; nếu không thì dùng câu trả lời của LLM (gemini
qua 9router, đã chạy trên cùng mẫu). Trễ một mẫu = trễ decision + (nếu fallback) trễ LLM, tức là gọi tuần tự.
Chi phí [ước tính]:
  decision = token vào × giá niêm yết (Jev $0,042/1M; gemini-decision-1 $0,10/1M);
  LLM = (token vào của gdec + 50) × $0,75/1M + token ra đo được × $3,75/1M (Gemini 3.8 Flash trả phí, gọi thẳng).
  Số token đo qua 9router bị đội lên nên không dùng.
  python3 docs/research/poc/jev-decision/cascade.py > docs/research/poc/jev-decision/cascade.md
"""
import json
import statistics
from pathlib import Path

HERE = Path(__file__).resolve().parent
R = json.loads((HERE / "results.json").read_text())
PRICE = {"jev": 0.042e-6, "gdec": 0.10e-6}
TAUS = [0.5, 0.6, 0.7, 0.8, 0.9, 0.95, 0.98]
NAMES = {"A": "A tường lửa (200)", "B": "B khủng hoảng (37)", "C": "C định tuyến (61)", "D": "D chèn lệnh (12)"}


def conf(task, x):
    return x["p"] if task == "C" else max(x["p"], 1 - x["p"])


def pct(xs, q):
    xs = sorted(xs)
    return xs[max(0, int(round(q * len(xs))) - 1)]


def run(task, dec):
    D = {x["id"]: x for x in R[task][dec]["items"]}
    L = {x["id"]: x for x in R[task]["gemini"]["items"]}
    gtok = {x["id"]: x.get("in_tok") or 0 for x in R[task].get("gdec", {}).get("items", [])} or {k: 300 for k in D}
    rows = []
    llm_ok = [L[k] for k in D if "pred" in L[k]]
    llm_acc = sum(x["pred"] == x["label"] for x in llm_ok) / len(D)
    llm_cost = statistics.mean((gtok.get(k, 300) + 50) * 0.75e-6 + (L[k].get("out_tok") or 25) * 3.75e-6 for k in D)
    llm_lat = [L[k]["ms"] for k in D]
    miss_llm = sum(1 for k in D if D[k]["label"] is True and L[k].get("pred") is not True)
    rows.append(("chỉ LLM", 0.0, None, llm_acc, miss_llm, statistics.median(llm_lat), pct(llm_lat, .95), llm_cost))
    dec_acc = sum(x.get("pred") == x["label"] for x in D.values()) / len(D)
    miss_dec = sum(1 for x in D.values() if x["label"] is True and x.get("pred") is not True)
    dlat = [x["ms"] for x in D.values()]
    dcost = statistics.mean((x.get("in_tok") or 0) * PRICE[dec] for x in D.values())
    rows.append((f"chỉ {dec}", 1.0, dec_acc, dec_acc, miss_dec, statistics.median(dlat), pct(dlat, .95), dcost))
    for t in TAUS:
        fast = [k for k, x in D.items() if "pred" in x and conf(task, x) >= t]
        fs = set(fast)
        final = {k: (D[k]["pred"] if k in fs else L[k].get("pred")) for k in D}
        acc = sum(final[k] == D[k]["label"] for k in D) / len(D)
        facc = sum(D[k]["pred"] == D[k]["label"] for k in fast) / len(fast) if fast else None
        miss = sum(1 for k in D if D[k]["label"] is True and final[k] is not True)
        lat = [D[k]["ms"] + (0 if k in fs else L[k]["ms"]) for k in D]
        cost = statistics.mean((D[k].get("in_tok") or 0) * PRICE[dec] + (0 if k in fs else (gtok.get(k, 300) + 50) * 0.75e-6 + (L[k].get("out_tok") or 25) * 3.75e-6) for k in D)
        rows.append((f"{dec} τ={t}", len(fast) / len(D), facc, acc, miss, statistics.median(lat), pct(lat, .95), cost))
    if task != "C":  # bất đối xứng (chỉ được thêm cờ): "có" ≥ 0,5 → dùng ngay; "không" chỉ dùng ngay khi P(có) ≤ ε; vùng giữa → LLM
        for eps in (0.02, 0.05, 0.1, 0.2):
            fast = {k for k, x in D.items() if "pred" in x and (x["p"] >= 0.5 or x["p"] <= eps)}
            final = {k: (D[k]["pred"] if k in fast else L[k].get("pred")) for k in D}
            acc = sum(final[k] == D[k]["label"] for k in D) / len(D)
            facc = sum(D[k]["pred"] == D[k]["label"] for k in fast) / len(fast) if fast else None
            miss = sum(1 for k in D if D[k]["label"] is True and final[k] is not True)
            lat = [D[k]["ms"] + (0 if k in fast else L[k]["ms"]) for k in D]
            cost = statistics.mean((D[k].get("in_tok") or 0) * PRICE[dec] + (0 if k in fast else (gtok.get(k, 300) + 50) * 0.75e-6 + (L[k].get("out_tok") or 25) * 3.75e-6) for k in D)
            rows.append((f"{dec} bất đối xứng ε={eps}", len(fast) / len(D), facc, acc, miss, statistics.median(lat), pct(lat, .95), cost))
    return rows


def main():
    print("| Tác vụ | Cấu hình | % đi đường nhanh | Độ đúng phần nhanh | Độ đúng tổng | Sót dương | Trễ p50 / p95 (ms) | Chi phí / 1.000 yêu cầu (USD, ước tính) |")
    print("| --- | --- | --- | --- | --- | --- | --- | --- |")
    for task in "ABCD":
        for dec in ("gdec", "jev"):
            if dec not in R.get(task, {}):
                continue
            for i, (name, cov, facc, acc, miss, p50, p95, cost) in enumerate(run(task, dec)):
                if dec == "jev" and i == 0:
                    continue  # dòng "chỉ LLM" in một lần
                f = "–" if facc is None else f"{facc:.3f}"
                m = "–" if task == "C" else miss
                print(f"| {NAMES[task]} | {name} | {cov:.0%} | {f} | {acc:.3f} | {m} | {p50:.0f} / {p95:.0f} | {cost * 1000:.3f} |")


if __name__ == "__main__":
    main()
