import sys, os, subprocess; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
from gdlib import window
from g105x import drows2, envswap
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "s10")); import qlib
def pg(q): return subprocess.run(["docker", "exec", "ep-qc-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-tAc", q], capture_output=True, text=True)
def run():
    for t in ("CLASSIFY", "UTILITY", "CHAT"): set_route(t, "QC fake", "fake-chat")
    gate("THREADS_PRECHECK", True, [C]); gate("CHAT_ROUTING", True, [C])
    ren = False
    try:
        r = pg("ALTER TABLE decision_gates RENAME TO decision_gates_qcx; ALTER TABLE course_decision_gates RENAME TO course_decision_gates_qcx"); ren = True; print("đổi tên", r.stdout.strip(), r.stderr.strip()[:100])
        print("restart", envswap(None)[:50]); time.sleep(5); s = now(); out = []
        for i in range(8): out.append(post("sv.kha", C, (f"QC105 cfg {i} {time.time()}", "Mình hỏi về cách nộp bài tập tuần này ạ?"))[0])
        for i in range(3): cs, sj, _ = api("sv.kha", "POST", "/chat/sessions", {"course_id": C}); out.append(qlib.sse("sv.kha", f"/chat/sessions/{sj['id']}/messages", {"content": f"Hỏi thêm về lịch học {i}"})[0])
        d = sql(f"select count(*) from llm_audit where task='DECIDE' and created_at>='{s}'"); time.sleep(3); d = sql(f"select count(*) from llm_audit where task='DECIDE' and created_at>='{s}'")
        lg = subprocess.run(["bash", "-c", "docker logs --tail 300 ep-qc-gateway-1 2>&1 | grep -ci 'decision_gates\\|decision.*config\\|gate.*load' "], capture_output=True, text=True).stdout.strip()
        print("D02-15 bảng cấu hình không đọc được + gateway khởi động lại: mã", out, "DECIDE", d, "readyz", subprocess.run(["curl", "-sk", "https://localhost:943/api/v1/readyz"], capture_output=True, text=True).stdout, "dòng log nhắc cấu hình", lg)
        print(subprocess.run(["bash", "-c", "docker logs --tail 300 ep-qc-gateway-1 2>&1 | grep -i 'decision' | cut -c1-220 | head -4"], capture_output=True, text=True).stdout)
    finally:
        if ren: r = pg("ALTER TABLE decision_gates_qcx RENAME TO decision_gates; ALTER TABLE course_decision_gates_qcx RENAME TO course_decision_gates"); print("hoàn tên", r.stdout.strip(), r.stderr.strip()[:100])
        print("restart", envswap(None)[:50])
        sql("UPDATE course_decision_gates SET enabled=false WHERE course_id=(select id from courses where code='QD105A')") if False else off()
        for t in ("CLASSIFY", "UTILITY", "CHAT"): set_route(t, "QC 9router", "ag/gemini-3.8-flash")
with window(): run()
