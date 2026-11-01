#!/usr/bin/env python3
"""Bảng tính tham chiếu của BA cho FEAT-gradebook (độc lập với mã Go của dev).

Dùng:  python3 tinh_diem_tham_chieu.py            # in ra, không ghi
       python3 tinh_diem_tham_chieu.py --write    # ghi seed/expected_final_grades.csv, seed/expected_risk.csv và cases-tong-hop.csv

Công thức và tham số lấy NGUYÊN VĂN từ quy-che-lop1.md (xem README.md). Chỉ dùng decimal; không float.
Chuẩn hoá độ chính xác máy theo TLR-1: mọi phép chia dừng ở 20 chữ số; HP thô làm tròn về 10 chữ số ngay trước bước làm tròn nghiệp vụ.
Script này là công cụ của BA để soạn đáp án; dev KHÔNG import và KHÔNG sửa; mã Go phải tự cho ra đúng các số trong CSV.
"""
import csv
import sys
from decimal import ROUND_HALF_UP, Decimal as D, getcontext
from pathlib import Path

HERE = Path(__file__).resolve().parent
ROOT = HERE.parents[3]  # .../docs/specs/FEAT-gradebook/fixtures -> gốc repo

# ---- Tham số của quy chế lớp 1 (quy-che-lop1.md) ----------------------------------------------------------------
W0 = {"midterm": D(20), "weekly": D(20), "final": D(60)}  # % của điểm học phần; QT = midterm + weekly = 40 %
BONUS_CAP = D("1.0")
FREE_ABSENCES, PER_ABSENCE = 2, D("0.5")
QT_FLOOR, QT_CAP = D(0), D(10)
STEP = D("0.5")  # làm tròn đến 0,5, nửa lên
LETTERS = [("9.0", "A+"), ("8.5", "A"), ("8.0", "B+"), ("7.0", "B"), ("6.5", "C+"), ("5.5", "C"), ("5.0", "D+"), ("4.0", "D"), ("0", "F")]
FAIL_FINAL_BELOW, FAIL_ABSENT_AT_LEAST = D("3.0"), 7
E20, E10 = D("1e-20"), D("1e-10")  # TLR-1: DivRound(·, 20) và chuẩn hoá HP thô về 10 chữ số trước bước làm tròn

getcontext().prec = 60


def div(a: D, b: D) -> D:
    """DivRound(a, b, 20): mọi phép chia dừng ở 20 chữ số thập phân (nửa lên) — TLR-1."""
    return (a / b).quantize(E20, rounding=ROUND_HALF_UP)


def rnd(x: D, step: D) -> D:
    """TLR-13: HP = HP_thô.Div(step).Round(0).Mul(step); Round của shopspring là nửa xa 0 (= nửa lên vì HP >= 0)."""
    return ((x / step).quantize(D(1), rounding=ROUND_HALF_UP)) * step


def letter(hp: D) -> str:
    for lo, name in LETTERS:
        if hp >= D(lo):
            return name
    return "F"


def fx(x: D) -> str:
    """Số đầy đủ chữ số, không làm tròn, bỏ số 0 thừa (TLR-5)."""
    return "" if x is None else format(x.normalize(), "f")


# trạng thái ô "kiểm tra hằng tuần" (TLR-2): quy tắc 0 chỉ áp khi nguồn đã chốt cho item đó
SCORE, EXAM_PUBLISHED_NO_ATTEMPT, EXAM_UNPUBLISHED, MANUAL_BLANK_UNPUBLISHED, MANUAL_BLANK_PUBLISHED = (
    "SCORE", "EXAM_PUBLISHED_NO_ATTEMPT", "EXAM_UNPUBLISHED", "MANUAL_BLANK_UNPUBLISHED", "MANUAL_BLANK_PUBLISHED")


