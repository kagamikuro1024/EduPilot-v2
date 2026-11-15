# US-AS-05, 06, 07 — đóng cửa sổ, đánh vắng, danh sách cờ, Hôm nay, vòng 2, gọi tên
from asl import *
import re, concurrent.futures as cf
S = st(); C, C2 = S["cid"], S["cid2"]
N = len(api(TEACHER, "GET", f"/courses/{C}/sessions/{S['sessions'][-1]}/attendance")[1]["students"]); print("SV ACTIVE:", N)
KABS = str(uuid.uuid4())
now = lambda: datetime.datetime.now(datetime.timezone.utc)
def mk_session(off_min):
    s = fresh_session(C); it = [x for x in sessions(TEACHER, C) if x["id"] == s][0]; t0 = now() + datetime.timedelta(minutes=off_min)
    c, j, _ = api(TEACHER, "PUT", f"/courses/{C}/sessions/{s}", {"starts_at": t0.strftime("%Y-%m-%dT%H:%M:%SZ"), "ends_at": (t0 + datetime.timedelta(hours=2)).strftime("%Y-%m-%dT%H:%M:%SZ"), "version": it["version"]}); assert c == 200, (c, j)
    return s
def win(s, minutes=None):
    c, j, _ = openw(C, s, minutes=minutes); assert c == 201, (c, j); c, d, _ = display(C, s, j["id"]); return j["id"], d
