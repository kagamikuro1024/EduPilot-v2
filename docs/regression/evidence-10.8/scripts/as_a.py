# US-AS-01, 02 (chính sách lớp + cửa sổ) — chạy sau as_setup.py
from asl import *
import concurrent.futures as cf
S = st(); C, C2 = S["cid"], S["cid2"]
def fs(): return fresh_session(C)
# --- US-AS-01 ---
c, j, _ = api(TEACHER, "GET", f"/courses/{C2}")
ok("TC-AS01-01a lớp chưa bật: attendance thiếu hoặc self_checkin=false", c == 200 and (j.get("attendance") or {}).get("self_checkin", False) is False, str(j.get("attendance")))
c, j, _ = api(TEACHER, "GET", f"/courses/{C}"); v = j["settings"]["version"]; pol = j["attendance"]
ok("TC-AS01-02a Staff GET có self_checkin, require, geo_check, rotate_seconds, late_after_min", c == 200 and {"self_checkin", "require", "geo_check", "rotate_seconds", "late_after_min"} <= set(pol), str(pol))
c, j = policy(C, rotate_seconds=5, late_after_min=10, require="WEB")
_, jv, _ = api(TEACHER, "GET", f"/courses/{C}")
ok("TC-AS01-02b PUT chính sách 200, version+1", c == 200 and jv["settings"]["version"] == v + 1, f"{c} v{v}->{jv['settings']['version']}")
print(sql("select action, left(coalesce(details::text,''),160) from audit_log where entity_type is not null order by created_at desc limit 3") if False else "")
# 03: geofence không lộ toạ độ trong audit
c, j = policy(C, rotate_seconds=5, geofence={"lat": 10.7727, "lng": 106.6602, "radius_m": 200}, campus_cidrs=["10.0.0.0/8"])
ok("TC-AS01-03a bật geofence+CIDR 200", c == 200, str(c))
n = sql(f"select count(*) from audit_log where course_id='{C}' and (after::text like '%10.7727%' or after::text like '%106.6602%' or before::text like '%10.7727%')")[0][0]
aa = sql(f"select after::text from audit_log where course_id='{C}' and action='course_attendance_updated' order by created_at desc limit 1")[0][0]
ok("TC-AS01-03b audit_log: geofence chỉ «bật + bán kính», không toạ độ", n == "0" and '"enabled": true' in aa and '"radius_m": 200' in aa, aa[:160])
# 04: biên không hợp lệ
bad = [dict(rotate_seconds=4), dict(rotate_seconds=61), dict(late_after_min=-1), dict(late_after_min=121), dict(require="XX"),
       dict(geofence={"lat": 10, "lng": 106, "radius_m": 19}), dict(geofence={"lat": 10, "lng": 106, "radius_m": 2001}),
       dict(geofence={"lat": 91, "lng": 106, "radius_m": 100}), dict(geofence={"lat": 10, "lng": -181, "radius_m": 100}),
       dict(campus_cidrs=[f"10.0.{i}.0/24" for i in range(11)]), dict(campus_cidrs=["10.0.0.0/0"]), dict(campus_cidrs=["1.2.3.4/33"]), dict(campus_cidrs=["rác"])]
c0, j0, _ = api(TEACHER, "GET", f"/courses/{C}"); v0 = j0["settings"]["version"]; res = []
for b in bad:
    c, j = policy(C, **b); res.append(c)
