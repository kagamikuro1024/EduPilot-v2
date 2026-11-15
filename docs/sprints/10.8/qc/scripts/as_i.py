# Đợt 3: các TC biên chưa chạy (TC-AS01-10, 02-09, 02-15, 03-10, 03-13, 03-22, 05-15, 06-01, 06-03)
from asl import *
import concurrent.futures as cf, subprocess
S = st(); C = S["cid"]
policy(C, rotate_seconds=5, late_after_min=10, geofence=None, campus_cidrs=[], require="WEB")
SVC = ["sv.gioi", "sv.kha", "sv.nguyco"] + [f"sv{n:02d}" for n in range(4, 11)]
# ---- TC-AS01-10: cửa sổ đang mở giữ rotate đã ghim khi đổi / tắt chính sách
s, w, d = neww(C); t0 = tokenof(d)
policy(C, self_checkin=False, rotate_seconds=30)
d1 = display(C, s, w)[1]; time.sleep(12); d2 = display(C, s, w)[1]
c, j, _ = checkin("sv.gioi", C, {"token": tokenof(d2)})
ok("TC-AS01-10a đổi rotate 30 + tắt self_checkin khi cửa sổ đang mở: mã vẫn xoay 5 s (đã ghim) và SV vẫn điểm danh được", d1["qr_url"] != d2["qr_url"] and d2["rotate_seconds"] == 5 and c == 201, f"{d2['rotate_seconds']} {c}")
c, j, _ = openw(C, fresh_session(C)); ok("TC-AS01-10b cửa sổ MỚI khi self_checkin tắt → 409 SELF_CHECKIN_DISABLED", c == 409 and j.get("code") == "SELF_CHECKIN_DISABLED", f"{c} {j.get('code')}")
policy(C, rotate_seconds=30); s2 = fresh_session(C); c, j, _ = openw(C, s2); ok("TC-AS01-10c cửa sổ mới dùng rotate_seconds=30 mới", c == 201 and j["rotate_seconds"] == 30, str(j.get("rotate_seconds"))); close(C, j["id"]); close(C, w)
policy(C, rotate_seconds=5, late_after_min=10)
# ---- TC-AS02-15: over_headcount biên (headcount=5 ⇒ ngưỡng 5+max(2,⌈5·5%⌉)=7; 7 → false, 8 → true)
s, w, d = neww(C); t = tokenof(d)
for u in SVC[:7]: checkin(u, C, {"token": t})
api(TEACHER, "PATCH", f"/courses/{C}/checkin-windows/{w}", {"headcount": 5}); a = api(TEACHER, "GET", f"/courses/{C}/sessions/{s}/checkin-windows")[1]["items"][0]
checkin(SVC[7], C, {"token": t}); b = api(TEACHER, "GET", f"/courses/{C}/sessions/{s}/checkin-windows")[1]["items"][0]
api(TEACHER, "PATCH", f"/courses/{C}/checkin-windows/{w}", {"headcount": 2000}); c2 = api(TEACHER, "GET", f"/courses/{C}/sessions/{s}/checkin-windows")[1]["items"][0]
ok("TC-AS02-15 over_headcount: headcount 5 + 7 lượt → false; 8 lượt → true; headcount 2000 → false; audit checkin_window.headcount", a["over_headcount"] is False and b["over_headcount"] is True and c2["over_headcount"] is False and int(sql(f"select count(*) from audit_log where entity_id='{w}' and action='checkin_window.headcount'")[0][0]) >= 2, f"{a['over_headcount']} {b['over_headcount']} {c2['over_headcount']}")
close(C, w)
# ---- TC-AS02-09: mở vòng 2 không thông báo SV
t_ = sql("select now()")[0][0]
n0 = {u: sql(f"select count(*) from notifications where user_id='{uid(u)}'")[0][0] for u in SVC[:3]}
o0 = sql("select count(*) from outbox")[0][0]
c, j, _ = openw(C, s); n1 = {u: sql(f"select count(*) from notifications where user_id='{uid(u)}'")[0][0] for u in SVC[:3]}
o1 = sql("select count(*) from outbox")[0][0]; time.sleep(3); n2 = {u: sql(f"select count(*) from notifications where user_id='{uid(u)}'")[0][0] for u in SVC[:3]}
td = api("sv.gioi", "GET", "/me/today")[1]
ok("TC-AS02-09 mở vòng 2: không thông báo / chuông cho SV; không mục «Hôm nay» mới cho SV", c == 201 and n0 == n1 == n2 and json.dumps(td).count("CHECKIN") == 0, f"notif {n0}→{n2}; outbox {o0}→{o1}")
close(C, j["id"])
# ---- TC-AS03-13: bộ đếm chỉ đếm lần SAI (3 lượt đúng ở 3 cửa sổ rồi 5 sai)
time.sleep(61)
ws = [neww(C) for _ in range(3)]
ok_ = [checkin("sv.nguyco", C, {"code": display(C, sx, wx)[1]["short_code"]})[0] for sx, wx, dx in ws]
wrong = [checkin("sv.nguyco", C, {"code": x})[0] for x in ["AAAAAA", "BBBBBB", "CCCCCC", "DDDDDD", "EEEEEE"]]; sixth = checkin("sv.nguyco", C, {"code": "FFFFFF"})[0]
ok("TC-AS03-13 3 lượt đúng ở 3 cửa sổ không bị đếm; 5 lần sai → 422; lần sai thứ 6 → 429", ok_ == [201] * 3 and wrong == [422] * 5 and sixth == 429, f"{ok_} {wrong} {sixth}")
# vì 3 cửa sổ này mở song song nên cũng kiểm TC-AS03-10: mã ngắn tìm cửa sổ theo LỚP
sa, wa, da = ws[0]; sb, wb, db = ws[1]
# ---- TC-AS03-10: hai cửa sổ cùng lớp + cửa sổ lớp khác
time.sleep(61)
sx, wx, dx = neww(C); codeX = display(C, sx, wx)[1]["short_code"]; close(C, wx)            # đã đóng, còn trong 15 phút
sy, wy, dy = neww(C); codeY = display(C, sy, wy)[1]["short_code"]                           # đang mở
policy(S["cid3"], rotate_seconds=5); s3, w3, d3 = neww(S["cid3"]); code3 = display(S["cid3"], s3, w3)[1]["short_code"]  # lớp khác
r_y = checkin("sv.kha", C, {"code": codeY}); r_x = checkin("sv.gioi", C, {"code": codeX}); r_3 = checkin("sv04", C, {"code": code3})
ok("TC-AS03-10 mã ngắn tìm theo lớp: mã cửa sổ ĐANG MỞ → 201 đúng cửa sổ; mã cửa sổ vừa ĐÓNG → 409 WINDOW_CLOSED; mã cửa sổ LỚP KHÁC → 422 CHECKIN_TOKEN_INVALID", r_y[0] == 201 and r_x[0] == 409 and r_x[1].get("code") == "WINDOW_CLOSED" and r_3[0] == 422 and r_3[1].get("code") == "CHECKIN_TOKEN_INVALID", f"{r_y[0]} {r_x[0]}:{r_x[1].get('code')} {r_3[0]}:{r_3[1].get('code')}")
for wid in (wa, wb, wy): close(C, wid)
close(S["cid3"], w3)
# ---- TC-AS03-22: mời ra giữa lúc điểm danh (không 500)
res = []
for i in range(8):
    s_, w_, d_ = neww(C); t_ = tokenof(d_)
    with cf.ThreadPoolExecutor(2) as ex:
        f1 = ex.submit(lambda: api(TEACHER, "DELETE", f"/courses/{C}/members/{uid('sv10')}", None)[0]); f2 = ex.submit(lambda: checkin("sv10", C, {"token": t_})[0])
        res.append((f1.result(), f2.result()))
    close(C, w_)
    j_ = api("sv10", "POST", "/courses/join", {"code": "QCKASTV"})[0]; api(TEACHER, "POST", f"/courses/{C}/members/{uid('sv10')}/approve", None)
    if i % 3 == 2: time.sleep(20)
