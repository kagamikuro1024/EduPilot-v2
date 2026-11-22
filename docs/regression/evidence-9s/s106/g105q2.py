import sys, os, subprocess; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
from gdlib import window
from g105x import drows2
def run():
    for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC fake", "fake-chat")
    try:
        gate("THREADS_PRECHECK", True, [C]); redis_del("ep:llm:cb:*"); time.sleep(2)
        lk = subprocess.Popen(["docker", "exec", "ep-qc-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-c", "BEGIN; LOCK TABLE decision_gates, course_decision_gates IN ACCESS EXCLUSIVE MODE; select pg_sleep(25); COMMIT;"], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL); time.sleep(2)
        s = now(); res = []
        for i in range(10): t = time.time(); c = post("sv.kha", C, (f"QC105 khoá {i} {time.time()}", "Mình hỏi về cách nộp bài tập tuần này ạ?"))[0]; res.append((c, round((time.time() - t) * 1000)))
        d = drows2(s, "THREADS_PRECHECK"); lk.wait()
        print("D02-14b khoá bảng cấu hình 25 s: 10 đăng bài", res, "DECIDE", len(d), [x[2] for x in d][:3])
    finally:
        off()
        for t in ("CLASSIFY", "UTILITY"): set_route(t, "QC 9router", "ag/gemini-3.8-flash")
with window(): run()