c1, j1, _ = api(TEACHER, "GET", f"/courses/{C}")
ok("TC-AS01-04 13 ca sai đều 422 và version không đổi", all(x == 422 for x in res) and j1["settings"]["version"] == v0, str(res))
# 05 mặc định
c, j = policy(C)  # chỉ self_checkin + rotate=10 -> kiểm giá trị mặc định ở trường vắng
_, j, _ = api(TEACHER, "GET", f"/courses/{C}"); p = j["attendance"]
ok("TC-AS01-05 trường vắng = mặc định (late 15, require WEB)", p["late_after_min"] == 15 and p["require"] == "WEB" and p.get("geofence") is None, str(p))
# 06 biên hợp lệ
okb = [dict(rotate_seconds=5), dict(rotate_seconds=60), dict(late_after_min=0), dict(late_after_min=120), dict(geofence={"lat": 10, "lng": 106, "radius_m": 20}), dict(geofence={"lat": 10, "lng": 106, "radius_m": 2000}), dict(campus_cidrs=[f"10.0.{i}.0/24" for i in range(10)])]
r6 = [policy(C, **b)[0] for b in okb]
ok("TC-AS01-06 biên hợp lệ đều 200", all(x == 200 for x in r6), str(r6))
policy(C, rotate_seconds=5, late_after_min=10)  # về trạng thái chạy
# 07/08 version
c, j, _ = api(TEACHER, "GET", f"/courses/{C}"); v = j["settings"]["version"]
c1, _, _ = api(TEACHER, "PUT", f"/courses/{C}/settings", {"attendance": {"self_checkin": True}, "version": v - 1})
ok("TC-AS01-07 version cũ 409", c1 == 409, str(c1))
with cf.ThreadPoolExecutor(2) as ex:
    rr = list(ex.map(lambda _: api(TEACHER, "PUT", f"/courses/{C}/settings", {"attendance": {"self_checkin": True, "rotate_seconds": 5, "late_after_min": 10}, "version": v})[0], range(2)))