def compute(gk, pe, ck, bonus_pts, penalty_pts, absent, step=STEP, w=None, gk_max=D(10), weekly_state=SCORE, penalty_rule=True):
    """gk / ck: Decimal; pe: Decimal hoặc None; weekly_state: xem trên; ck None → chưa đủ điểm."""
    w = w or W0
    missing = []
    gk_s = div(gk * 10, gk_max)
    if pe is not None and weekly_state == SCORE:
        pe_s = pe
    elif weekly_state in (EXAM_PUBLISHED_NO_ATTEMPT, MANUAL_BLANK_PUBLISHED):
        pe_s = D(0)  # TLR-2: nguồn đã chốt → 0 theo MISSING_SCORE.zero_components
    else:
        pe_s = None
        missing.append("weekly")
    if ck is None:
        missing.append("final")
    w_qt, w_ck = w["midterm"] + w["weekly"], w["final"]
    bonus = min(bonus_pts, BONUS_CAP)
    # TLR-17: PENALTY.applicable = false → P = 0 (cả buổi vắng lẫn sự kiện PENALTY ghi tay)
    penalty = (max(0, absent - FREE_ABSENCES) * PER_ABSENCE + penalty_pts) if penalty_rule else D(0)
    base = dict(bonus=bonus, penalty=penalty, qt_base=None, qt_raw=None, qt=None, hp_raw=None, hp=None, letter=None)
    if missing:
        return {**base, "verdict": "INCOMPLETE:" + "+".join(missing)}
    if w_qt == 0:  # TLR-9: nhóm có w = 0 → bỏ số hạng, điểm cộng / trừ không có tác dụng
        qt_base = qt_raw = qt = None
        hp_raw = div(w_ck * ck, D(100))
    else:
        qt_base = div(w["midterm"] * gk_s + w["weekly"] * pe_s, w_qt)
        qt_raw = qt_base + bonus - penalty
        qt = max(QT_FLOOR, min(QT_CAP, qt_raw))
        hp_raw = div(w_qt * qt + w_ck * ck, D(100))
    hp_raw = hp_raw.quantize(E10, rounding=ROUND_HALF_UP)  # TLR-1: chuẩn hoá độ chính xác máy
    hp = rnd(hp_raw, step)
    verdict, lt = "OK", letter(hp)
    if ck < FAIL_FINAL_BELOW:
        verdict, lt = "LIET:final<3.0", "F"
    elif absent >= FAIL_ABSENT_AT_LEAST:
        verdict, lt = "LIET:absent>=7", "F"
    return dict(qt_base=qt_base, bonus=bonus, penalty=penalty, qt_raw=qt_raw, qt=qt, hp_raw=hp_raw, hp=hp, letter=lt, verdict=verdict)


def explain(r, gk, pe, ck, bonus_pts, absent):
    pe_txt = "vang=0" if pe is None else fx(pe)
    s = f"QT=(20x{fx(gk)}+20x{pe_txt})/40={fx(r['qt_base'])}; +B min({fx(bonus_pts)};1)={fx(r['bonus'])}; -P {fx(r['penalty'])} (vang {absent}, mien 2, 0.5/buoi)"
    s += f"; QT={fx(r['qt_raw'])}" + ("" if r["qt"] == r["qt_raw"] else f"->{fx(r['qt'])}")
    if r["hp_raw"] is None:
        return s + "; chua du diem"
    s += f"; HP=(40x{fx(r['qt'])}+60x{fx(ck)})/100={fx(r['hp_raw'])}->{fx(r['hp'])}"
    return s + ("" if r["verdict"] == "OK" else f"; {r['verdict']}")


def dec(s):
    return None if s == "" else D(s)


def exact(x: D, places: int) -> str:
    """In cố định `places` chữ số, kiểm tra không mất chữ số nào (không làm tròn âm thầm)."""
    if x is None:
        return ""
    q = x.quantize(D(1).scaleb(-places))
    assert q == x, f"{x} cần nhiều hơn {places} chữ số"
    return f"{q:.{places}f}"


