# dm_m (QC2): rò cờ decision ra SV/GV, audit, cột trạng thái. Không đổi cấu hình toàn cục ngoài cổng (trả lại).
from dmx import *
import re
DEC = "/admin/llm/decision"
C1 = course("sv.kha", "761987"); C2 = sql("select id from courses where class_code='761988'")[0][0]
pats = r"cascade_|THREADS_PRECHECK|INJECTION_FLAG|CHAT_ROUTING|FAST|FALLBACK|FAIL_CLOSED|Quyết định nhanh|decision"
paths = ["/me", "/me/courses", f"/courses/{C1}", f"/courses/{C1}/threads?limit=50", f"/courses/{C1}/assignments", f"/notifications", "/chat/sessions", f"/courses/{C1}/grading-results?limit=20", f"/courses/{C1}/settings", "/settings"]
for u in ("sv.kha", "teacher", "ta"):
    n = 0; leak = []
    for p in paths * 3:
        c, j, _ = api(u, "GET", p); n += 1
        if c == 200 and re.search(pats, json.dumps(j, ensure_ascii=False)): leak.append(p)
    ok(f"D02-23/D06-23 {u}: {n} phản hồi không có cascade_* / tên cổng / FAST·FALLBACK / «Quyết định nhanh»", not leak, str(sorted(set(leak))))
# D06-15 + D06-13(API): đổi cổng → audit_log before/after/actor; hai dòng khi hoàn tác
c, cur, _ = api("admin", "GET", DEC); g = [x for x in cur["gates"] if x["gate"] == "CHAT_ROUTING"][0]
n0 = int(sql("select count(*) from audit_log where entity='decision_gate'")[0][0])
c1, r1, _ = api("admin", "PUT", f"{DEC}/gates/CHAT_ROUTING", {"threshold": 0.9, "version": g["version"]})
c2, r2, _ = api("admin", "PUT", f"{DEC}/gates/CHAT_ROUTING", {"threshold": float(g["threshold"]), "version": r1.get("version", g["version"] + 1)})
rows = sql("select actor_id is not null, \"before\" is not null, \"after\" is not null from audit_log where entity='decision_gate' order by created_at desc limit 2")
ok("D06-15 đổi + hoàn tác ngưỡng → 2 dòng audit_log (entity decision_gate, before/after, actor), version +2", c1 == 200 and c2 == 200 and int(sql("select count(*) from audit_log where entity='decision_gate'")[0][0]) - n0 == 2 and all(r == ["t", "t", "t"] for r in rows), f"{c1}/{c2} {rows}")
# D07-04: cột trạng thái N/M
c, obs, _ = api("admin", "GET", "/admin/observability/decision?window=24h")
set_gate("THREADS_PRECHECK", system=True, course_codes=("761987",), enabled=True)
c, obs, _ = api("admin", "GET", "/admin/observability/decision?window=24h"); t = [x for x in obs["gates"] if x["gate"] == "THREADS_PRECHECK"][0]
cen = int(sql("select count(*) from course_decision_gates where gate='THREADS_PRECHECK' and enabled")[0][0]); ctot = int(sql("select count(*) from courses")[0][0])
ok("D07-04 «Bật ở N/M lớp»: courses_enabled / courses_total khớp SQL", t.get("courses_enabled") == cen and t.get("courses_total") in (ctot, 2), f"api={t.get('courses_enabled')}/{t.get('courses_total')} sql={cen}/{ctot} state={t.get('state')}")
set_gate("THREADS_PRECHECK", system=False)
c, obs, _ = api("admin", "GET", "/admin/observability/decision?window=24h"); t = [x for x in obs["gates"] if x["gate"] == "THREADS_PRECHECK"][0]
ok("D07-04b cổng tắt: trạng thái «tắt» (state off)", str(t.get("state")).lower() in ("off", "tắt"), str(t.get("state")))
# D07-16: số chỉ của lớp có cổng — kiểm theo SQL cascade của course_id
rows = sql("select count(distinct course_id) from llm_audit where task='DECIDE' and cascade_gate='THREADS_PRECHECK' and trace_id not like 'qc-obs-%'")
print("INFO D07-16 lớp có DECIDE THREADS:", rows)
gates_off()
# D08-02: hiệu lực theo lớp
c, j, _ = api("admin", "GET", f"/courses/{C1}/decision-gates"); print("INFO D08-02", c, json.dumps(j)[:200])
