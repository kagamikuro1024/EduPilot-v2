# Kiểm lại L3: 'Ôn chủ đề' biến mất ≤ 5 s ở /courses/{id}/today sau khi luyện xong chủ đề
from pa import *
U = "sv.nguyco"; sid = sql(f"select id from users where email='{U}@edupilot.local'")[0][0]
sql(f"update practice_attempts set status='SUBMITTED', submitted_at=now(), correct_count=0, score=0 where student_id='{sid}' and status='IN_PROGRESS'")  # đóng lượt dở để start sau này tạo lượt MỚI
sql(f"update practice_attempts set started_at=started_at-interval '6 days', submitted_at=submitted_at-interval '6 days' where student_id='{sid}'")
c, a, _ = start(U, {"mode": "TOPIC", "topic": "Hàm băm và chữ ký", "count": 5}); submit(U, a["id"])  # kích sự kiện, chủ đề không yếu
top = None
for _ in range(80):
    r = api(U, "GET", P + "/today")[1].get("recommended") or {}
    if r.get("kind") == "PRACTICE_WEAK_TOPIC": top = r["title"].replace("Ôn chủ đề ", ""); break
    time.sleep(1)
print("việc hiện:", top)
if top:
    c, a, _ = start(U, {"mode": "TOPIC", "topic": top, "count": 5}); submit(U, a["id"]); t0 = time.time(); gone = None
    for _ in range(15):
        r = api(U, "GET", P + "/today")[1].get("recommended") or {}
        if top not in r.get("title", ""): gone = time.time() - t0; break
        time.sleep(0.5)
    ok("RECHECK-L3 'Ôn chủ đề' biến mất ≤ 5 s ở /courses/{id}/today", gone is not None and gone <= 5, f"sau {gone}s; kế tiếp: {r.get('title')}")
else: ok("RECHECK-L3", False, "không dựng được điều kiện (không có việc 'Ôn chủ đề')")
