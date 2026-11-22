import sys, os, shutil; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
import g105p1 as PP
from g105x import *
os.mkdir("/tmp/qc-global.lock"); open("/tmp/qc-global.lock/owner", "w").write("QC3 105 env g1d")
try:
    for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC fake", "fake-chat")
    PP.phase("g1d", {"LLM_DECISION_RPM": "10"}); gate("THREADS_PRECHECK", True, [C]); redis_del("ep:llm:cb:*"); redis_del("ep:rl:*")
    PP.MS.clear(); s = now(); t0 = time.time(); codes = PP.storm(30, 3, 0.9); dt = time.time() - t0; d = drows2(s, "THREADS_PRECHECK"); rl = [x for x in d if x[2] == "overloaded"]
    print("D RPM10 (tổng, bị giới hạn, latency_ms, giây, post p50, post p95)", (len(d), len(rl), [round(float(x[4])) for x in rl][:5], round(dt, 1), round(PP.pct(PP.MS, 50)), round(PP.pct(PP.MS, 95))), set(codes), [(x[1], x[2]) for x in d][:3], flush=True)
finally:
    off(); fake(decision_mode="", decision_error_rate=0.0, decision_p50_ms=0, decision_p95_ms=0)
    for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC 9router", "ag/gemini-3.8-flash")
    print("restore", envswap(None)[:50]); shutil.rmtree("/tmp/qc-global.lock", ignore_errors=True)
