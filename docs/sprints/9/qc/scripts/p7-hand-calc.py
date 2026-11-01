#!/usr/bin/env python3
"""QC P7 — tự tính lại đáp án của BA bằng phân số (không dùng mã của dev).
  python3 p7-hand-calc.py grading   # 7 mẫu: Σ tiêu chí, kẹp, bước 0,25, trừ muộn  (expected_grading.json)
  python3 p7-hand-calc.py v2        # 30 sinh viên điểm học phần theo quy chế v2    (seed/expected_final_grades_v2.csv)
  python3 p7-hand-calc.py all
  python3 p7-hand-calc.py --largest-remainder 33,33,34 10   # chia phần trăm → max_score bước 0,25 (TC-P7-02-09)
Thoát 1 nếu có lệch."""
import csv, json, sys, os
from fractions import Fraction as F
ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), *[".."] * 5))
FX = f"{ROOT}/docs/specs/FEAT-assignments/fixtures"
def q(x): return F(str(x).replace(",", "."))
def step(x, s=F(1, 4)):   # làm tròn nửa lên về bước s
    n = x / s; k = int(n); r = n - k
    return (k + (1 if r >= F(1, 2) else 0)) * s
def fmt(x): return f"{float(x):.2f}"

def grading():
    d = json.load(open(f"{FX}/expected_grading.json")); bad = 0
    rub = {c["id"]: q(c["max_score"]) for c in json.load(open(f"{FX}/rubric-bt03.json"))["criteria"]}
    P_DAY, MAXD = F(1, 2), 2
    for s in d:
        e = s["expected"]; late = s["late_days"]
        if s.get("llm_pass1") is None:
            tot = F(0)
        else:
            tot = sum(min(max(step(q(c["score"])), F(0)), rub[c["id"]]) for c in s["llm_pass1"]["criteria"])
        pen = min(late, MAXD) * P_DAY; fin = max(F(0), tot - pen)
        ok = fmt(tot) == e["total_before_late"] and fmt(pen) == e["late_penalty"] and fmt(fin) == e["total"]
        print(("OK  " if ok else "LỆCH"), s["id"], "Σ", fmt(tot), "trừ", fmt(pen), "→", fmt(fin), "| BA:", e["total_before_late"], e["late_penalty"], e["total"])
        bad += not ok
    return bad

def v2():
    v1 = {r["account"]: r for r in csv.DictReader(open(f"{ROOT}/seed/expected_final_grades.csv"))}
    bad = 0; diff = 0
    for r in csv.DictReader(open(f"{ROOT}/seed/expected_final_grades_v2.csv")):
        a = v1[r["account"]]
        mid, wk, fin = q(r["midterm"]), (q(r["weekly_exam_t9"]) if r["weekly_exam_t9"] else F(0)), q(r["final_exam"])
        asg = q(r["assignment_bt03"]) if r["assignment_bt03"] else F(0)
        raw = (10 * mid + 10 * wk + 20 * asg) / 40 + q(a["bonus_applied"]) - q(a["penalty"])
        qt = min(max(raw, F(0)), F(10)); hp = (40 * qt + 60 * fin) / 100
        hp_r = step(hp, F(1, 2))
        ok = fmt(qt) == fmt(q(r["qt"])) and fmt(hp_r) == fmt(q(r["hp"]))
        diff += r["hp"] != a["hp"]
        if not ok: print("LỆCH", r["account"], "QT", fmt(qt), r["qt"], "HP", fmt(hp_r), r["hp"]); bad += 1
    print(f"v2: {30 - bad}/30 khớp; {diff}/30 khác bản 1 (BA ghi 22)")
    return bad

def lr(pcts, mx):
    p = [F(x) for x in pcts.split(",")]; mx = F(mx); assert sum(p) == 100
    raw = [mx * x / 100 / F(1, 4) for x in p]; base = [int(x) for x in raw]
    left = int(mx / F(1, 4)) - sum(base)
    for i in sorted(range(len(p)), key=lambda i: raw[i] - base[i], reverse=True)[:left]: base[i] += 1
    out = [b * F(1, 4) for b in base]; assert sum(out) == mx; print([fmt(x) for x in out]); return 0

if __name__ == "__main__":
    a = sys.argv[1:] or ["all"]
    if a[0] == "--largest-remainder": sys.exit(lr(a[1], a[2]))
    bad = (grading() if a[0] in ("grading", "all") else 0) + (v2() if a[0] in ("v2", "all") else 0)
    sys.exit(1 if bad else 0)
