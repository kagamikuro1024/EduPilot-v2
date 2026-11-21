import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from g105lib import *
import gdlib
from gdlib import window, mk_assignment, grade_all_wait, CG0
sys.path.insert(0, os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "s10")); import qlib
def pset(en):
    pid = api("admin", "GET", DEC)[1]["provider"]["id"]; cur = [x for x in api("admin", "GET", "/admin/llm/providers")[1]["items"] if x["id"] == pid][0]
    return api("admin", "PUT", f"/admin/llm/providers/{pid}", {**{k: cur.get(k) for k in ("name", "type", "base_url", "rpm_limit", "tpm_limit") if cur.get(k) is not None}, "enabled": en, "version": cur["version"]})[0]
def run():
    for t in ("CHAT", "CLASSIFY", "UTILITY"): set_route(t, "QC fake", "fake-chat")
    try:
        for g in ("THREADS_PRECHECK", "CHAT_ROUTING"): gate(g, True, [C])
        sql("UPDATE decision_gates SET enabled=true, version=version+1 WHERE gate='INJECTION_FLAG'"); sql(f"INSERT INTO course_decision_gates(course_id,gate,enabled) VALUES ('{CG0}','INJECTION_FLAG',true) ON CONFLICT (course_id,gate) DO UPDATE SET enabled=true"); os.system("docker exec ep-qc-redis-1 redis-cli publish ep:decision:cfg x >/dev/null"); time.sleep(1.5)
        redis_del("ep:llm:cb:*"); fake(decision_mode="")
        print("provider tắt", pset(False)); time.sleep(3); s = now()
        tc = post("sv.kha", C)[0]
        cs, sj, _ = api("sv.kha", "POST", "/chat/sessions", {"course_id": C}); st, a, b, body = qlib.sse("sv.kha", f"/chat/sessions/{sj['id']}/messages", {"content": "Nói giúp mình về mấy cái bảo mật mạng căn bản trong môn này nhé"})
        ag = mk_assignment("g105i", 1); gc, gj, nn = grade_all_wait(ag); time.sleep(4)
        d = sql(f"select cascade_gate, count(*) from llm_audit where task='DECIDE' and created_at>='{s}' group by 1")
        done = sql(f"select count(*) from grading_results r join submissions s on s.id=r.submission_id and s.course_id=r.course_id where s.assignment_id='{ag}' and r.run_state='DONE'")[0][0]
        chk5("TC-D01-25", d == [] and tc in (200, 201, 422) and st == 200 and int(done) == 1, f"provider «Quyết định» tắt, cả 3 cổng bật (THREADS/CHAT lớp QD105A, INJECTION lớp QD9G): đăng bài HTTP {tc}, chat SSE {st} ({b} ms), chấm 1 bài DONE={done}; llm_audit DECIDE trong lượt = {d} (kỳ vọng rỗng)")
    finally:
        print("provider bật", pset(True)); off(); sql(f"UPDATE course_decision_gates SET enabled=false WHERE course_id='{CG0}'")
        for t in ("CHAT", "CLASSIFY", "UTILITY"): set_route(t, "QC 9router", "ag/gemini-3.8-flash")
with window(): run()
