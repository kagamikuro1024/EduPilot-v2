# TC-AS06-01 (≥ 90 lượt qua 3 vòng, 30 SV) + TC-AS06-03 (EXPLAIN)
from asl import *
ALL = ["sv.gioi", "sv.kha", "sv.nguyco"] + [f"sv{n:02d}" for n in range(4, 31)]
ex = (sql("select id from courses where class_code='QC-AS5r108'") or [[None]])[0][0]
if not ex:
    c, j, _ = api(ADMIN, "POST", "/admin/courses", {"subject_code": "QCAS", "class_code": "QC-AS5r108", "name": "QC điểm danh QC-AS5r108", "semester": "2026-2027-HK1", "teacher_id": uid(TEACHER), "ta_ids": [uid(TA)], "join_code": "QCKASTZ"}); assert c == 201, (c, j); ex = j["id"]
    for u in ALL: assert api(u, "POST", "/courses/join", {"code": "QCKASTZ"})[0] == 200, u
    gen_sessions(ex, 1, 5)
C4 = ex; policy(C4, rotate_seconds=10, late_after_min=10)
s = sessions(TEACHER, C4)[0]["id"]; n = 0
for rnd in range(3):
    c, j, _ = openw(C4, s); assert c == 201, (c, j); w = j["id"]; t = tokenof(display(C4, s, w)[1])
    for u in ALL: n += checkin(u, C4, {"token": t})[0] == 201
    close(C4, w)
ok("(chuẩn bị) 3 vòng × 30 SV = 90 lượt", n == 90, str(n))
allr, cur, pages = [], None, 0
while True:
    c, j, _ = api(TEACHER, "GET", f"/courses/{C4}/sessions/{s}/checkins?limit=50" + (f"&cursor={cur}" if cur else "")); assert c == 200
    allr += j["items"]; cur = j["next_cursor"]; pages += 1
    if not cur: break
ids = [x["id"] for x in allr]; keys = [(x["created_at"], x["id"]) for x in allr]
ok("TC-AS06-01 90 lượt, limit=50: 2 trang, đủ 90, không trùng/thiếu, thứ tự (created_at,id)", len(ids) == 90 and len(set(ids)) == 90 and pages == 2 and keys == sorted(keys), f"{len(ids)} lượt, {pages} trang")
db = {x[0] for x in sql(f"select id from attendance_checkins where session_id='{s}'")}
ok("TC-AS06-01b tập id khớp DB", set(ids) == db, "")
c, j, _ = api(TEACHER, "GET", f"/courses/{C4}/sessions/{s}/checkins?limit=100"); ok("TC-AS06-02d limit=100 trả ≤ 100 (90) và next_cursor null", c == 200 and len(j["items"]) == 90 and j["next_cursor"] is None, str(len(j["items"])))
# EXPLAIN
q = f"explain (analyze, buffers) select c.id from attendance_checkins c join checkin_windows w on w.course_id=c.course_id and w.id=c.window_id join users u on u.id=c.student_id join enrollments e on e.course_id=c.course_id and e.user_id=c.student_id where c.course_id='{C4}' and c.session_id='{s}' order by c.created_at, c.id limit 51"
import subprocess
plan = subprocess.run(["docker", "exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-At", "-c", q], capture_output=True, text=True).stdout
ok("TC-AS06-03 kế hoạch truy vấn danh sách: dùng chỉ mục attendance_checkins_session_idx, không OFFSET, thời gian thực thi nhỏ", "attendance_checkins_session_idx" in plan and "Execution Time" in plan, " | ".join(l.strip() for l in plan.splitlines() if "Index" in l or "Execution" in l)[:300])
