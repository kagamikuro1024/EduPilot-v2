# US-AS-03, 04 — điểm danh bằng mã / mã ngắn; cờ; mức tin cậy
from asl import *
import re
s7 = sql(f"select session_id from checkin_windows where course_id='{st()['cid']}' order by opened_at desc limit 1")[0][0]
t7 = ''
import concurrent.futures as cf
S = st(); C, C2 = S["cid"], S["cid2"]
def one(sv, d, **kw): return checkin(sv, C, {"token": tokenof(d), **kw})
# 09: hai lớp khác nhau cùng khoá → không SHARED (C và C2 — cần C2 bật self_checkin)
policy(C2, rotate_seconds=5)
s9, w9, d9 = neww(C2); t9 = tokenof(d9); k = Key()
c, _, _ = checkin("sv.gioi", C2, {"token": t9, "device": k.dev("sv.gioi", token=t9)}); s9b, w9b, d9b = neww(C); t9b = tokenof(d9b)
c2_, _, _ = checkin("sv.kha", C, {"token": t9b, "device": k.dev("sv.kha", token=t9b)})
f = sql(f"select coalesce(nullif(array_to_string(flags,','),''),'-') from attendance_checkins where window_id='{w9b}'")[0][0]
ok("TC-AS04-09 cùng khoá ở hai lớp: không DEVICE_SHARED", c == 201 and c2_ == 201 and "DEVICE_SHARED" not in f, f"{c} {c2_} flags={f!r}")
# 10: DEVICE_NEW
sA, wA, dA = neww(C); tA = tokenof(dA); K1, K2 = Key(), Key(); u = "sv07"; api(u, "POST", "/courses/join", {"code": "QCKASTV"})
print("K1", checkin(u, C, {"token": tA, "device": K1.dev(u, token=tA)})[:2])
sB, wB, dB = neww(C); tB = tokenof(dB); print("K2", checkin(u, C, {"token": tB, "device": K2.dev(u, token=tB)})[:2])
sC, wC, dC = neww(C); tC = tokenof(dC); print("K2b", checkin(u, C, {"token": tC, "device": K2.dev(u, token=tC)})[:2])
fa, fb, fc = (sql(f"select coalesce(nullif(array_to_string(flags,','),''),'-') from attendance_checkins where window_id='{x}' and student_id='{uid(u)}'")[0][0] for x in (wA, wB, wC))
ok("TC-AS04-10 DEVICE_NEW: K1 lần đầu không cờ; K2 đầu: DEVICE_NEW; K2 sau: không", "DEVICE_NEW" not in fa and "DEVICE_NEW" in fb and "DEVICE_NEW" not in fc, f"{fa!r} {fb!r} {fc!r}")
# 17 SV không thấy cờ
txt = ""
for p in (f"/courses/{C}/me/attendance", f"/courses/{C}", "/me/today"): txt += json.dumps(api("sv.gioi", "GET", p)[1])
import re
ok("TC-AS04-17a route SV: không flags/assurance/device_id/geo_/in_campus", not re.search(r'"flags"|"assurance"|device_id|geo_(?!check)|in_campus|DEVICE_SHARED', txt), "")
cs = [api(u, "GET", f"/courses/{C}/sessions/{s7}/checkins")[0] for u in ("sv.gioi", ADMIN)] + [api(u, "GET", f"/courses/{C}/sessions/{s7}/checkins")[0] for u in (TA, TEACHER)]
ok("TC-AS04-17b GET checkins: STUDENT/ADMIN 403; TA/TEACHER 200", cs == [403, 403, 200, 200], str(cs))
# 15: nhãn cờ
s7 = sql(f"select session_id from attendance_checkins where course_id='{C}' and cardinality(flags)>0 order by created_at desc limit 1")[0][0]
c, j = rows(C, s7); lab = {f["code"]: f["label"] for x in j["items"] for f in x["flags"]}
ok("TC-AS04-15 nhãn cờ tiếng Việt, không từ kỹ thuật", c == 200 and "DEVICE_SHARED" in lab and not re.search(r"token|HMAC|attest|nonce", " ".join(lab.values()), re.I), str(lab))
# 03-19 lỗi
c, j, _ = checkin("sv.gioi", C2 if False else C, {"token": t7, "geo": {"lat": "10.7"}}); ok("TC-AS03-19a geo.lat chuỗi → 422", c == 422, str(c))
c, j, _ = checkin("sv.gioi", C, {"token": t7, "device": {"kind": "WEB_KEY", "public_key": "!!", "signature": "@@@"}}); ok("TC-AS03-19b signature không phải base64: 422 hoặc 200 (ghi nhận)", c in (200, 422), f"{c} {json.dumps(j)[:90]}")
c, j, _ = api("sv.gioi", "POST", f"/courses/{S['cid2']}/checkins", {"token": "x"}) ; print("lớp C2 bật:", c)
