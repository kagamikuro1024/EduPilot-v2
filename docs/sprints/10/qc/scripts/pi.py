from pa1_lib import *
from datetime import datetime, timedelta, timezone
mcq = [q for q in bank() if q["type"] == "MCQ_SINGLE" and q["review_status"] == "APPROVED" and not q["used_in_exams"]][:5]
now = datetime.now(timezone.utc)
c, e, _ = api("teacher", "POST", f"{P}/exams", {"title": "QC khoá chat", "opens_at": (now + timedelta(days=3)).isoformat(), "closes_at": (now + timedelta(days=3, hours=2)).isoformat(), "duration_minutes": 30}); eid = e["id"]
c, j, _ = api("teacher", "PUT", f"{P}/exams/{eid}/items", {"items": [{"question_id": q["id"], "points": "2"} for q in mcq], "version": e["version"]}); print("items", c)
e2 = api("teacher", "GET", f"{P}/exams/{eid}")[1]
c, j, _ = api("teacher", "POST", f"{P}/exams/{eid}/schedule", {"version": e2["version"]}); print("schedule", c, json.dumps(j, ensure_ascii=False)[:120])
sql(f"update exams set status='OPEN', opens_at=now()-interval '1 minute', closes_at=now()+interval '2 hours' where id='{eid}'")
c, at, _ = api("sv.gioi", "POST", f"{P}/exams/{eid}/attempts", {}); print("attempt", c, json.dumps(at, ensure_ascii=False)[:120])
lock = api("sv.gioi", "GET", "/me/exam-lock")[1]; print("lock", lock)
ok("TC-10-07a đang có lượt bài thi PE: /me/exam-lock khoá", lock.get("locked") is True, str(lock))
# luyện đề vẫn start được
c, a, _ = start("sv.gioi", {"mode": "TOPIC", "topic": "Mật mã công khai", "count": 5}); ok("TC-10-07b luyện đề vẫn bắt đầu được khi bài thi đang làm", c in (200, 201), str(c))
# chat nội dung môn học bị khoá
c, s, _ = api("sv.gioi", "POST", f"/courses/{C1}/chat/sessions", {}); print("chat session", c, json.dumps(s, ensure_ascii=False)[:100])
if c in (200, 201):
    c, m, _ = api("sv.gioi", "POST", f"/chat/sessions/{s['id']}/messages", {"content": "Giải thích SQL injection là gì?"}); print("chat msg", c, json.dumps(m, ensure_ascii=False)[:200])
# nộp bài thi → mở lại
if at.get("id"):
    c, j, _ = api("sv.gioi", "POST", f"{P}/exams/{eid}/attempts/{at['id']}/submit", {}); print("submit", c)
    time.sleep(1); lock2 = api("sv.gioi", "GET", "/me/exam-lock")[1]; ok("TC-10-08 nộp bài thi xong → exam-lock mở", lock2.get("locked") is False, str(lock2))