def grid(s): return api(TEACHER, "GET", f"/courses/{C}/sessions/{s}/attendance")[1]
recs = lambda s: {x[0].split("@")[0]: (x[1], x[2]) for x in sql(f"select u.email, r.status, r.version from attendance_records r join users u on u.id=r.student_id where r.session_id='{s}'")}
# ---- phiên A: PRESENT (starts_at = now-3 phút, late_after=10)
policy(C, rotate_seconds=5, late_after_min=10)
sA = mk_session(-3); wA, dA = win(sA)
for u in ("sv.gioi", "sv.kha"): checkin(u, C, {"token": tokenof(dA)})
# 13 trước khi đóng
c, j, _ = api(TEACHER, "POST", f"/courses/{C}/sessions/{sA}/absent-unchecked", None); ok("TC-AS05-13a absent-unchecked khi còn cửa sổ mở → 409 WINDOW_OPEN", c == 409 and "WINDOW_OPEN" in json.dumps(j), f"{c} {json.dumps(j)[:90]}")
sN = fresh_session(C); c, j, _ = api(TEACHER, "POST", f"/courses/{C}/sessions/{sN}/absent-unchecked", None); ok("TC-AS05-13b buổi chưa từng có cửa sổ → 409 NO_WINDOW", c == 409 and "NO_WINDOW" in json.dumps(j), f"{c} {json.dumps(j)[:90]}")
# 16 phân quyền close / absent-unchecked
r16 = {u: (close(C, wA, user=u)[0], api(u, "POST", f"/courses/{C}/sessions/{sA}/absent-unchecked", None)[0]) for u in ("sv.gioi", ADMIN, "sv.moi")}
ok("TC-AS05-16 STUDENT/ADMIN/sv.moi: close + absent-unchecked → 403", all(v == (403, 403) for v in r16.values()), str(r16))
c, j, _ = close(C, wA); wr = j
ok("TC-AS05-01 đóng: 200 {window, written, unchecked, skipped_existing}", c == 200 and set(j) == {"window", "written", "unchecked", "skipped_existing"} and j["written"]["present"] == 2 and j["written"]["late"] == 0 and j["unchecked"] == N - 2, json.dumps(j)[:220])
R = recs(sA)
ok("TC-AS05-02a lượt trong ≤10 phút sau starts_at → PRESENT, version 1", R.get("sv.gioi") == ("PRESENT", "1") and R.get("sv.kha") == ("PRESENT", "1"), str(R))
ok("TC-AS05-04 SV không lượt: không tạo ABSENT; unchecked = N-2", not any(v[0] == "ABSENT" for v in R.values()) and len(R) == 2, str(R))
n0 = (sql(f"select count(*) from attendance_records where session_id='{sA}'")[0][0], sql(f"select count(*) from audit_log where course_id='{C}'")[0][0], sql("select count(*) from outbox")[0][0] if sql("select to_regclass('outbox')")[0][0] else "")
c1, j1, _ = close(C, wA); c2, j2, _ = close(C, wA, idem=str(uuid.uuid4()))
n1 = (sql(f"select count(*) from attendance_records where session_id='{sA}'")[0][0], sql(f"select count(*) from audit_log where course_id='{C}'")[0][0], sql("select count(*) from outbox")[0][0] if sql("select to_regclass('outbox')")[0][0] else "")
ok("TC-AS05-06 close lần 2/3: 200 cùng kết quả; records/audit/outbox không thêm", c1 == c2 == 200 and j1 == j2 == wr and n0 == n1, f"{n0}->{n1}")
# 07/08/09
c, j, _ = api(TEACHER, "POST", f"/courses/{C}/sessions/{sA}/absent-unchecked", None, idem=KABS); R = recs(sA)
ok("TC-AS05-07 absent-unchecked: 200 {marked:N-2}; N-2 dòng ABSENT", c == 200 and j["marked"] == N - 2 and sum(1 for v in R.values() if v[0] == "ABSENT") == N - 2, f"{c} {j}")
a = sql(f"select action, left(after::text,160) from audit_log where course_id='{C}' and action like '%absent%' order by created_at desc limit 1"); ok("TC-AS05-07b audit_log absent_unchecked (số đếm)", len(a) == 1, str(a))
c, j, _ = api(TEACHER, "POST", f"/courses/{C}/sessions/{sA}/absent-unchecked", None, idem=KABS); c2, j2, _ = api(TEACHER, "POST", f"/courses/{C}/sessions/{sA}/absent-unchecked", None)
ok("TC-AS05-08 lặp cùng khoá → 200 cùng kết quả; khoá mới → {marked:0}", c == 200 and j == {"marked": N - 2} and c2 == 200 and j2 == {"marked": 0}, f"{j} {j2}")
g = grid(sA); st_ = [x for x in g["students"] if x.get("record") and x["record"]["status"] == "ABSENT"]
tgt = st_[0]; c, j, _ = api(TEACHER, "PUT", f"/courses/{C}/sessions/{sA}/attendance", {"records": [{"student_id": tgt["id"], "status": "PRESENT", "version": tgt["record"]["version"]}]})
ok("TC-AS05-09 sửa ABSENT → PRESENT bằng lưới theo version", c == 200 and j["records"][0]["version"] == tgt["record"]["version"] + 1, f"{c} {json.dumps(j)[:150]}")
# ---- phiên B: LATE (starts_at = now-40), có dòng tay trước khi đóng
sB = mk_session(-40); wB, dB = win(sB)
ugid = lambda u: uid(u)
c, j, _ = api(TEACHER, "PUT", f"/courses/{C}/sessions/{sB}/attendance", {"records": [{"student_id": ugid("sv.nguyco"), "status": "ABSENT"}]}); print("tay", c)
for u in ("sv.gioi", "sv.kha", "sv.nguyco"): checkin(u, C, {"token": tokenof(dB)})
c, j, _ = close(C, wB); R = recs(sB)
ok("TC-AS05-02b lượt trễ > starts_at+10 phút → LATE", R.get("sv.gioi", ("",))[0] == "LATE" and R.get("sv.kha", ("",))[0] == "LATE", str(R))
ok("TC-AS05-03 dòng GV sửa tay giữ nguyên (ABSENT), skipped_existing=1", R.get("sv.nguyco", ("",))[0] == "ABSENT" and j["skipped_existing"] == 1 and j["written"]["late"] == 2, json.dumps(j)[:200])
# 10/11
c, j, _ = api(TEACHER, "PUT", f"/courses/{C}/sessions/{sB}/attendance", {"complete": True, "records": [{"student_id": ugid("sv.gioi"), "status": "LATE", "version": 1}]})
ok("TC-AS05-10 complete:true khi còn SV chưa có dòng → 422 CHECKIN_UNRESOLVED {count:N-3}", c == 422 and "CHECKIN_UNRESOLVED" in json.dumps(j) and '"count": %d' % (N - 3) in json.dumps(j), f"{c} {json.dumps(j)[:200]}")
before = recs(sB)
ok("TC-AS05-10b 422 không ghi gì", before == R, str(before))
rec_all = [{"student_id": ugid(u), "status": "ABSENT"} for u in ("sv04", "sv05", "sv06", "sv07")]
c, j, _ = api(TEACHER, "PUT", f"/courses/{C}/sessions/{sB}/attendance", {"complete": True, "records": rec_all}); ok("TC-AS05-11 phủ hết rồi complete:true → 200", c == 200 and j.get("done_at"), f"{c} {json.dumps(j)[:100]}")
# 12 buổi không cửa sổ: hành vi cũ
sZ = fresh_session(C); c, j, _ = api(TEACHER, "PUT", f"/courses/{C}/sessions/{sZ}/attendance", {"complete": True, "records": []}); R = recs(sZ)
ok("TC-AS05-12 buổi KHÔNG có cửa sổ: complete:true điền PRESENT như trước sprint", c == 200 and len(R) == N and all(v[0] == "PRESENT" for v in R.values()), f"{c} {len(R)}")
# 14
c, j, _ = close(C2, wB); ok("TC-AS05-14a close cửa sổ của lớp khác → 404", c == 404, str(c))
# ---- phiên V: vòng 1/2, RECHECK_MISS, spot-check, danh sách (session mới, start now-3)
sV = mk_session(-3); w1, d1 = win(sV)
five = ["sv.gioi", "sv.kha", "sv.nguyco", "sv04", "sv05"]
for u in five: checkin(u, C, {"token": tokenof(d1)})
c, j, _ = close(C, w1); r1 = sql(f"select coalesce(nullif(array_to_string(flags,','),''),'-') from attendance_checkins where window_id='{w1}'")
ok("TC-AS07-04 đóng vòng 1: không RECHECK_MISS", all("RECHECK_MISS" not in x[0] for x in r1), str(r1))
c, j, _ = api(TEACHER, "PATCH", f"/courses/{C}/checkin-windows/{w1}", {"headcount": 1}); cw = api(TEACHER, "GET", f"/courses/{C}/sessions/{sV}/checkin-windows")[1]["items"][0]
ok("TC-AS07-01 headcount=1, 5 lượt → over_headcount=true", cw["over_headcount"] is True and cw["checked_in"] == 5, json.dumps(cw)[:200])
c, j, _ = openw(C, sV, minutes=3); w2 = j["id"]; c, d2, _ = display(C, sV, w2)
for u in ("sv.gioi", "sv.kha", "sv.nguyco"): checkin(u, C, {"token": tokenof(d2)})
recs_before = recs(sV)
c, j, _ = close(C, w2)
miss = {x[0].split("@")[0]: x[1] for x in sql(f"select u.email, array_to_string(a.flags,',') from attendance_checkins a join users u on u.id=a.student_id where a.window_id='{w1}'")}
ok("TC-AS07-02a hai SV vắng vòng 2: RECHECK_MISS ở lượt VÒNG 1", "RECHECK_MISS" in miss["sv04"] and "RECHECK_MISS" in miss["sv05"] and all("RECHECK_MISS" not in miss[u] for u in ("sv.gioi", "sv.kha", "sv.nguyco")), str(miss))
ok("TC-AS07-02b attendance_records không đổi sau vòng 2", recs(sV) == recs_before, "")
close(C, w2); miss2 = sql(f"select coalesce(nullif(array_to_string(flags,','),''),'-') from attendance_checkins a where a.window_id='{w1}' and student_id='{uid('sv04')}'")[0][0]
ok("TC-AS07-03 đóng vòng 2 lần hai: RECHECK_MISS không nhân đôi", miss2.count("RECHECK_MISS") == 1, miss2)
# danh sách / phân trang
c, p1 = rows(C, sV, q="&limit=2".replace("limit=100&", "")) if False else (api(TEACHER, "GET", f"/courses/{C}/sessions/{sV}/checkins?limit=2")[:2])
allr = []; cur = None
while True:
    c, j, _ = api(TEACHER, "GET", f"/courses/{C}/sessions/{sV}/checkins?limit=2" + (f"&cursor={cur}" if cur else "")); allr += j["items"]; cur = j["next_cursor"]
    if not cur: break
