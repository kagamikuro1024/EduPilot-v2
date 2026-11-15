# US-AS-06 AC3 — mục «Hôm nay» CHECKIN_REVIEW
from asl import *
S = st(); C = S["cid"]
def today(user):
    time.sleep(1.6)
    j = api(user, "GET", "/me/today")[1]; return [x for x in (j.get("actions", []) + j.get("attention", [])) if x.get("kind") == "CHECKIN_REVIEW" and (x.get("course") or {}).get("id") == C]
# xoá cờ tồn đọng trước
for i in [x[0] for x in sql(f"select id from attendance_checkins where course_id='{C}' and cardinality(flags)>0 and flags_cleared_at is null")]: api(TEACHER, "POST", f"/courses/{C}/checkins/{i}/clear-flags", None)
ok("TC-AS06-07a hết cờ → không có mục CHECKIN_REVIEW của lớp QC-AS", not today(TEACHER), "")
s, w, d = neww(C)
for u in ("sv.gioi", "sv.kha", "sv.nguyco"): checkin(u, C, {"token": tokenof(d)})
close(C, w); it = today(TEACHER)
ok("TC-AS06-06 TEACHER /me/today: CHECKIN_REVIEW, «3 lượt điểm danh cần xem», href ?filter=review", len(it) == 1 and "3 lượt" in it[0]["title"] and "filter=review" in it[0]["href"] and f"session={s}" in it[0]["href"], json.dumps(it, ensure_ascii=False)[:300])
ok("TC-AS06-06b TA cũng thấy", len(today(TA)) == 1, "")
ok("TC-AS06-09 SV không thấy CHECKIN_REVIEW", not today("sv.gioi"), "")
ids = [x[0] for x in sql(f"select id from attendance_checkins where window_id='{w}'")]
api(TEACHER, "POST", f"/courses/{C}/checkins/{ids[0]}/clear-flags", None); it = today(TEACHER)
ok("TC-AS06-06c bỏ cờ 1 lượt → còn «2 lượt»", len(it) == 1 and "2 lượt" in it[0]["title"], it[0]["title"] if it else "")
sql(f"update checkin_windows set closed_at = now() - interval '8 days' where id='{w}'")
ok("TC-AS06-08 cửa sổ đóng > 7 ngày → mục biến mất", not today(TEACHER), "")
sql(f"update checkin_windows set closed_at = now() - interval '2 days' where id='{w}'")
ok("TC-AS06-08b (đối chứng) đóng 2 ngày trước → mục còn", len(today(TEACHER)) == 1, "")
for i in ids[1:]: api(TEACHER, "POST", f"/courses/{C}/checkins/{i}/clear-flags", None)
ok("TC-AS06-07 bỏ cờ hết → mục biến mất", not today(TEACHER), "")