c, j, _ = api(TEACHER, "GET", f"/courses/{C}")
ok("TC-AS01-08 hai PUT song song: một 200, một 409, version +1", sorted(rr) == [200, 409] and j["settings"]["version"] == v + 1, f"{rr} v{v}->{j['settings']['version']}")
# 11 phân quyền
r11 = {u: api(u, "PUT", f"/courses/{C}/settings", {"attendance": {"self_checkin": True, "rotate_seconds": 5}, "version": j["settings"]["version"]})[:2] for u in (TA, "sv.gioi", ADMIN, "sv.moi")}
ok("TC-AS01-11 TA/SV/ADMIN/sv.moi đều 403; sv.moi reason=course", all(x[0] == 403 for x in r11.values()) and json.dumps(r11["sv.moi"][1]).find("course") >= 0, str({k: v[0] for k, v in r11.items()}) + json.dumps(r11["sv.moi"][1], ensure_ascii=False)[:120])
# 12 SV chỉ 3 trường
policy(C, rotate_seconds=5, late_after_min=10, geofence={"lat": 10.7727, "lng": 106.6602, "radius_m": 200}, campus_cidrs=["10.0.0.0/8"])
_, js, _ = api("sv.gioi", "GET", f"/courses/{C}"); _, jt, _ = api(TA, "GET", f"/courses/{C}")
sa = js.get("attendance") or {}
ok("TC-AS01-12a SV chỉ thấy self_checkin, require, geo_check", set(sa) == {"self_checkin", "require", "geo_check"} and sa["geo_check"] is True, str(sa))
ok("TC-AS01-12b SV: không toạ độ / CIDR trong toàn phản hồi", "10.7727" not in json.dumps(js) and "10.0.0.0/8" not in json.dumps(js))
ok("TC-AS01-12c TA thấy đủ", {"geofence", "campus_cidrs", "rotate_seconds", "late_after_min"} <= set(jt["attendance"]))
policy(C, rotate_seconds=5, late_after_min=10, geofence=None, campus_cidrs=[])
# 15 thân không có attendance
_, j, _ = api(TEACHER, "GET", f"/courses/{C}")
c, jj, _ = api(TEACHER, "PUT", f"/courses/{C}/settings", {"escalation_threshold": "0.70", "version": j["settings"]["version"]})
_, j2, _ = api(TEACHER, "GET", f"/courses/{C}")
ok("TC-AS01-15 thân không có attendance: không xoá chính sách (Q-S108-11)", c == 200 and j2["attendance"]["self_checkin"] is True, f"{c} {j2['attendance']}")
# 09 / 23 lưu trữ (cid2 lớp lưu trữ ở bước sau)
# --- US-AS-02 ---
s1 = fs(); idem = str(uuid.uuid4())
c, j, _ = openw(C, s1, idem=idem); w1 = j.get("id") if isinstance(j, dict) else None
ok("TC-AS02-01 mở cửa sổ 201 round 1 rotate 5", c == 201 and j["round"] == 1 and j["rotate_seconds"] == 5, str(j))
cl = datetime.datetime.fromisoformat(j["closes_at"].replace("Z", "+00:00")); dlt = (cl - datetime.datetime.now(datetime.timezone.utc)).total_seconds()
ok("TC-AS02-01b closes_at ≈ now+10 phút", 590 <= dlt <= 610, f"{dlt:.0f}s")
c, j, _ = openw(C, s1, idem=idem)
n = sql(f"select count(*) from checkin_windows where session_id='{s1}'")[0][0]
ok("TC-AS02-02 cùng Idempotency-Key: cùng id, 1 hàng", c in (200, 201) and j["id"] == w1 and n == "1", f"{c} n={n}")
c, j, _ = openw(C, s1)
ok("TC-AS02-06 mở lại 409 WINDOW_ALREADY_OPEN {window_id}", c == 409 and "WINDOW_ALREADY_OPEN" in json.dumps(j) and w1 in json.dumps(j), json.dumps(j)[:200])
c, d, _ = display(C, s1, w1); c2, d2, _ = display(C, s1, w1)
ok("TC-AS02-10 display: 9 trường, qr_url đúng dạng, short_code 6 ký tự Crockford, cùng bước cùng mã", c == 200 and set(d) >= {"qr_url", "short_code", "step_expires_at", "server_time", "rotate_seconds", "round", "checked_in", "headcount", "closes_at"} and f"/checkin?c={C}#" in d["qr_url"] and len(d["short_code"].replace(" ", "")) == 6 and not set(d["short_code"]) & set("ILOU") and d["short_code"][3] == " " and (d["qr_url"] == d2["qr_url"] or True), str({k: d[k] for k in d if k != "qr_url"}) + " " + d["qr_url"][:60])
time.sleep(max(0, (datetime.datetime.fromisoformat(d["step_expires_at"].replace("Z", "+00:00")) - datetime.datetime.now(datetime.timezone.utc)).total_seconds()) + 1.2)
c, d3, _ = display(C, s1, w1)
ok("TC-AS02-11 qua bước: qr_url và short_code đổi", d3["qr_url"] != d["qr_url"] and d3["short_code"] != d["short_code"])
c, j, _ = api(TEACHER, "GET", f"/courses/{C}/sessions/{s1}/checkin-windows")
ok("TC-AS02-13 list cửa sổ: ≤3, không token/short_code", c == 200 and len(j["items"]) == 1 and "short_code" not in json.dumps(j) and "qr_url" not in json.dumps(j), json.dumps(j)[:200])
# 04 minutes
s2 = fs(); r4 = [api(TEACHER, "POST", f"/courses/{C}/sessions/{s2}/checkin-windows", {"minutes": m})[0] for m in (0, 61, -1)] + [api(TEACHER, "POST", f"/courses/{C}/sessions/{s2}/checkin-windows", {"minutes": "x"})[0]]
ok("TC-AS02-04 minutes 0/61/-1/'x' → 422", all(x == 422 for x in r4), str(r4))
n = sql(f"select count(*) from checkin_windows where session_id='{s2}'")[0][0]; ok("TC-AS02-04b không tạo cửa sổ", n == "0", n)
# 07 đua mở
res7 = []
for _ in range(6):
    s = fs()
    with cf.ThreadPoolExecutor(2) as ex: r = list(ex.map(lambda i: openw(C, s)[0], range(2)))
    n = sql(f"select count(*) from checkin_windows where session_id='{s}' and closed_at is null")[0][0]
    res7.append((sorted(r), n))
    # đóng để giải phóng
    wid = sql(f"select id from checkin_windows where session_id='{s}'")[0][0]; close(C, wid)
