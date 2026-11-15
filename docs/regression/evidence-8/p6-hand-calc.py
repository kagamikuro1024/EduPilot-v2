#!/usr/bin/env python3
"""QC P6: tính tay độc lập (không import tinh_diem_tham_chieu.py của BA, không đọc mã của dev).
Công thức chép từ `fixtures/quy-che-lop1.md` (điều 1–4), dùng fractions.Fraction (chính xác, không float).
Đối chiếu: seed/expected_final_grades.csv (30 dòng) và fixtures/cases-tong-hop.csv (18 ca).
Chạy từ gốc repo:  python3 docs/sprints/8/qc/scripts/p6-hand-calc.py        (rc=0 khi mọi dòng khớp)
In thêm phép tính từng bước: python3 … A B C sv04 sv05 sv06 sv13 sv09 sv20
      Bảng cả lớp với tham số khác (đối chiếu `grade-scheme/impact`): python3 … --table step=0.1 cap=1.5 free=1
"""
import csv, sys
from fractions import Fraction as F

FX = "docs/specs/FEAT-gradebook/fixtures/"
EXP = "seed/expected_final_grades.csv"
LETTERS = [(F(9), "A+"), (F(17, 2), "A"), (F(8), "B+"), (F(7), "B"), (F(13, 2), "C+"),
           (F(11, 2), "C"), (F(5), "D+"), (F(4), "D"), (F(0), "F")]


def fr(s):
    s = (s or "").strip()
    return None if s == "" else F(s)


def round_half_up(x, step):
    n = x / step
    return (n + F(1, 2)).__floor__() * step  # x,25→x,5 · x,75→x+1,0 · x,5 giữ (không "chẵn gần nhất")


def compute(gk, kt, ck, bonus, pen_events, absent, *, w=(20, 20, 60), step=F(1, 2), pen_on=True, cap=F(1), free=2, per=F(1, 2)):
    """trả (qt, hp_raw, hp, letter, verdict, lines) hoặc ('INCOMPLETE', thành phần thiếu)."""
    miss = [n for n, v in (("midterm", gk), ("weekly", kt), ("final", ck)) if v is None]
    if miss:
        return ("INCOMPLETE", miss)
    wm, ww, wf = map(F, w)
    wq = wm + ww
    qt0 = (wm * gk + ww * kt) / wq if wq else F(0)
    b = min(bonus, cap) if wq else F(0)
    p = (per * max(0, absent - free) + abs(pen_events)) if pen_on else F(0)
    qt = min(F(10), max(F(0), qt0 + b - p)) if wq else F(0)
    raw = (wq * qt + wf * ck) / 100 if wq else ck
    hp = round_half_up(raw, step)
    letter = next(l for t, l in LETTERS if hp >= t)
    verdict = "OK"
    if ck < 3:
        letter, verdict = "F", "LIET:final<3.0"
    elif absent >= 7:
        letter, verdict = "F", "LIET:absent>=7"
    lines = [f"QT0 = ({wm}·{gk} + {ww}·{kt})/{wq} = {float(qt0):.4f}", f"B = min({bonus}, {cap}) = {float(b)}",
             f"P = {per}·max(0,{absent}-{free}) + |{pen_events}| = {float(p)}", f"QT = {float(qt):.4f}",
             f"HP thô = ({wq}·QT + {wf}·{ck})/100 = {float(raw):.4f}", f"HP = {float(hp)}  → {letter} ({verdict})"]
    return qt, raw, hp, letter, verdict, lines


