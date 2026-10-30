#!/usr/bin/env python3
"""Bảng tính tham chiếu của BA cho FEAT-gradebook (độc lập với mã Go của dev).

Dùng:  python3 tinh_diem_tham_chieu.py            # in ra, không ghi
       python3 tinh_diem_tham_chieu.py --write    # ghi seed/expected_final_grades.csv và cases-tong-hop.csv

Công thức và tham số lấy NGUYÊN VĂN từ quy-che-lop1.md (xem README.md). Chỉ dùng decimal; không float.
Script này là công cụ của BA để soạn đáp án; dev KHÔNG import và KHÔNG sửa; mã Go phải tự cho ra đúng các số trong CSV.
"""
import csv
import sys
from decimal import ROUND_HALF_UP, Decimal as D
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[3]  # .../docs/specs/FEAT-gradebook/fixtures -> gốc repo

# ---- Tham số của quy chế lớp 1 (quy-che-lop1.md) ----------------------------------------------------------------
W = {"midterm": D(20), "weekly": D(20), "final": D(60)}  # % của điểm học phần; QT = midterm + weekly = 40 %
BONUS_CAP = D("1.0")
FREE_ABSENCES, PER_ABSENCE = 2, D("0.5")
QT_FLOOR, QT_CAP = D(0), D(10)
STEP = D("0.5")  # làm tròn đến 0,5, nửa lên
LETTERS = [("9.0", "A+"), ("8.5", "A"), ("8.0", "B+"), ("7.0", "B"), ("6.5", "C+"), ("5.5", "C"), ("5.0", "D+"), ("4.0", "D"), ("0", "F")]
FAIL_FINAL_BELOW, FAIL_ABSENT_AT_LEAST = D("3.0"), 7


def rnd(x: D, step: D) -> D:
    return ((x / step).quantize(D(1), rounding=ROUND_HALF_UP)) * step


def letter(hp: D) -> str:
    for lo, name in LETTERS:
        if hp >= D(lo):
            return name
    return "F"


def compute(gk, pe, ck, bonus_pts, penalty_pts, absent, step=STEP):
    """gk / ck: Decimal; pe: Decimal hoặc None (vắng bài kiểm tra → 0 theo quy chế); ck None → chưa đủ điểm."""
    pe_used = D(0) if pe is None else pe
    qt_base = (W["midterm"] * gk + W["weekly"] * pe_used) / (W["midterm"] + W["weekly"])
    bonus = min(bonus_pts, BONUS_CAP)
    penalty = max(0, absent - FREE_ABSENCES) * PER_ABSENCE + penalty_pts
    qt_raw = qt_base + bonus - penalty
    qt = max(QT_FLOOR, min(QT_CAP, qt_raw))
    if ck is None:
        return dict(qt_base=qt_base, bonus=bonus, penalty=penalty, qt_raw=qt_raw, qt=qt, hp_raw=None, hp=None, letter=None, verdict="INCOMPLETE:final_exam")
    w_qt = W["midterm"] + W["weekly"]
    hp_raw = (w_qt * qt + W["final"] * ck) / D(100)
    hp = rnd(hp_raw, step)
    verdict, lt = "OK", letter(hp)
    if ck < FAIL_FINAL_BELOW:
        verdict, lt = "LIET:final<3.0", "F"
    elif absent >= FAIL_ABSENT_AT_LEAST:
        verdict, lt = "LIET:absent>=7", "F"
    return dict(qt_base=qt_base, bonus=bonus, penalty=penalty, qt_raw=qt_raw, qt=qt, hp_raw=hp_raw, hp=hp, letter=lt, verdict=verdict)


def f(x, places=2):
    return "" if x is None else f"{x:.{places}f}"


def explain(r, gk, pe, ck, bonus_pts, absent):
    pe_txt = "vang=0" if pe is None else f"{pe:.2f}"
    s = f"QT=(20x{gk:.2f}+20x{pe_txt})/40={r['qt_base']:.3f}; +B min({bonus_pts:.2f};1.00)={r['bonus']:.2f}; -P {r['penalty']:.2f} (vang {absent}, mien 2, 0.5/buoi)"
    s += f"; QT={r['qt_raw']:.3f}" + ("" if r["qt"] == r["qt_raw"] else f"->{r['qt']:.2f}")
    if r["hp_raw"] is None:
        return s + "; chua du diem"
    s += f"; HP=(40x{r['qt']:.2f}+60x{ck:.2f})/100={r['hp_raw']:.4f}->{r['hp']:.1f}"
    return s + ("" if r["verdict"] == "OK" else f"; {r['verdict']}")


def dec(s):
    return None if s == "" else D(s)


