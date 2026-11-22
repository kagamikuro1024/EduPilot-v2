import sys, os, re, subprocess; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
from gdlib import window
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "s10")); import qlib
def pg(q): return subprocess.run(["docker", "exec", "ep-qc-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True).stdout.strip()
def run():
    for t in ("CLASSIFY", "UTILITY", "CHAT"): set_route(t, "QC fake", "fake-chat")
    old = pg("show log_statement")
    try:
        for g in ("THREADS_PRECHECK", "CHAT_ROUTING"): gate(g, True, [C])
        redis_del("ep:llm:cb:*"); time.sleep(2)
        print("ALTER", pg("ALTER SYSTEM SET log_statement='all'"), pg("select pg_reload_conf()"), "→", pg("show log_statement")); time.sleep(2)
        import datetime
        seen = {}
        def collect():
            o = subprocess.run(["docker", "logs", "--tail", "3000", "ep-qc-postgres-1"], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, text=True).stdout or ""
            for l in o.splitlines():
                if l[:2] == "20": seen[l] = 1
        t0 = time.time(); s = now(); n = 0; codes = set(); ts = datetime.datetime.utcfromtimestamp(t0).strftime("%Y-%m-%d %H:%M:%S")
        for i in range(25):
            codes.add(post("sv.kha", C, (f"QC105 log {i} {time.time()}", "Mình hỏi về cách nộp bài tập tuần này ạ?"))[0]); n += 1
            if i % 3 == 2: collect()
        for i in range(25):
            cs, sj, _ = api("sv.kha", "POST", "/chat/sessions", {"course_id": C}); codes.add(qlib.sse("sv.kha", f"/chat/sessions/{sj['id']}/messages", {"content": f"Cho mình hỏi thêm về lịch học tuần sau nhé {i}"})[0]); n += 1
            if i % 3 == 2: collect()
        t1 = time.time(); time.sleep(3); collect()
        lines = [l for l in seen if l[:19] >= ts]; gl = [l for l in lines if re.search(r"decision_gates|course_decision_gates", l)]
        stm = sum(1 for l in lines if "statement:" in l or "execute" in l); first = min(l[:23] for l in lines)
        print("yêu cầu", n, "mã", codes, "| DECIDE trong khung", sql(f"select cascade_gate,count(*) from llm_audit where task='DECIDE' and created_at>='{s}' group by 1"), "| statement trong khung", stm, "(dòng log đầu", first, ", bắt đầu", ts, ") | chạm bảng cổng:", len(gl), "trong", round(t1 - t0, 1), "s"); print("\n".join(l[:200] for l in gl[:6]))
    finally:
        pg("ALTER SYSTEM RESET log_statement"); pg("select pg_reload_conf()"); print("hoàn nguyên log_statement →", pg("show log_statement"), "(gốc", old + ")")
        off()
        for t in ("CLASSIFY", "UTILITY", "CHAT"): set_route(t, "QC 9router", "ag/gemini-3.8-flash")
with window(): run()