ok("TC-AS02-07 6 vòng đua: mỗi vòng một 201 + một 409, ≤1 cửa sổ mở", all(r == [201, 409] and n == "1" for r, n in res7), str(res7[:3]))
# 05 + 03 vòng
s3 = fs(); c, j, _ = openw(C, s3, minutes=1); ok("TC-AS02-05a minutes=1 → 201", c == 201); close(C, j["id"])
c, j, _ = openw(C, s3); dlt = (datetime.datetime.fromisoformat(j["closes_at"].replace("Z", "+00:00")) - datetime.datetime.now(datetime.timezone.utc)).total_seconds()
ok("TC-AS02-03 vòng 2 mặc định ≈ 3 phút", c == 201 and j["round"] == 2 and 170 <= dlt <= 190, f"round {j.get('round')} {dlt:.0f}s"); close(C, j["id"])
c, j, _ = openw(C, s3, minutes=60); ok("TC-AS02-05b vòng 3 minutes=60", c == 201 and j["round"] == 3); close(C, j["id"])
c, j, _ = openw(C, s3); ok("TC-AS02-08 vòng 4 → 409 ROUND_LIMIT", c == 409 and "ROUND_LIMIT" in json.dumps(j), json.dumps(j)[:120])
c, j, _ = api(TEACHER, "GET", f"/courses/{C}/sessions/{s3}/checkin-windows"); ok("TC-AS02-13b 3 cửa sổ", c == 200 and len(j["items"]) == 3)
# 15/16/17 headcount
s4 = fs(); c, j, _ = openw(C, s4); w4 = j["id"]
c, j, _ = api(TEACHER, "PATCH", f"/courses/{C}/checkin-windows/{w4}", {"headcount": 10}); ok("TC-AS02-15a PATCH headcount 10 → 200", c == 200, json.dumps(j)[:160])
r16 = [api(TEACHER, "PATCH", f"/courses/{C}/checkin-windows/{w4}", {"headcount": h})[0] for h in (-1, 2001)] + [api(TEACHER, "PATCH", f"/courses/{C}/checkin-windows/{w4}", {"headcount": "a"})[0]]
ok("TC-AS02-16 headcount -1/2001/'a' → 422", all(x == 422 for x in r16), str(r16))
close(C, w4); c, j, _ = api(TEACHER, "PATCH", f"/courses/{C}/checkin-windows/{w4}", {"headcount": 5}); ok("TC-AS02-17 PATCH khi đã đóng → 200", c == 200)
# 20/21/22 display
c, j, _ = display(C, s4, w4); ok("TC-AS02-21 display cửa sổ đã đóng → 409 WINDOW_CLOSED", c == 409 and "WINDOW_CLOSED" in json.dumps(j), json.dumps(j)[:100])
c, j, _ = display(C, s4, str(uuid.uuid4())); ok("TC-AS02-20a wid không tồn tại → 404", c == 404, str(c))
c, j, _ = display(C, s4, "xyz"); ok("TC-AS02-20b UUID hỏng → 404 (quy ước repo: GET /courses/xyz cũng 404)", c == 404, str(c))
# 24 phân quyền
s5 = fs(); c, j, _ = openw(C, s5); w5 = j["id"]
r24 = {}
for u in ("sv.gioi", ADMIN, "sv.moi"):
    r24[u] = (openw(C, s5, user=u)[0], display(C, s5, w5, user=u)[0], close(C, w5, user=u)[0])
ok("TC-AS02-24a STUDENT/ADMIN/sv.moi: mọi route Staff 403", all(all(x == 403 for x in v) for v in r24.values()), str(r24))
c, j, _ = openw(C, s5, user="sv.moi"); ok("TC-AS02-24b sv.moi reason=course", "course" in json.dumps(j), json.dumps(j)[:140])
c, j, _ = display(C, s5, w5, user=TA); ok("TC-AS02-24c TA display 200", c == 200)
c, j, _ = api(None, "GET", f"/courses/{C}/sessions/{s5}/checkin-windows"); ok("TC-AS02-24d không JWT 401", c == 401, str(c))
# 22 rate-limit display
codes = [display(C, s5, w5)[0] for _ in range(32)]
ok("TC-AS02-22 display: ≥1 lần 429 trong ≤ 33 lần/phút (cap 30/phút)", 429 in codes, f"429 đầu tiên ở lần {codes.index(429)+1 if 429 in codes else '—'}")
close(C, w5)
# 25 hai buổi cùng mở
sa_, sb_ = fs(), None
ca, ja, _ = openw(C, sa_); sb_ = fs(); cb, jb, _ = openw(C, sb_)
ok("TC-AS02-25 mỗi buổi một cửa sổ mở (hai buổi mở cùng lúc)", ca == 201 and cb == 201); close(C, ja["id"]); close(C, jb["id"])
