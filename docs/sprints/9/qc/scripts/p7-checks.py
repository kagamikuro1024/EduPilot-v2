#!/usr/bin/env python3
"""QC P7 — cài lại độc lập `grading.Check` / `grading.Merge` (không đọc mã dev) và so với fixtures/expected_checks.json
(13 ca một lượt K01–K13 + 4 ca hai lượt D01–D04). Thoát 1 nếu lệch. Dùng ở pha 1 (kiểm đáp án BA) và pha 2 (chuẩn đối chiếu cho TestCheckTable/TestMergeTable)."""
import json, os, re, sys, unicodedata
from fractions import Fraction as F
FX = os.path.join(os.path.dirname(__file__), *[".."] * 4, "specs/FEAT-assignments/fixtures")
RUB = {c["id"]: F(c["max_score"]) for c in json.load(open(f"{FX}/rubric-bt03.json"))["criteria"]}
SCORE_RE = re.compile(r"^-?\d{1,3}(\.\d{1,2})?$")  # K04 có điểm âm → phải qua được Check (xem Q-06-c)
def fold(s):
    s = unicodedata.normalize("NFD", s.replace("đ", "d").replace("Đ", "D"))
    return re.sub(r"\s+", " ", "".join(c for c in s if unicodedata.category(c) != "Mn")).strip().lower()
def step(x, s=F(1, 4)):
    n = x / s; k = int(n); return (k + (1 if n - k >= F(1, 2) else 0)) * s
def fmt(x): return f"{float(x):.2f}"
def check(case):
    o = case["llm"]; text = fold(case["text"]); flags = []; dropped = []; ids = [c["id"] for c in o["criteria"]]
    if sorted(ids) != sorted(RUB) or len(set(ids)) != len(ids) or any(not SCORE_RE.match(str(c["score"])) for c in o["criteria"]):
        return {"valid": False, "error": "GRADING_OUTPUT_INVALID"}
    sc = {}
    for c in o["criteria"]:
        raw = F(c["score"]); v = min(max(step(raw), F(0)), RUB[c["id"]])
        if v != raw: flags.append("SCORE_ADJUSTED")
        sc[c["id"]] = v
        for q in c["evidence_quotes"]:
            if fold(q) not in text: dropped.append({"criterion": c["id"], "quote": q}); flags.append("EVIDENCE_NOT_FOUND")
    tot = sum(sc.values())
    if o.get("total") is not None and F(o["total"]) != tot: flags.append("TOTAL_MISMATCH")  # fixture: K03,K04,K06–K12 không có `total` (xem Q-06-b)
    if o["confidence"] < 0.6: flags.append("LOW_CONFIDENCE")
    flags += [f for f in o["flags"] if f == "OFF_TOPIC"]
    order = ["SCORE_ADJUSTED", "TOTAL_MISMATCH", "EVIDENCE_NOT_FOUND", "LOW_CONFIDENCE", "OFF_TOPIC"]
    return {"valid": True, "criteria_scores": {k: fmt(v) for k, v in sc.items()}, "total": fmt(tot),
            "flags": sorted(set(flags), key=order.index), "dropped_quotes": dropped}
def merge(case):
    a = [F(x) for x in case["pass1_scores"]]; b = [F(x) for x in case["pass2_scores"]]
    ids = sorted(RUB); dis = [{"criterion": i, "delta": fmt(abs(x - y))} for i, x, y in zip(ids, a, b) if abs(x - y) > 1]
    td = abs(sum(a) - sum(b)); flag = bool(dis) or td > 1
    return {"flags": ["NEEDS_REVIEW"] if flag else [], "total_delta": fmt(td), "disagree": dis, "total": fmt(sum(a))}
if __name__ == "__main__":
    c = json.load(open(f"{FX}/expected_checks.json")); bad = 0
    for k in c["single_pass"]:
        got = check(k); ok = got == k["expected"]; bad += not ok
        print("OK  " if ok else "LỆCH", k["id"], "" if ok else f"\n   QC : {got}\n   BA : {k['expected']}")
    for k in c["two_pass"]:
        got = merge(k); ok = got == k["expected"]; bad += not ok
        print("OK  " if ok else "LỆCH", k["id"], "" if ok else f"\n   QC : {got}\n   BA : {k['expected']}")
    print(f"RESULT: {'PASS' if not bad else 'FAIL'} ({17 - bad}/17 khớp độc lập)"); sys.exit(1 if bad else 0)
