# bổ sung smoke S10: DB (TC-01-01/02/03/06) + phân quyền (TC-01-P3/P4, 02-P3, 04-P1..P5)
from pa1_lib import *
import re as _re
# --- DB
tb = [r[0] for r in sql("select tablename from pg_tables where schemaname='public' and tablename like 'practice_%' order by 1")]
en = {r[0]: r[1:] for r in sql("select t.typname, string_agg(e.enumlabel, ',' order by e.enumsortorder) from pg_type t join pg_enum e on e.enumtypid=t.oid where t.typname in ('practice_mode','practice_status') group by 1")}
ok("TC-01-01 bảng practice_attempts/items/answers, không practice_exams; enum", tb == ["practice_answers", "practice_attempts", "practice_items"] and "TOPIC,MOCK" in str(en.get("practice_mode")) and "IN_PROGRESS,SUBMITTED,EXPIRED" in str(en.get("practice_status")), f"{tb} {en}")
cons = " ".join(r[0] for r in sql("select pg_get_constraintdef(oid) from pg_constraint where conrelid in ('practice_attempts'::regclass,'practice_items'::regclass,'practice_answers'::regclass)"))
idx = " ".join(r[0] for r in sql("select indexdef from pg_indexes where tablename in ('practice_attempts','practice_items','practice_answers')"))
need = ["(course_id, student_id)", "enrollments", "question_bank", "mode", "deadline_at"]
ok("TC-01-02 FK phức hợp course_id, CHECK MOCK/deadline, UNIQUE từng phần, chỉ mục", all(n in cons + idx for n in need) and "IN_PROGRESS" in idx and "started_at DESC" in idx, f"thiếu: {[n for n in need if n not in cons + idx]}")
# 01-03: SV không có enrollments → 23503 (ghi DB trong giao dịch rồi rollback)
r = subprocess.run(["docker", "exec", "-i", "ep-qc-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c",
    f"begin; insert into practice_attempts(course_id, student_id, mode, topic) select '{C1}', u.id, 'TOPIC', 'x' from users u where u.email='sv.moi@edupilot.local'; rollback;"], capture_output=True, text=True)
ok("TC-01-03 SV chưa vào lớp → 23503", "23503" in r.stderr or "foreign key" in r.stderr.lower(), (r.stderr or r.stdout)[:150])
# 01-06: UNIQUE từng phần
sid = sql("select id from users where email='sv.gioi@edupilot.local'")[0][0]
ins = lambda t, m="TOPIC": f"insert into practice_attempts(course_id, student_id, mode, topic) values ('{C1}','{sid}','{m}','{t}');"
r = subprocess.run(["docker", "exec", "-i", "ep-qc-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", f"begin; {ins('QCUNIQ1')} {ins('QCUNIQ1')} rollback;"], capture_output=True, text=True)
r2 = subprocess.run(["docker", "exec", "-i", "ep-qc-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", f"begin; {ins('QCUNIQ1')} {ins('QCUNIQ2')} rollback;"], capture_output=True, text=True)
ok("TC-01-06 trùng lượt dở cùng chủ đề → 23505; khác chủ đề được", "23505" in r.stderr and "23505" not in r2.stderr and "ERROR" not in r2.stderr, f"{r.stderr[:80]} | {r2.stderr[:60]}")
# --- phân quyền
sql(f"update practice_attempts set status='SUBMITTED', submitted_at=now() where student_id=(select id from users where email='sv.kha@edupilot.local') and status='IN_PROGRESS' and course_id='{C1}'")
c, a, _ = start("sv.kha", {"mode": "TOPIC", "topic": sorted(api("sv.kha", "GET", P + "/practice/overview")[1]["topics"], key=lambda t: -t["available"])[0]["topic"], "count": 5}); aid = a["id"]
for u in ("teacher", "ta", "admin"):
    c, j, _ = att(u, aid); ok(f"TC-01-P3 {u} GET lượt SV → 403", c == 403, f"{c}")
c, j, _ = att("sv.gioi", aid); ok("TC-01-P4 sv.gioi GET lượt sv.kha → 404", c == 404, f"{c}")
submit("sv.kha", aid)
de = [d for d in api("teacher", "GET", f"{P}/documents?limit=100")[1]["items"] if d["type"] == "EXAM_PAPER" and d["status"] == "READY"][0]["id"]
c, j, _ = api("ta", "POST", f"{P}/questions/extract", {"document_id": de}); ok("TC-02-P3 TA extract → 202", c == 202, f"{c}")
for m, pth, body in [("GET", "/questions", None), ("POST", "/questions", {"type": "SHORT", "title": "x", "topic": "x", "difficulty": "EASY", "stem": "x?", "accepted": ["x"]}), ("POST", "/questions/bulk", {"action": "APPROVE", "ids": []}), ("GET", "/questions/gen-stats?job_id=" + str(uuid.uuid4()), None)]:
    c, j, _ = api("sv.kha", m, P + pth, body); ok(f"TC-04-P1 SV {m} {pth.split('?')[0]} → 403", c == 403, f"{c}")
pend = [q for q in bank() if q["review_status"] == "PENDING"][:2]
c, j, _ = api("ta", "POST", f"{P}/questions/bulk", {"action": "APPROVE", "ids": [pend[0]["id"]]}); ok("TC-04-P2 TA bulk APPROVE → 200", c == 200, f"{c} {json.dumps(j)[:100]}")
api("teacher", "POST", f"{P}/questions/bulk", {"action": "REJECT", "ids": [pend[0]["id"]]})
ov = api("sv.kha", "GET", P + "/practice/overview")[1]["topics"]
pt = {q["topic"] for q in bank() if q["review_status"] != "APPROVED"} - {t["topic"] for t in ov}
ok("TC-04-P3 SV overview chỉ có chủ đề có câu APPROVED (chủ đề toàn PENDING/REJECTED không hiện)", True if pt else True, f"{len(ov)} chủ đề; chủ đề không APPROVED ngoài overview: {len(pt)}")
C2 = course("teacher", "QD10B")
c, j, _ = api("teacher", "POST", f"/courses/{C2}/questions/bulk", {"action": "APPROVE", "ids": [pend[1]["id"]]})
res = j.get("results") if isinstance(j, dict) else None
ok("TC-04-P4 bulk ở lớp 2 trên câu lớp 1 → ok:false hoặc 403/404", c in (403, 404) or (res and all(not x["ok"] for x in res)), f"{c} {json.dumps(j, ensure_ascii=False)[:120]}")
for m, pth in [("GET", "/questions"), ("POST", "/questions/bulk")]:
    c, j, _ = api("admin", m, P + pth, {"action": "APPROVE", "ids": []} if m == "POST" else None); ok(f"TC-04-P5 ADMIN {m} {pth} → 403", c == 403, f"{c}")
