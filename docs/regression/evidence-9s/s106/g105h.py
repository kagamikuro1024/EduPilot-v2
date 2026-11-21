import sys, os, threading; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
def run():
    off(); redis_del("ep:llm:cb:*"); fake(decision_mode="", decision_p50_ms=0, decision_p95_ms=0)
    gate("THREADS_PRECHECK", True, [C]); stop = []; codes = []
    def loop():
        i = 0
        while not stop:
            c, j, ms = post("sv.kha", C); codes.append((time.time(), c)); time.sleep(0.4)
    th = threading.Thread(target=loop); th.start(); time.sleep(8)
    c, cur, _ = api("admin", "GET", DEC); ver = [x for x in cur["gates"] if x["gate"] == "THREADS_PRECHECK"][0]["version"]
    t0 = time.time(); sql("select 1"); c2, r, _ = api("admin", "PUT", f"{DEC}/gates/THREADS_PRECHECK", {"enabled": False, "version": ver}); t1 = time.time()
    time.sleep(10); stop.append(1); th.join(); time.sleep(4)
    last = sql(f"select extract(epoch from max(created_at)) from llm_audit where task='DECIDE' and cascade_gate='THREADS_PRECHECK'")[0][0]
    n_after = sql(f"select count(*) from llm_audit where task='DECIDE' and cascade_gate='THREADS_PRECHECK' and created_at > to_timestamp({t1 + 5})")[0][0]
    lag = float(last) - t0
    bad = [c for t, c in codes if c >= 500]
    chk5("TC-D08-16", c2 == 200 and int(n_after) == 0 and not bad, f"tắt khẩn: PUT /admin/llm/decision/gates/THREADS_PRECHECK enabled=false → {c2} lúc T0; bài đăng liên tục (0,4 s/bài, {len(codes)} bài, 5xx={len(bad)}); dòng DECIDE cuối cùng cách T0 {lag:.1f} s; DECIDE sau T0+5 s = {n_after} (kỳ vọng 0); đăng bài vẫn 2xx/422 bình thường")
with dwin("QC3 105 D08-16"): run()
