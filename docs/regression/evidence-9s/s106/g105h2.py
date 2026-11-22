import sys, os, re, shutil, signal; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105x import *
from gdlib import window, mk_assignment, grade_all_wait, CG0
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "s10")); import qlib
STUBS = []; PH = sys.argv[1]; NAMEK, MSK, NAMEG, MSG = "Bùi Thanh Khải", "20229002", "Vũ Hoàng Giang", "20229001"
def inj(on, courses=()):
    sql(f"UPDATE decision_gates SET enabled={'true' if on else 'false'}, version=version+1 WHERE gate='INJECTION_FLAG'")
    sql(f"INSERT INTO course_decision_gates(course_id,gate,enabled) VALUES ('{CG0}','INJECTION_FLAG',{'true' if on else 'false'}) ON CONFLICT (course_id,gate) DO UPDATE SET enabled=EXCLUDED.enabled, version=course_decision_gates.version+1")
    os.system("docker exec ep-qc-redis-1 redis-cli publish ep:decision:cfg x >/dev/null"); time.sleep(1.5)
def newpost(i, body="Mình hỏi về cách nộp bài tập tuần này ạ?", title=None): return post("sv.kha", C, (title or f"QC105 h2 {i} {time.time()}", body))
def st(s, g="THREADS_PRECHECK"): d = drows2(s, g); return [(x[1], x[2], x[3], x[4]) for x in d]
def stop(p):
    try: p.terminate(); p.wait(5)
    except Exception: pass
def run():
    for t in ("CLASSIFY", "UTILITY", "CHAT"): set_route(t, "QC fake", "fake-chat")
    stub = None
    try:
        if PH == "0":
            sql(f"UPDATE llm_models SET enabled=(model='gemini-decision-1') WHERE provider_id='{PID}'"); print("SQL (ghi DB theo TC): llm_models gemini-decision-1 enabled, decision-fake tắt; ALLOW_REAL rỗng; recreate:", envswap(None)[:40], flush=True); gate("THREADS_PRECHECK", True, [C]); redis_del("ep:llm:cb:*")
            s = now(); t0 = time.time(); stub = start_stub(); STUBS.append(stub); c = newpost(0)[0]; d = st(s)
            print("D01-20 post", c, "DECIDE", d, "yêu cầu tới máy chủ giả", len(sreqs(t0)))
        else:
            stub = start_stub(); STUBS.append(stub); time.sleep(1)
            if PH == "1":
                print("provider →", dprov("gemini-decision-1", key="qc-db-key"), flush=True); time.sleep(4); gate("THREADS_PRECHECK", True, [C]); redis_del("ep:llm:cb:*"); redis_del("ep:decision:real:*"); t0 = time.time(); out = []
                for i in range(5): s = now(); c = newpost(i)[0]; out.append((c, st(s))); time.sleep(0.3)
                print("D01-21 cap=3: 5 lời gọi", [(c, [(x[1], x[2]) for x in d]) for c, d in out], "yêu cầu tới máy chủ", len(sreqs(t0)))
                redis_del("ep:decision:real:*"); s = now(); c = newpost(9)[0]; print("D01-21 sang ngày (xoá khoá đếm ngày ep:decision:real:*):", c, st(s), flush=True)
            else:
                exec(open(os.path.join(os.path.dirname(os.path.abspath(__file__)), f"g105h2_{PH}.py"), encoding="utf8").read(), globals())
    finally:
        [stop(p) for p in STUBS]
        off(); inj(False)
        for t in ("CLASSIFY", "UTILITY", "CHAT"): set_route(t, "QC 9router", "ag/gemini-3.8-flash")
if __name__ == "__main__":
    envs = {"0": None, "1": {"LLM_DECISION_BASE_URL": "http://host.docker.internal:9770", "LLM_DECISION_ALLOW_REAL": "true", "GEMINI_API_KEY": "test-key", "LLM_DECISION_REAL_DAILY_CAP": "3"}}
    envs["2"] = {**envs["1"], "LLM_DECISION_REAL_DAILY_CAP": "100000"}; envs["3b"] = {**envs["2"], "GEMINI_API_KEY": ""}; envs["3d"] = envs["3c"] = {**envs["2"], "GEMINI_API_KEY": "", "LLM_DECISION_REAL_DAILY_CAP": "1000"}
    with window():
      try:
          key = PH if PH in ("0", "1", "3b", "3c", "3d") else "2"
          if PH != "0": ov = write_override(f"ep-qc.h{key}.yml", envs[key]); print("swap", envswap(ov)[:50], flush=True); snap("h" + key)
          run()
      finally:
          try:
              sql(f"UPDATE llm_models SET enabled=(model='decision-fake') WHERE provider_id='{PID}'") if PH == "0" else None
              dprov("decision-fake") if PH != "0" else None
          except Exception as e: print("dprov hoàn:", e)
          print("restore", envswap(None)[:50])
