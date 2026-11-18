# TC-10-07c / 08 / 08b trên tuyến chat hiện tại (POST /chat/sessions {course_id}); bản của pi.py dùng tuyến cũ /courses/{id}/chat/sessions
from pa1_lib import *
from datetime import datetime, timedelta, timezone
import uuid as _u
mcq = [q for q in bank() if q["type"] == "MCQ_SINGLE" and q["review_status"] == "APPROVED" and not q["used_in_exams"]][:5]
now = datetime.now(timezone.utc)
c, e, _ = api("teacher", "POST", f"{P}/exams", {"title": "QC khoá chat r10", "opens_at": (now + timedelta(days=3)).isoformat(), "closes_at": (now + timedelta(days=3, hours=2)).isoformat(), "duration_minutes": 30}); eid = e["id"]
api("teacher", "PUT", f"{P}/exams/{eid}/items", {"items": [{"question_id": q["id"], "points": "2"} for q in mcq], "version": e["version"]})
e2 = api("teacher", "GET", f"{P}/exams/{eid}")[1]
api("teacher", "POST", f"{P}/exams/{eid}/schedule", {"version": e2["version"]})
sql(f"update exams set status='OPEN', opens_at=now()-interval '1 minute', closes_at=now()+interval '2 hours' where id='{eid}'")
TAB = str(_u.uuid4()); c, at, _ = api("sv.gioi", "POST", f"{P}/exams/{eid}/attempts", {}, headers={"X-Exam-Tab": TAB}); aid = (at.get("attempt") or at)["id"]
c, s, _ = api("sv.gioi", "POST", "/chat/sessions", {"course_id": C1}); print("chat session", c, json.dumps(s, ensure_ascii=False)[:120])
c, m, _ = api("sv.gioi", "POST", f"/chat/sessions/{s['id']}/messages", {"content": "Giải thích SQL injection là gì?"})
ok("TC-10-07c chat từ chối nội dung môn học trong giờ thi", c == 409 and "EXAM_IN_PROGRESS" in json.dumps(m), f"{c} {json.dumps(m, ensure_ascii=False)[:120]}")
c, j, _ = api("sv.gioi", "POST", f"{P}/exams/{eid}/attempts/{aid}/submit", {}, headers={"X-Exam-Tab": TAB})
time.sleep(1); lock2 = api("sv.gioi", "GET", "/me/exam-lock")[1]
ok("TC-10-08 nộp bài thi xong → exam-lock mở", c == 200 and lock2.get("locked") is False, f"submit={c} {lock2}")
c, m, _ = api("sv.gioi", "POST", f"/chat/sessions/{s['id']}/messages", {"content": "Giải thích SQL injection là gì?"})
ok("TC-10-08b sau thi chat trả lời lại (nhận tin nhắn)", c in (200, 201, 202), f"{c} {json.dumps(m, ensure_ascii=False)[:100]}")
