# smoke r10.8: TC-AS06-06/07a/08 với chờ hết TTL cache «Hôm nay» (~29 s) vì fixture SQL không làm mất cache
from asl import *
S = st(); C = S["cid"]
def today(user, wait=31):
    time.sleep(wait)
    j = api(user, "GET", "/me/today")[1]; return [x for x in (j.get("actions", []) + j.get("attention", [])) if x.get("kind") == "CHECKIN_REVIEW" and (x.get("course") or {}).get("id") == C]
for i in [x[0] for x in sql(f"select id from attendance_checkins where course_id='{C}' and cardinality(flags)>0 and flags_cleared_at is null")]: api(TEACHER, "POST", f"/courses/{C}/checkins/{i}/clear-flags", None)
ok("TC-AS06-07a hết cờ → không có mục CHECKIN_REVIEW của lớp QC-AS (chờ TTL)", not today(TEACHER), "")
s, w, d = neww(C)
print([checkin(u, C, {"token": tokenof(d)})[0] for u in ("sv.gioi", "sv.kha", "sv.nguyco")])
close(C, w); it = today(TEACHER)
ok("TC-AS06-06 (chờ TTL) TEACHER /me/today: CHECKIN_REVIEW, «3 lượt điểm danh cần xem», href ?filter=review", len(it) == 1 and "3 lượt" in it[0]["title"] and "filter=review" in it[0]["href"], json.dumps(it, ensure_ascii=False)[:300])
sql(f"update checkin_windows set closed_at = now() - interval '8 days' where id='{w}'")
ok("TC-AS06-08 (fixture SQL, chờ TTL) cửa sổ đóng > 7 ngày → mục biến mất", not today(TEACHER), "")
for i in [x[0] for x in sql(f"select id from attendance_checkins where window_id='{w}'")]: api(TEACHER, "POST", f"/courses/{C}/checkins/{i}/clear-flags", None)