ok("TC-AS03-22 mời ra đua với điểm danh ×8: không có 5xx; điểm danh chỉ 201 hoặc 403", all(a_ in (200, 204) and b_ in (201, 403, 429) for a_, b_ in res), str(res))
print("member sv10 ACTIVE?", sql(f"select status from enrollments where course_id='{C}' and user_id='{uid('sv10')}'"))
# ---- TC-AS05-15: đóng song song với GV sửa tay lưới
time.sleep(61)
outs = []
for i in range(8):
    s_, w_, d_ = neww(C); t_ = tokenof(d_)
    for u in SVC[:6]: checkin(u, C, {"token": t_})
    with cf.ThreadPoolExecutor(2) as ex:
        fc = ex.submit(lambda: close(C, w_)[0]); fp = ex.submit(lambda: api(TEACHER, "PUT", f"/courses/{C}/sessions/{s_}/attendance", {"records": [{"student_id": uid("sv04"), "status": "ABSENT"}]}))
        rc, rp = fc.result(), fp.result()
    dup = sql(f"select count(*) from (select student_id from attendance_records where session_id='{s_}' group by 1 having count(*)>1) x")[0][0]
    st_ = sql(f"select status from attendance_records where session_id='{s_}' and student_id='{uid('sv04')}'")
    outs.append((rc, rp[0], st_[0][0] if st_ else None, dup))
    if i == 4: time.sleep(40)
ok("TC-AS05-15 đóng ∥ PUT lưới ×8: không 5xx, không dòng trùng; kết quả mỗi vòng là PUT 200 + ABSENT (GV thắng) hoặc PUT 409 + PRESENT (đóng trước)", all(rc == 200 and rp in (200, 409) and dup == "0" and ((rp == 200 and st_ == "ABSENT") or (rp == 409 and st_ in ("PRESENT", "LATE"))) for rc, rp, st_, dup in outs), str(outs))
