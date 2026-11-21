import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
import gdlib
def one(mode, prob):
    redis_del("ep:llm:cb:*"); fake(decision_mode=mode, decision_prob=prob); time.sleep(0.5)
    s = now(); c, j, ms = post("sv.kha", C); time.sleep(3); return c, rows(s)
def run():
    gate("THREADS_PRECHECK", True, [C]); out = {}
    out["FAST"] = one("", 0.01); out["FALLBACK(error)"] = one("error", None); out["FALLBACK(lưỡng lự)"] = one("", 0.25)
    for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC lỗi", gdlib._prov()["QC lỗi"]["models"][0]["model"])
    out["FAIL_CLOSED"] = one("error", None)
    for k, (c, d) in out.items(): print(k, c, [(x[0], x[1], x[2]) for x in d])
    exp = {"FAST": "FAST", "FALLBACK(error)": "FALLBACK", "FALLBACK(lưỡng lự)": "FALLBACK", "FAIL_CLOSED": "FAIL_CLOSED"}
    good = all(any(x[1] == exp[k] for x in d) for k, (c, d) in out.items()) and out["FAIL_CLOSED"][0] == 422 and out["FAST"][0] in (200, 201)
    chk5("TC-D02-03", good, "4 tình huống ở cổng THREADS (QD105A): " + "; ".join(f"{k}: HTTP {c} path={sorted(set(x[1] for x in d))} status={sorted(set(x[2] for x in d))}" for k, (c, d) in out.items()) + " (FAIL_CLOSED dựng bằng decision error + CLASSIFY/UTILITY → provider «QC lỗi»; Threads chặn 422)")
with dwin("QC3 105 D02-03"): 
    try: run()
    finally: fake(decision_mode="", decision_prob=None) if False else fake(decision_mode="")
