# US-AS-03 (ca lỗi, hạn mức, mã ngắn), US-AS-01/02/03/05/06 trên lớp lưu trữ
from asl import *
import re
S = st(); C, C2 = S["cid"], S["cid2"]
policy(C, rotate_seconds=5, late_after_min=10, geofence=None, campus_cidrs=[])
cool = lambda: time.sleep(61)
body = lambda c, j: (c, re.sub(r"[0-9a-f]{32}", "X", json.dumps(j.get("code") if isinstance(j, dict) else j)), len(json.dumps(j.get("details") or {}) if isinstance(j, dict) else ""))
# --- 04 / 11: hết hạn
s, w, d = neww(C); t = tokenof(d); code = d["short_code"]; time.sleep(12.5)
c, j, _ = checkin("sv05", C, {"token": t}); ok("TC-AS03-04 token > 2 bước → 422 CHECKIN_TOKEN_EXPIRED (không TOKEN_EXPIRED)", c == 422 and j.get("code") == "CHECKIN_TOKEN_EXPIRED", f"{c} {j.get('code')}")
c, j, _ = checkin("sv06", C, {"code": code}); ok("TC-AS03-11 mã ngắn hết hạn → 422 CHECKIN_TOKEN_INVALID", c == 422 and j.get("code") == "CHECKIN_TOKEN_INVALID", f"{c} {j.get('code')}")
# --- 05: token sai
s, w, d = neww(C); t = tokenof(d); s2_, w2_, d2_ = neww(C2); t_other = tokenof(d2_)
var = {"mac": t[:-1] + ("A" if t[-1] != "A" else "B"), "wid_lạ": uuid.uuid4().hex + t[32:], "rác": "abc", "lớp_khác": t_other, "cụt": t[:20]}
res = {k: checkin("sv.kha", C, {"token": v}) for k, v in var.items()}
bs = {k: body(r[0], r[1]) for k, r in res.items()}
ok("TC-AS03-05 5 kiểu token sai đều 422 CHECKIN_TOKEN_INVALID, cùng dạng phản hồi", all(r[0] == 422 and r[1].get("code") == "CHECKIN_TOKEN_INVALID" for r in res.values()) and len(set(bs.values())) == 1, str(bs))
# --- 06 đóng
s, w, d = neww(C); t = tokenof(d); close(C, w); c, j, _ = checkin("sv.nguyco", C, {"token": t}); ok("TC-AS03-06 cửa sổ đã đóng, token đúng → 409 WINDOW_CLOSED", c == 409 and j.get("code") == "WINDOW_CLOSED", f"{c} {j.get('code')}")
# --- 07 thứ tự kiểm
ca = raw("POST", f"/courses/{C}/checkins", {"token": "x"}, {"Idempotency-Key": str(uuid.uuid4())})[0]
cb, jb, _ = checkin("sv.moi", C, {"token": "x"})
policy(C2, self_checkin=False); cc, jc, _ = checkin("sv.gioi", C2, {"token": "x"}); policy(C2, rotate_seconds=5)
cd, jd, _ = checkin("sv.gioi", C, {"token": t[:-1] + "A"})
ok("TC-AS03-07 thứ tự kiểm: 401 → 403 → 409 SELF_CHECKIN_DISABLED → (đóng+MAC sai)", ca == 401 and cb == 403 and cc == 409 and jc.get("code") == "SELF_CHECKIN_DISABLED", f"{ca} {cb} {cc}:{jc.get('code')} | đóng+MAC sai → {cd}:{jd.get('code')}")
# --- 03 bước trước
cool()
s, w, d = neww(C); t1 = tokenof(d); wait_step(d, 0.8); c, j, _ = checkin("sv04", C, {"token": t1}); ok("TC-AS03-03 token của bước trước (≤ 1 bước) → 201", c == 201, f"{c} {j.get('code')}")
# --- 08/09 mã ngắn
s, w, d = neww(C); cd_ = d["short_code"]; a, b = cd_.split(" "); 
c1, j1, _ = checkin("sv.gioi", C, {"code": cd_.replace(" ", "").lower()}); c2, j2, _ = checkin("sv.kha", C, {"code": f"{a}-{b}"}); c3, j3, _ = checkin("sv.nguyco", C, {"code": f" {a} {b} "})
sub = cd_.replace(" ", "").replace("0", "O").replace("1", "I"); c4, j4, _ = checkin("sv05", C, {"code": sub}); r = {x[0].split("@")[0]: x[1] for x in dbrows(s, w)}
ok("TC-AS03-08/09 mã ngắn: chữ thường / gạch / khoảng trắng / 0→O,1→I đều 201, channel=CODE", [c1, c2, c3] == [201] * 3 and set(r.values()) == {"CODE"} and (c4 == 201 or ("0" not in cd_ and "1" not in cd_)), f"{[c1, c2, c3, c4]} {r} code={cd_}")
# --- 12 mã sai 5 lần
cool(); s, w, d = neww(C)
bad = ["AAAAAA", "BBBBBB", "CCCCCC", "DDDDDD", "EEEEEE"]
r5 = [checkin("sv.kha", C, {"code": x}) for x in bad]; r6 = raw("POST", f"/courses/{C}/checkins", {"code": "FFFFFF"}, {"Idempotency-Key": str(uuid.uuid4())}, token=tok("sv.kha"))
good = display(C, s, w)[1]["short_code"]; r7 = checkin("sv.kha", C, {"code": good})
ok("TC-AS03-12a 5 lần sai → 422; lần 6 → 429 + Retry-After; nhập đúng trong phút đó vẫn 429", all(x[0] == 422 for x in r5) and r6[0] == 429 and "Retry-After" in {k.title(): v for k, v in r6[3].items()} and r7[0] == 429, f"{[x[0] for x in r5]} {r6[0]} {r7[0]} hd={ {k:v for k,v in r6[3].items() if k.lower()=='retry-after'} }")
cool(); good = display(C, s, w)[1]["short_code"]; r8 = checkin("sv.kha", C, {"code": good})
ok("TC-AS03-12b sau hết phút: nhập đúng → 201", r8[0] == 201, f"{r8[0]} {r8[1]}")
# --- 20 hạn mức 10/phút (đường token)
cool(); s, w, d = neww(C)
r = [checkin("sv05", C, {"token": "bad"})[0] for _ in range(11)]
ok("TC-AS03-20 11 lượt/phút (token sai): lượt thứ 11 → 429", r[-1] == 429 and r[:10].count(422) == 10, str(r))
# --- 21 phân quyền
cool(); s, w, d = neww(C); t = tokenof(d)
r21 = {u: checkin(u, C, {"token": t})[:2] for u in (TA, TEACHER, ADMIN, "sv.moi")}
ok("TC-AS03-21 TA/TEACHER/ADMIN/sv.moi POST checkins → 403; sv.moi reason=course; 401 không JWT", all(v[0] == 403 for v in r21.values()) and "course" in json.dumps(r21["sv.moi"][1]) and raw("POST", f"/courses/{C}/checkins", {"token": t}, {"Idempotency-Key": str(uuid.uuid4())})[0] == 401, str({k: v[0] for k, v in r21.items()}))
# --- 15 vòng 2
c, j, _ = checkin("sv.gioi", C, {"token": t}); close(C, w); c2, j2, _ = openw(C, s, minutes=3); d2 = display(C, s, j2["id"])[1]; c3, j3, _ = checkin("sv.gioi", C, {"token": tokenof(d2)})
ok("TC-AS03-15 vòng 2: cùng SV điểm danh lại → 201 round 2, hàng thứ hai", c3 == 201 and j3["round"] == 2 and sql(f"select count(*) from attendance_checkins where session_id='{s}' and student_id='{uid('sv.gioi')}'")[0][0] == "2", f"{c3} {j3}")
close(C, j2["id"])
# --- lớp lưu trữ
s, w, d = neww(C2); t = tokenof(d); cid_w = w
c, j, _ = api(ADMIN, "POST", f"/admin/courses/{C2}/archive", None); print("archive", c)
r = {"checkin": checkin("sv.gioi", C2, {"token": t}), "settings": (lambda v: api(TEACHER, "PUT", f"/courses/{C2}/settings", {"attendance": {"self_checkin": True}, "version": v}))(api(TEACHER, "GET", f"/courses/{C2}")[1]["settings"]["version"]),
     "open": openw(C2, fresh_session(C2) if False else sessions(TEACHER, C2)[5]["id"]), "close": close(C2, w), "absent": api(TEACHER, "POST", f"/courses/{C2}/sessions/{s}/absent-unchecked", None, idem=str(uuid.uuid4())),
     "headcount": api(TEACHER, "PATCH", f"/courses/{C2}/checkin-windows/{w}", {"headcount": 3}), "spot": api(TEACHER, "POST", f"/courses/{C2}/checkin-windows/{w}/spot-check", {"n": 3})}
codes = {k: (v[0], v[1].get("code") if isinstance(v[1], dict) else None) for k, v in r.items()}
ok("TC-AS01-09/02-23/03-19/05-14/07-… lớp lưu trữ: mọi ghi → 409 COURSE_ARCHIVED", all(v == (409, "COURSE_ARCHIVED") for v in codes.values()), str(codes))
rd = {"windows": api(TEACHER, "GET", f"/courses/{C2}/sessions/{s}/checkin-windows")[0], "checkins": api(TEACHER, "GET", f"/courses/{C2}/sessions/{s}/checkins")[0], "display": display(C2, s, w)[0]}
ok("TC-AS06-10 lớp lưu trữ: đọc danh sách cửa sổ + lượt → 200", rd["windows"] == 200 and rd["checkins"] == 200, str(rd))