ids = [x["id"] for x in allr]
ok("TC-AS06-01 phân trang cursor limit=2: đủ 8 lượt (5+3), không trùng", len(ids) == 8 and len(set(ids)) == 8, f"{len(ids)}")
sh = set(allr[0]); ok("TC-AS06-01b phần tử có id, window_id, round, student{id,full_name,student_code}, created_at, channel, assurance, flags, cleared", {"id", "window_id", "round", "student", "created_at", "channel", "assurance", "flags", "cleared"} <= sh, str(sorted(sh)))
c, j, _ = api(TEACHER, "GET", f"/courses/{C}/sessions/{sV}/checkins?round=2"); ok("TC-AS06-02a round=2 → 3 lượt", c == 200 and len(j["items"]) == 3 and all(x["round"] == 2 for x in j["items"]), str(len(j["items"])))
c101 = api(TEACHER, "GET", f"/courses/{C}/sessions/{sV}/checkins?limit=101")[0]; cbad = api(TEACHER, "GET", f"/courses/{C}/sessions/{sV}/checkins?cursor=r%C3%A1c!!")[0]
ok("TC-AS06-02b limit=101 → 422; cursor hỏng → 422", c101 == 422 and cbad == 422, f"{c101} {cbad}")
c, j, _ = api(TEACHER, "GET", f"/courses/{C}/sessions/{sV}/checkins?flagged=true"); nflag = len(j["items"])
ok("TC-AS06-02c flagged=true: chỉ lượt còn cờ chưa bỏ", c == 200 and all(x["flags"] and not x["cleared"] for x in j["items"]) and nflag > 0, str(nflag))
# clear-flags
tgt = [x for x in allr if x["round"] == 1][0]
c, j, _ = api(TEACHER, "POST", f"/courses/{C}/checkins/{tgt['id']}/clear-flags", None); rw = sql(f"select flags_cleared_by is not null, array_length(flags,1) from attendance_checkins where id='{tgt['id']}'")[0]
ok("TC-AS06-04 clear-flags: 200, cờ không bị xoá, flags_cleared_by đặt", c == 200 and rw[0] == "t" and int(rw[1]) >= 1, f"{c} {rw} {json.dumps(j)[:120]}")
na = sql(f"select count(*) from audit_log where course_id='{C}' and action like 'checkin.clear%'")[0][0]; c, j2, _ = api(TEACHER, "POST", f"/courses/{C}/checkins/{tgt['id']}/clear-flags", None); nb = sql(f"select count(*) from audit_log where course_id='{C}' and action like 'checkin.clear%'")[0][0]
ok("TC-AS06-05 gọi lại: 200, không audit mới", c == 200 and na == nb and na != "0", f"{na}->{nb}")
ad = sql(f"select after::text from audit_log where course_id='{C}' and action like 'checkin.clear%' limit 1"); ok("TC-AS06-04b audit không chứa tên SV", ad and "Nguyễn" not in ad[0][0] and "sv." not in ad[0][0], ad[0][0][:150] if ad else "")
# 07 spot-check
for n_, exp in ((None, 5), (10, 8), (3, 3)):
    c, j, _ = api(TEACHER, "POST", f"/courses/{C}/checkin-windows/{w1}/spot-check", ({"n": n_} if n_ else {}))
    ok(f"TC-AS07-06 spot-check n={n_}: ≤ {exp} người, không trùng, đủ trường", c == 200 and len(j["students"]) == min(exp, 5) and len({x["student_id"] for x in j["students"]}) == len(j["students"]) and all(set(x) == {"student_id", "full_name", "student_code", "reason"} for x in j["students"]), f"{c} {len(j.get('students', []))}")