def main(write: bool):
    out_rows, risk_in = [], []
    with open(HERE / "inputs-lop1.csv", newline="") as fh:
        for row in csv.DictReader(fh):
            gk, pe, ck = dec(row["midterm"]), dec(row["weekly_exam_t9"]), dec(row["final_exam"])
            b, pp, ab = D(row["speak_bonus_points"]), D(row["penalty_event_points"]), int(row["absent"])
            # weekly_exam_t9 trống = bài thi tuần 9 ĐÃ công bố mà sinh viên không có lượt GRADED → 0 (TLR-2)
            r = compute(gk, pe, ck, b, pp, ab, weekly_state=SCORE if pe is not None else EXAM_PUBLISHED_NO_ATTEMPT)
            risk_in.append((row["account"], row["student_code"], r["qt"]))
            out_rows.append([row["account"], row["student_code"], exact(gk, 2), exact(pe, 2), exact(ck, 2), exact(b, 2), exact(r["bonus"], 2), ab, exact(r["penalty"], 2), exact(r["qt_raw"], 3), exact(r["qt"], 3), exact(r["hp_raw"], 4), exact(r["hp"], 1), r["letter"], r["verdict"], explain(r, gk, pe, ck, b, ab)])
    head = ["account", "student_code", "midterm", "weekly_exam_t9", "final_exam", "speak_bonus_points", "bonus_applied", "absent", "penalty", "qt_before_clamp", "qt", "hp_raw", "hp", "letter", "verdict", "explanation"]
    # ca tổng hợp: điều kiện biên mà 30 sinh viên seed không chạm tới.
    # cột: id, gk, weekly, weekly_state, ck, bonus_pts, penalty_pts, absent, step, w_gk, w_weekly, w_final, gk_max, penalty_rule, ghi chú, cảnh báo của công thức
    cases = [
        ("S01-san-QT", "0.00", "", EXAM_PUBLISHED_NO_ATTEMPT, "6.00", "0.25", "0.00", 4, "0.5", 20, 20, 60, 10, "ON", "", "QT âm → sàn 0"),
        ("S02-tran-cong-dung-1.00", "7.00", "7.00", SCORE, "7.00", "1.00", "0.00", 0, "0.5", 20, 20, 60, 10, "ON", "", "B = 1,00 đúng trần"),
        ("S03-tran-cong-1.25", "7.00", "7.00", SCORE, "7.00", "1.25", "0.00", 0, "0.5", 20, 20, 60, 10, "ON", "", "B = 1,25 → 1,00"),
        ("S04-vang-2-mien", "7.00", "7.00", SCORE, "7.00", "0.00", "0.00", 2, "0.5", 20, 20, 60, 10, "ON", "", "vắng 2 → P = 0"),
        ("S05-vang-3-tru-0.5", "7.00", "7.00", SCORE, "7.00", "0.00", "0.00", 3, "0.5", 20, 20, 60, 10, "ON", "", "vắng 3 → P = 0,5"),
        ("S06-su-kien-PENALTY", "7.00", "7.00", SCORE, "7.00", "0.00", "0.50", 0, "0.5", 20, 20, 60, 10, "ON", "", "sự kiện PENALTY −0,5 cộng vào P"),
        ("S07-liet-vang-7", "8.00", "8.00", SCORE, "8.00", "0.00", "0.00", 7, "0.5", 20, 20, 60, 10, "ON", "", "vắng 7 → liệt"),
        ("S08-vang-6-chua-liet", "8.00", "8.00", SCORE, "8.00", "0.00", "0.00", 6, "0.5", 20, 20, 60, 10, "ON", "", "vắng 6 → chưa liệt (P = 2,0)"),
        ("S09-thieu-cuoi-ky", "8.00", "8.00", SCORE, "", "0.00", "0.00", 0, "0.5", 20, 20, 60, 10, "ON", "", "thiếu cuối kỳ → chưa đủ điểm"),
        ("S10-lam-tron-0.1-nua-len", "7.00", "7.00", SCORE, "7.25", "0.25", "0.00", 0, "0.1", 20, 20, 60, 10, "ON", "", "raw 7,25 → 7,3 (chẵn-gần-nhất sẽ ra 7,2)"),
        ("S11-lam-tron-0.5-duoi-1-chut", "6.25", "6.00", SCORE, "6.00", "0.50", "0.00", 0, "0.5", 20, 20, 60, 10, "ON", "", "raw 6,25 (nửa lên → 6,5; chẵn-gần-nhất sẽ ra 6,0)"),
        ("S12-A-khong-len-A+", "8.00", "8.00", SCORE, "9.00", "0.00", "0.00", 0, "0.5", 20, 20, 60, 10, "ON", "", "raw 8,60 → 8,5 → A (không phải A+)"),
        # TLR-2: số 0 theo MISSING_SCORE chỉ áp khi nguồn đã chốt cho item đó
        ("S13-bai-thi-chua-cong-bo", "8.00", "", EXAM_UNPUBLISHED, "8.00", "0.00", "0.00", 0, "0.5", 20, 20, 60, 10, "ON", "", "bài thi tuần chưa công bố → INCOMPLETE (không phải 0)"),
        ("S14-item-tay-trong-chua-cong-bo", "8.00", "", MANUAL_BLANK_UNPUBLISHED, "8.00", "0.00", "0.00", 0, "0.5", 20, 20, 60, 10, "ON", "", "item nhập tay chưa công bố cột, ô trống → INCOMPLETE"),
        ("S15-item-tay-trong-da-cong-bo", "8.00", "", MANUAL_BLANK_PUBLISHED, "8.00", "0.00", "0.00", 0, "0.5", 20, 20, 60, 10, "ON", "", "item nhập tay đã công bố cột, ô trống → 0"),
        # TLR-1: phép chia không hết (item 10/30) không được làm lệch biên làm tròn
        ("S16-chia-khong-het", "10.00", "0.00", SCORE, "8.75", "0.00", "0.00", 0, "0.5", 30, 10, 60, 30, "ON", "", "item 10/30 → 3,333…; HP thô đúng 6,25 → 6,5 (TLR-1: làm tròn trung gian ở 16 chữ số có thể ra 6,0)"),
        # TLR-9: nhóm có w = 0 → bỏ số hạng, không chia 0
        ("S17-nhom-QT-bang-0", "5.00", "5.00", SCORE, "7.40", "0.50", "0.00", 0, "0.5", 0, 0, 100, 10, "ON", "BONUS_NO_EFFECT", "w_QT = 0: HP = CK; điểm cộng không có tác dụng"),
        # TLR-17: PENALTY.applicable = false → sự kiện PENALTY và buổi vắng không bị trừ
        ("S18-PENALTY-khong-ap-dung", "7.00", "7.00", SCORE, "7.00", "0.00", "0.50", 5, "0.5", 20, 20, 60, 10, "OFF", "PENALTY_NOT_IN_SCHEME", "PENALTY.applicable=false: P = 0 dù vắng 5 và có sự kiện −0,5"),
    ]
    case_rows = []
    for cid, gk, pe, st, ck, bp, pp, ab, step, w1, w2, w3, gmax, prule, warn, note in cases:
        r = compute(D(gk), dec(pe), dec(ck), D(bp), D(pp), ab, step=D(step), w={"midterm": D(w1), "weekly": D(w2), "final": D(w3)}, gk_max=D(gmax), weekly_state=st, penalty_rule=prule == "ON")
        case_rows.append([cid, gk, pe, st, ck, bp, pp, ab, step, w1, w2, w3, gmax, prule, fx(r["hp_raw"]), (exact(r["hp"], 2) if step == "0.1" else exact(r["hp"], 1)) if r["hp"] is not None else "", r["letter"] or "", r["verdict"], warn, note])
    chead = ["case", "midterm", "weekly_exam_t9", "weekly_state", "final_exam", "speak_bonus_points", "penalty_event_points", "absent", "round_step", "w_midterm", "w_weekly", "w_final", "midterm_max", "penalty_rule", "hp_raw", "hp", "letter", "verdict", "scheme_warnings", "note"]
    # at-risk lớp 1 sau khi có tín hiệu GRADE (proposals #5): GRADE thấp khi QT hiện tại < 5,0 (nghiêm ngặt)
    # mức rủi ro: tuần W1 = tuần 9, W2 = tuần 8; tuần yếu = >= 2 tín hiệu thấp và >= 1 tín hiệu neo thấp (ATTENDANCE hoặc GRADE)
    level = {"sv.nguyco": "AT_RISK", "sv12": "WATCH", "sv19": "WATCH", "sv26": "WATCH"}
    act = {  # ràng buộc hoạt động (chat riêng + thread + bình luận) mỗi tuần, dev rải seed để thoả
        "sv24": (">=1", ">=1"), "sv25": (">=1", ">=1"), "sv27": (">=1", ">=1"), "sv29": (">=1", ">=1"),
        "sv26": ("0", ">=1"),  # tuần 9 không hoạt động → W1 yếu (GRADE + ACTIVITY), tuần 8 có → W2 không yếu → WATCH
    }
    risk_rows = []
    for acc, code, qt in risk_in:
        low = qt is not None and qt < D("5.0")
        w9, w8 = act.get(acc, ("", ""))
        risk_rows.append([acc, code, fx(qt), "LOW" if low else "OK", w9, w8, level.get(acc, "NONE")])
    rhead = ["account", "student_code", "qt_current", "grade_signal", "activity_week9", "activity_week8", "risk_level"]
    if write:
        for path, h, rows in ((ROOT / "seed" / "expected_final_grades.csv", head, out_rows), (HERE / "cases-tong-hop.csv", chead, case_rows), (ROOT / "seed" / "expected_risk.csv", rhead, risk_rows)):
            with open(path, "w", newline="") as fh:
                w = csv.writer(fh, lineterminator="\n")
                w.writerow(h)
                w.writerows(rows)
        print("đã ghi", len(out_rows), "dòng +", len(case_rows), "ca tổng hợp +", len(risk_rows), "dòng at-risk")
    else:
        for r in out_rows:
            print(",".join(map(str, r[:15])))
        for r in case_rows:
            print(",".join(map(str, r)))


if __name__ == "__main__":
    main("--write" in sys.argv)
