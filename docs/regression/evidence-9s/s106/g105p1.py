import sys, os, threading, statistics; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105x import *
import gdlib
from concurrent.futures import ThreadPoolExecutor
MS = []
def storm(n, par=4, gap=0.0):
    codes = []
    def one(i):
        t = (f"QC105 bão {i} {time.time()}", "Mình muốn hỏi về cách trình bày phần phân tích trong bài tập số " + str(i) + " ạ, có cần nêu ví dụ cụ thể không?")
        c, j, ms = post("sv.kha", C, t); codes.append(c); MS.append(ms); time.sleep(gap)
    with ThreadPoolExecutor(par) as ex: list(ex.map(one, range(n)))
    return codes
def pct(xs, p): xs = sorted(xs); return xs[min(len(xs) - 1, int(round(p / 100 * (len(xs) - 1))))]
def phase(tag, env):
    ov = write_override(f"ep-qc.{tag}.yml", env) if env else None
    r = envswap(ov); print(tag, "readyz", r[:60], flush=True); snap(tag); return r
def run():
    for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC fake", "fake-chat")
    try:
        phase("g1a", None); gate("THREADS_PRECHECK", True, [C]); redis_del("ep:llm:cb:*"); fake(decision_mode="", decision_p50_ms=426, decision_p95_ms=682, decision_error_rate=0.0)
        s = now(); codes = storm(100, 1, 0.45); d = drows2(s, "THREADS_PRECHECK"); lat = [float(x[4]) for x in d if x[2] == "ok"]; err = sum(1 for x in d if x[2] != "ok")
        resA = (len(d), round(pct(lat, 50)), round(pct(lat, 95)), err); print("A mặc định", resA, set(codes), flush=True)
        phase("g1b", {"LLM_DECISION_FAKE_P50_MS": "100", "LLM_DECISION_FAKE_P95_MS": "150", "LLM_DECISION_RPM": "1000000"}); gate("THREADS_PRECHECK", True, [C]); redis_del("ep:llm:cb:*")
        s = now(); codes = storm(200, 3); d = drows2(s, "THREADS_PRECHECK"); lat = [float(x[4]) for x in d if x[2] == "ok"]; err = sum(1 for x in d if x[2] != "ok")
        resB = (len(d), round(pct(lat, 50)), round(pct(lat, 95)), err); print("B 100/150", resB, set(codes), flush=True)
        phase("g1c", {"LLM_DECISION_FAKE_ERROR_RATE": "0.5", "LLM_DECISION_FAKE_P50_MS": "100", "LLM_DECISION_FAKE_P95_MS": "150", "LLM_DECISION_RPM": "1000000"}); gate("THREADS_PRECHECK", True, [C]); redis_del("ep:llm:cb:*")
        s = now(); reached = 0; opens = 0; tot = 0
        while reached < 200 and tot < 900:
            storm(20, 2, 0.05); tot += 20; d = drows2(s, "THREADS_PRECHECK"); reached = sum(1 for x in d if x[2] in ("ok", "error")); co = sum(1 for x in d if x[2] == "circuit_open")
            if co > opens: opens = co; time.sleep(31)
        d = drows2(s, "THREADS_PRECHECK"); ok = sum(1 for x in d if x[2] == "ok"); er = sum(1 for x in d if x[2] == "error"); co = sum(1 for x in d if x[2] == "circuit_open"); ot = len(d) - ok - er - co
        resC = (len(d), ok, er, co, ot, round(100 * er / max(1, ok + er), 1)); print("C lỗi 0,5 (tổng, ok, error, circuit_open, khác, %lỗi tới provider)", resC, flush=True)
        phase("g1d", {"LLM_DECISION_RPM": "10"}); gate("THREADS_PRECHECK", True, [C]); redis_del("ep:llm:cb:*"); redis_del("ep:rl:*")
        MS.clear(); s = now(); t0 = time.time(); codes = storm(30, 3, 0.9); dt = time.time() - t0; d = drows2(s, "THREADS_PRECHECK"); rl = [x for x in d if x[2] == "overloaded"]
        resD = (len(d), len(rl), [round(float(x[4])) for x in rl][:3], round(dt, 1), round(pct(MS, 50)), round(pct(MS, 95))); print("D RPM10 (tổng, bị giới hạn, latency_ms, giây, post p50, post p95)", resD, set(codes), flush=True)
        json.dump({"A": resA, "B": resB, "C": resC, "D": resD}, open(W + "g1res.json", "w"))
    finally:
        off(); fake(decision_mode="", decision_error_rate=0.0, decision_p50_ms=0, decision_p95_ms=0)
        for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC 9router", "ag/gemini-3.8-flash")
if __name__ == "__main__":
    with dwin("QC3 105 g1") if False else __import__("contextlib").nullcontext():
        import shutil
        os.mkdir("/tmp/qc-global.lock") if not os.path.exists("/tmp/qc-global.lock") else sys.exit("LOCK")
        open("/tmp/qc-global.lock/owner", "w").write("QC3 105 env g1")
        try: run()
        finally: shutil.rmtree("/tmp/qc-global.lock", ignore_errors=True)
