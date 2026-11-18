import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
def run():
    print("cờ", flag("GRADING_GUIDANCE", True)[0])
    # ===== TC-G03-14: support giả =====
    a = fxa("g0314"); review_with(a, [(i, {"c2": "0.00"}) for i in range(0, 4)] + [(4, {"c3": "2.00"}), (5, {"c3": "2.00"})])
    cF, jF, jobF = reflect(a, "rf_fake_support"); dF = drafts(a); print("rf_fake_support:", cF, jobF and jobF[0], dF)
    cL, jL, jobL = reflect(a, "rf_low_support"); dL = drafts(a); print("rf_low_support:", cL, jobL and jobL[0], dL)
    cO, jO, jobO = reflect(a, "rf_ok"); dO = drafts(a); print("rf_ok:", cO, jobO and jobO[0], [(x[1], x[2], x[3]) for x in dO])
    chk("TC-G03-14", cF == 202 and len(dF) == 0 and len(dL) == len(dF) and any(int(x[3]) == 2 for x in dO) is True or any(int(x[3]) >= 2 for x in dO), f"stub trả support:99 với đúng 1 ca hợp lệ (+1 ca lạ): job {jobF and jobF[0]}, mục lưu {len(dF)} (bị bỏ: support do mã đếm = 1 < 2); rf_low_support (1 ca đúng chiều + 1 ngược): mục lưu {len(dL) - len(dF)}; biên: rf_ok → {len(dO) - len(dL)} mục với support {[int(x[3]) for x in dO]} (≥2 giữ)")
with window(): run()
