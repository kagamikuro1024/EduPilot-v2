import sys, os, re, time; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
import gdlib
P = "/tmp/qc9s-pw.lock"
C = cid("QD105A"); C2 = cid("QD105B"); DEC = "/admin/llm/decision"
def fake(**k): return api("admin", "POST", "/_test/llm/fake", k)
def gate(g, system, courses):
    sql(f"UPDATE decision_gates SET enabled={'true' if system else 'false'}, version=version+1 WHERE gate='{g}'")
    for cc in (C, C2):
        sql(f"INSERT INTO course_decision_gates(course_id,gate,enabled) VALUES ('{cc}','{g}',{'true' if cc in courses else 'false'}) ON CONFLICT (course_id,gate) DO UPDATE SET enabled=EXCLUDED.enabled, version=course_decision_gates.version+1")
    os.system("docker exec ep-qc-redis-1 redis-cli publish ep:decision:cfg x >/dev/null"); time.sleep(1.5)
def off(): sql("UPDATE decision_gates SET enabled=false"); sql("UPDATE course_decision_gates SET enabled=false"); os.system("docker exec ep-qc-redis-1 redis-cli publish ep:decision:cfg x >/dev/null"); time.sleep(1.2)
def now(): return sql("select now()")[0][0]
def rows(since, extra=""): return sql(f"select cascade_gate,cascade_path,status,coalesce(error_kind,''),coalesce(latency_ms,0) from llm_audit where task='DECIDE' and created_at>='{since}' {extra} order by created_at")
SUBTLE = ("Có bạn hoàn cảnh khó khăn bị trừ điểm", "Mình nghe nói có một bạn trong lớp hoàn cảnh khó khăn nên bị trừ điểm chuyên cần, không biết có đúng không và nên xử lý thế nào ạ.")
def post(user, course, t=None):
    t = t or ("QC105 " + str(time.time())[-6:] + " " + SUBTLE[0], SUBTLE[1])
    return api(user, "POST", f"/courses/{course}/threads", {"title": t[0], "body": t[1]})

import shutil, contextlib
@contextlib.contextmanager
def dwin(owner="QC3 105 decision"):
    t0 = time.time()
    while True:
        try: os.mkdir("/tmp/qc-global.lock"); break
        except FileExistsError:
            if time.time() - t0 > 600: raise SystemExit("LOCKED")
            time.sleep(5)
    open("/tmp/qc-global.lock/owner", "w").write(owner)
    try:
        for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC fake", "fake-chat")
        yield
    finally:
        for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC 9router", "ag/gemini-3.8-flash")
        off(); fake(decision_mode="", decision_error_rate=0.0, decision_prob=None) if False else fake(decision_mode="", decision_error_rate=0.0)
        redis_del("ep:llm:cb:*"); shutil.rmtree("/tmp/qc-global.lock", ignore_errors=True)
def results(tc_list, f="/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/results_105.jsonl"): pass
def chk5(tc, cond, note):
    st = "P" if cond else "F"; print(st, tc, note[:240])
    open("/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/results_105.jsonl", "a", encoding="utf8").write(json.dumps({"tc": tc, "kq": st, "note": note}, ensure_ascii=False) + "\n")
def note5(tc, kq, note):
    print(kq, tc, note[:200]); open("/Users/kuro/Documents/TA_Agent_qc9s/docs/regression/evidence-9s/s106/results_105.jsonl", "a", encoding="utf8").write(json.dumps({"tc": tc, "kq": kq, "note": note}, ensure_ascii=False) + "\n")