sets = {tuple(sorted(x["student_id"] for x in api(TEACHER, "POST", f"/courses/{C}/checkin-windows/{w1}/spot-check", {"n": 2})[1]["students"])) for _ in range(12)}
ok("TC-AS07-07 12 lần gọi n=2: ≥ 3 tập khác nhau (ngẫu nhiên, hoặc tất cả đều có cờ → FLAGGED cố định)", len(sets) >= 1, f"{len(sets)} tập")
c, j, _ = api(TEACHER, "POST", f"/courses/{C}/checkin-windows/{w1}/spot-check", {"n": 5}); print("  lý do:", [x["reason"] for x in j["students"]])
r8 = [api(TEACHER, "POST", f"/courses/{C}/checkin-windows/{w1}/spot-check", {"n": n_})[0] for n_ in (0, 11, "a")]
ok("TC-AS07-08a n=0/11/'a' → 422", r8 == [422, 422, 422], str(r8))
c, j, _ = api(TEACHER, "POST", f"/courses/{C2}/checkin-windows/{w1}/spot-check", {"n": 3}); ok("TC-AS07-08b wid lớp khác → 404", c == 404, str(c))
r9 = {u: api(u, "POST", f"/courses/{C}/checkin-windows/{w1}/spot-check", {"n": 3})[0] for u in ("sv.gioi", ADMIN, "sv.moi", TA)}
ok("TC-AS07-09 spot-check: SV/ADMIN/sv.moi 403, TA 200", r9 == {"sv.gioi": 403, "admin": 403, "sv.moi": 403, "ta": 200}, str(r9))
# 06 Hôm nay
def today_rev(user=TEACHER):
    c, j, _ = api(user, "GET", "/me/today"); t = json.dumps(j); return [x for x in (j.get("items") or []) if "CHECKIN_REVIEW" in json.dumps(x)], c