def main(write: bool):
    out_rows = []
    with open(HERE / "inputs-lop1.csv", newline="") as fh:
        for row in csv.DictReader(fh):
            gk, pe, ck = dec(row["midterm"]), dec(row["weekly_exam_t9"]), dec(row["final_exam"])
            b, pp, ab = D(row["speak_bonus_points"]), D(row["penalty_event_points"]), int(row["absent"])
            r = compute(gk, pe, ck, b, pp, ab)
            out_rows.append([row["account"], row["student_code"], f(gk), f(pe), f(ck), f(b), f(r["bonus"]), ab, f(r["penalty"]), f(r["qt_raw"], 3), f(r["qt"], 3), f(r["hp_raw"], 4), f(r["hp"], 1), r["letter"], r["verdict"], explain(r, gk, pe, ck, b, ab)])
    head = ["account", "student_code", "midterm", "weekly_exam_t9", "final_exam", "speak_bonus_points", "bonus_applied", "absent", "penalty", "qt_before_clamp", "qt", "hp_raw", "hp", "letter", "verdict", "explanation"]
    # ca tổng hợp: những điều kiện biên mà 30 sinh viên seed không chạm tới
    cases = [
        # id, gk, pe, ck, bonus_pts, penalty_pts, absent, step, ghi chú
        ("S01-san-QT", "0.00", "", "6.00", "0.25", "0.00", 4, "0.5", "QT âm → sàn 0"),
        ("S02-tran-cong-dung-1.00", "7.00", "7.00", "7.00", "1.00", "0.00", 0, "0.5", "B = 1,00 đúng trần"),
        ("S03-tran-cong-1.25", "7.00", "7.00", "7.00", "1.25", "0.00", 0, "0.5", "B = 1,25 → 1,00"),
        ("S04-vang-2-mien", "7.00", "7.00", "7.00", "0.00", "0.00", 2, "0.5", "vắng 2 → P = 0"),
        ("S05-vang-3-tru-0.5", "7.00", "7.00", "7.00", "0.00", "0.00", 3, "0.5", "vắng 3 → P = 0,5"),
        ("S06-su-kien-PENALTY", "7.00", "7.00", "7.00", "0.00", "0.50", 0, "0.5", "sự kiện PENALTY −0,5 cộng vào P"),
        ("S07-liet-vang-7", "8.00", "8.00", "8.00", "0.00", "0.00", 7, "0.5", "vắng 7 → liệt"),
        ("S08-vang-6-chua-liet", "8.00", "8.00", "8.00", "0.00", "0.00", 6, "0.5", "vắng 6 → chưa liệt (P = 2,0)"),
        ("S09-thieu-cuoi-ky", "8.00", "8.00", "", "0.00", "0.00", 0, "0.5", "thiếu cuối kỳ → chưa đủ điểm"),
        ("S10-lam-tron-0.1-nua-len", "7.00", "7.00", "7.25", "0.25", "0.00", 0, "0.1", "raw 7,25 → 7,3 (chẵn-gần-nhất sẽ ra 7,2)"),
        ("S11-lam-tron-0.5-duoi-1-chut", "6.25", "6.00", "6.00", "0.50", "0.00", 0, "0.5", "raw 6,25 (nửa lên → 6,5; chẵn-gần-nhất sẽ ra 6,0)"),
        ("S12-A-khong-len-A+", "8.00", "8.00", "9.00", "0.00", "0.00", 0, "0.5", "raw 8,60 → 8,5 → A (không phải A+)"),
    ]
    case_rows = []
    for cid, gk, pe, ck, bp, pp, ab, step, note in cases:
        r = compute(D(gk), dec(pe), dec(ck), D(bp), D(pp), ab, step=D(step))
        case_rows.append([cid, gk, pe, ck, bp, pp, ab, step, f(r["hp_raw"], 4), f(r["hp"], 2 if step == "0.1" else 1), r["letter"] or "", r["verdict"], note])
    chead = ["case", "midterm", "weekly_exam_t9", "final_exam", "speak_bonus_points", "penalty_event_points", "absent", "round_step", "hp_raw", "hp", "letter", "verdict", "note"]
    if write:
        for path, h, rows in ((ROOT / "seed" / "expected_final_grades.csv", head, out_rows), (HERE / "cases-tong-hop.csv", chead, case_rows)):
            with open(path, "w", newline="") as fh:
                w = csv.writer(fh, lineterminator="\n")
                w.writerow(h)
                w.writerows(rows)
        print("đã ghi", len(out_rows), "dòng + ", len(case_rows), "ca tổng hợp")
    else:
        for r in out_rows:
            print(",".join(map(str, r[:15])))
        for r in case_rows:
            print(",".join(map(str, r)))


if __name__ == "__main__":
    main("--write" in sys.argv)