def table(step, cap, free):
    """In HP từng sinh viên lớp 1 với tham số tuỳ chọn (dùng đối chiếu `POST …/grade-scheme/impact` khi đổi số)."""
    for i in csv.DictReader(open(FX + "inputs-lop1.csv")):
        kt = fr(i["weekly_exam_t9"]) if i["weekly_exam_t9"].strip() else F(0)
        r = compute(fr(i["midterm"]), kt, fr(i["final_exam"]), fr(i["speak_bonus_points"]), fr(i["penalty_event_points"]),
                    int(i["absent"]), step=step, cap=cap, free=free)
        print(f"{i['account']:<10} {i['student_code']}  QT={float(r[0]):.4f}  HP_thô={float(r[1]):.4f}  HP={float(r[2])}  {r[3]} {r[4]}")


def main(show):
    bad = 0
    inp = {r["account"]: r for r in csv.DictReader(open(FX + "inputs-lop1.csv"))}
    alias = {"A": "sv.gioi", "B": "sv.kha", "C": "sv.nguyco"}
    for r in csv.DictReader(open(EXP)):
        i = inp[r["account"]]
        kt = fr(i["weekly_exam_t9"]) if i["weekly_exam_t9"].strip() else F(0)  # đã công bố, không dự → 0 (quy chế 1.4)
        qt, raw, hp, letter, verdict, lines = compute(fr(i["midterm"]), kt, fr(i["final_exam"]), fr(i["speak_bonus_points"]),
                                                      fr(i["penalty_event_points"]), int(i["absent"]))
        ok = (qt == fr(r["qt"]) and raw == fr(r["hp_raw"]) and hp == fr(r["hp"]) and letter == r["letter"] and verdict == r["verdict"])
        if not ok:
            bad += 1
            print(f"FAIL {r['account']}: tôi ra qt={float(qt)} raw={float(raw)} hp={float(hp)} {letter} {verdict}; CSV qt={r['qt']} raw={r['hp_raw']} hp={r['hp']} {r['letter']} {r['verdict']}")
        for s in show:
            if alias.get(s, s) == r["account"]:
                print(f"--- {s} ({r['account']})"); print("\n".join("   " + l for l in lines))
    print(f"expected_final_grades.csv: {30 - bad}/30 khớp")
    nbad = 0
    for c in csv.DictReader(open(FX + "cases-tong-hop.csv")):
        w = (int(c["w_midterm"]), int(c["w_weekly"]), int(c["w_final"]))
        st = c["weekly_state"]
        kt = fr(c["weekly_exam_t9"])
        if st in ("EXAM_UNPUBLISHED", "MANUAL_BLANK_UNPUBLISHED"):
            kt = None
        elif st in ("EXAM_PUBLISHED_NO_ATTEMPT", "MANUAL_BLANK_PUBLISHED"):
            kt = F(0)
        gk = fr(c["midterm"]) * 10 / F(c["midterm_max"])  # điểm item quy về thang 10 (S16: 10/30)
        res = compute(gk, kt, fr(c["final_exam"]), fr(c["speak_bonus_points"]), fr(c["penalty_event_points"]),
                      int(c["absent"]), w=w, step=F(c["round_step"]), pen_on=(c["penalty_rule"] == "ON"))
        if res[0] == "INCOMPLETE":
            ok = c["verdict"].startswith("INCOMPLETE")
        else:
            ok = (res[2] == F(c["hp"]) and res[3] == c["letter"] and res[4] == c["verdict"]
                  and res[1] == F(c["hp_raw"]))
        if not ok:
            nbad += 1
            print(f"FAIL ca {c['case']}: tôi ra {res[:5]}; CSV hp_raw={c['hp_raw']} hp={c['hp']} {c['letter']} {c['verdict']}")
    print(f"cases-tong-hop.csv: {18 - nbad}/18 khớp")
    return 1 if (bad or nbad) else 0


if __name__ == "__main__":
    a = sys.argv[1:]
    if a and a[0] == "--table":  # --table [step=0.5] [cap=1] [free=2]
        kv = dict(x.split("=") for x in a[1:])
        table(F(kv.get("step", "0.5")), F(kv.get("cap", "1")), int(kv.get("free", "2")))
        sys.exit(0)
    sys.exit(main(a))
