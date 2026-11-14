from pa1_lib import *
from datetime import datetime, timedelta, timezone
# TC-01-13: SHORT/ESSAY đã APPROVED không thêm được vào bài thi
sh = [q for q in bank() if q["type"] == "SHORT" and q["review_status"] == "APPROVED"][0]
es = [q for q in bank() if q["type"] == "ESSAY" and q["review_status"] == "APPROVED"][0]
now = datetime.now(timezone.utc)
c, e, _ = api("teacher", "POST", f"{P}/exams", {"title": "QC bài thi nháp", "opens_at": (now + timedelta(days=3)).isoformat(), "closes_at": (now + timedelta(days=3, hours=2)).isoformat(), "duration_minutes": 30})
print("create exam", c, json.dumps(e, ensure_ascii=False)[:150])
if c == 201:
    for nm, q in (("SHORT", sh), ("ESSAY", es)):
        c, j, _ = api("teacher", "PUT", f"{P}/exams/{e['id']}/items", {"items": [{"question_id": q["id"], "points": "1"}], "version": e["version"]})
        ok(f"TC-01-13 {nm} vào bài thi → 422 TYPE_NOT_SUPPORTED_IN_EXAM", c == 422 and "TYPE_NOT_SUPPORTED_IN_EXAM" in json.dumps(j), f"{c} {json.dumps(j, ensure_ascii=False)[:160]}")