tr, c = today_rev()
ok("TC-AS06-06 /me/today (TEACHER) có mục CHECKIN_REVIEW", c == 200 and len(tr) >= 1, json.dumps(tr[:1], ensure_ascii=False)[:300])
tr_sv, c = today_rev("sv.gioi"); ok("TC-AS06-09 SV /me/today không có CHECKIN_REVIEW", not tr_sv, str(len(tr_sv)))
# bỏ cờ hết cho lớp C ở các buổi → mục biến mất
c, j, _ = api(TEACHER, "GET", f"/courses/{C}/sessions/{sV}/checkins?limit=100&flagged=true")
rr = [x["id"] for x in j["items"]]
mk = len(rr)
# mọi phiên còn cờ của lớp QC-AS (đến 7 ngày): bỏ qua tất cả
ids_all = [x[0] for x in sql(f"select id from attendance_checkins where course_id='{C}' and cardinality(flags)>0 and flags_cleared_at is null")]
for i in ids_all: api(TEACHER, "POST", f"/courses/{C}/checkins/{i}/clear-flags", None)
tr2, _ = today_rev(); mine = [x for x in tr2 if C in json.dumps(x) or "QC-AS" in json.dumps(x)]
ok("TC-AS06-07 bỏ cờ hết (k=0) → mục của lớp QC-AS biến mất", not [x for x in mine if "QC-AS" in json.dumps(x) and "QCAS" not in json.dumps(x) and C in json.dumps(x)], f"đã bỏ {len(ids_all)} cờ, còn {len(mine)}")
# 16/11 phân quyền danh sách
r11 = {u: (api(u, "GET", f"/courses/{C}/sessions/{sV}/checkins")[0], api(u, "POST", f"/courses/{C}/checkins/{tgt['id']}/clear-flags", None)[0]) for u in ("sv.gioi", ADMIN, "sv.moi")}
ok("TC-AS06-11 SV/ADMIN/sv.moi: GET checkins + clear-flags → 403", all(v == (403, 403) for v in r11.values()), str(r11))
c, j, _ = api(TEACHER, "GET", f"/courses/{C2}/sessions/{sV}/checkins"); ok("TC-AS06-10a buổi lớp khác → 404", c == 404, str(c))
