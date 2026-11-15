# US-AS-03, 04 — điểm danh bằng mã / mã ngắn; cờ; mức tin cậy
from asl import *
import concurrent.futures as cf
S = st(); C, C2 = S["cid"], S["cid2"]
def one(sv, d, **kw): return checkin(sv, C, {"token": tokenof(d), **kw})
s, w, d = neww(C)
print("mẫu token (độ dài):", len(tokenof(d)), tokenof(d)[:6] + "…", "| code:", d["short_code"])
c, j, _ = one("sv.gioi", d)
r = dbrows(s, w)
ok("TC-AS03-01 SV điểm danh bằng token: 201 {checked_in_at, round:1}; 1 hàng WEB", c == 201 and set(j) == {"checked_in_at", "round"} and j["round"] == 1 and r[0][1] == "WEB", f"{c} {j} {r}")
c1, j1, _ = api("sv.kha", "POST", f"/courses/{C}/checkins", {"token": tokenof(d), "code": "ABC123"}); c2, j2, _ = api("sv.kha", "POST", f"/courses/{C}/checkins", {}); 
ok("TC-AS03-02a thân có cả token+code / thân rỗng → 422", c1 == 422 and c2 == 422, f"{c1} {c2}")
c3, j3, _ = api("sv.kha", "POST", f"/courses/{C}/checkins", {"token": tokenof(d), "student_id": uid("sv.gioi"), "user_id": uid("sv.gioi")})
r = dbrows(s, w); who = [x[0] for x in r]
ok("TC-AS03-02b student_id/user_id trong thân bị bỏ qua (hợp đồng: bỏ qua; thực tế 422 'unknown')", c3 == 201 and who == ["sv.gioi@edupilot.local", "sv.kha@edupilot.local"], f"{c3} {who}")
c3, j3, _ = one("sv.kha", d); r = dbrows(s, w)
ok("TC-AS03-02c thân chỉ có token: 201, hàng ghi cho người gọi", c3 == 201 and [x[0] for x in r] == ["sv.gioi@edupilot.local", "sv.kha@edupilot.local"], f"{c3} {[x[0] for x in r]}")
# 14 gửi lại / song song
c, j2, _ = one("sv.gioi", d)
ok("TC-AS03-14a gửi lại: 200 cùng checked_in_at/round", c == 200 and j2 == j, f"{c} {j2} vs {j}")
with cf.ThreadPoolExecutor(8) as ex: rr = list(ex.map(lambda _: one("sv.gioi", d)[0], range(8)))
n = sql(f"select count(*) from attendance_checkins where window_id='{w}' and student_id='{uid('sv.gioi')}'")[0][0]
ok("TC-AS03-14b 8 yêu cầu song song: toàn 200 hoặc 429, đúng 1 hàng", n == "1" and all(x in (200, 429) for x in rr), f"{rr} n={n}")
# 16 giống nhau
s16, w16, d16 = neww(C)
ca, ja, _ = one("sv.nguyco", d16, device=None) if False else one("sv.nguyco", d16)
k = Key(); cb, jb, _ = checkin("sv04", C, {"token": tokenof(d16), "device": k.dev("sv04", token=tokenof(d16))})
ok("TC-AS03-16 phản hồi hai SV (có / không chữ ký) cùng dạng; không flags/assurance/device/geo", ca == cb == 201 and set(ja) == set(jb) == {"checked_in_at", "round"} and len(ja["checked_in_at"].split(".")[0]) == len(jb["checked_in_at"].split(".")[0]) and ja["round"] == jb["round"], f"{ja} {jb}")
# 18 me/attendance
c, m, _ = api("sv.gioi", "GET", f"/courses/{C}/me/attendance"); items = m["attendance"]["sessions"]
row = [x for x in items if x.get("checked_in_at")]
txt = json.dumps(m)
import re
ok("TC-AS03-18 me/attendance: buổi đã điểm danh có checked_in_at; không cờ/mức/thiết bị", c == 200 and bool(row) and bool(row[0].get("checked_in_at")) and not re.search(r'flags|assurance|device|geo', txt), f"{row[:1]} keys={list(items[0]) if items else list(m)} n_checked={len(row)}")
# 04: cờ & mức tin cậy
time.sleep(62)
s4, w4, d4 = neww(C)
t = tokenof(d4)
k1 = Key(); k2 = Key()
c, j, _ = checkin("sv.gioi", C, {"token": t, "device": k1.dev("sv.gioi", token=t)}); ok("TC-AS04-01 WEB_KEY r‖s hợp lệ: 201", c == 201, str(c))
c, j, _ = checkin("sv.kha", C, {"token": t, "device": k2.dev("sv.kha", token=t, der=True)}); ok("TC-AS04-02 chữ ký DER hợp lệ: 201", c == 201, str(c))
c, j, _ = checkin("sv.nguyco", C, {"token": t}); ok("TC-AS04-05 không device: 201", c == 201)
bad = k1.dev("sv04", token=t); sg = bytearray(base64.b64decode(bad["signature"])); sg[5] ^= 1; bad["signature"] = b64(bytes(sg))
c, j, _ = checkin("sv04", C, {"token": t, "device": bad}); ok("TC-AS04-06a chữ ký sai: vẫn 201", c == 201, str(c))
c, j, _ = checkin("sv05", C, {"token": t, "device": {"kind": "WEB_KEY", "public_key": "rác", "signature": "rác"}}); ok("TC-AS04-06b public_key hỏng: 201 hoặc 422 (ghi nhận)", c in (201, 422), f"{c} {json.dumps(j)[:100]}")
rw = {x[0].split("@")[0]: (x[2], x[3]) for x in dbrows(s4, w4)}
ok("TC-AS04-01c sv.gioi assurance 2, không cờ tín hiệu (DEVICE_NEW cho phép: SV đã có khoá cũ từ lần chạy trước)", rw.get("sv.gioi", ("", ""))[0] == "2" and rw["sv.gioi"][1] in ("", "DEVICE_NEW"), str(rw.get("sv.gioi")))
ok("TC-AS04-02c sv.kha (DER) assurance 2", rw.get("sv.kha", ("", ""))[0] == "2", str(rw.get("sv.kha")))
ok("TC-AS04-05c không device: NO_DEVICE_KEY, mức 1", rw.get("sv.nguyco") == ("1", "NO_DEVICE_KEY"), str(rw.get("sv.nguyco")))
ok("TC-AS04-06c chữ ký sai: DEVICE_UNTRUSTED, mức 1", rw.get("sv04", ("", ""))[0] == "1" and "DEVICE_UNTRUSTED" in rw.get("sv04", ("", ""))[1], str(rw.get("sv04")))
ok("TC-AS04-06d public_key hỏng: có cờ (NO_DEVICE_KEY|DEVICE_UNTRUSTED), mức 1", c != 201 or (rw.get("sv05", ("", ""))[0] == "1" and rw["sv05"][1] != ""), str(rw.get("sv05")))
# 07: máy dùng chung
s7, w7, d7 = neww(C); t7 = tokenof(d7)
kk = Key()
checkin("sv.gioi", C, {"token": t7, "device": kk.dev("sv.gioi", token=t7)}); checkin("sv.kha", C, {"token": t7, "device": kk.dev("sv.kha", token=t7)})
rw = {x[0].split("@")[0]: x[3] for x in dbrows(s7, w7)}
ok("TC-AS04-07 cùng khoá hai SV: cả hai lượt DEVICE_SHARED (cờ gắn cả lượt trước)", "DEVICE_SHARED" in rw["sv.gioi"] and "DEVICE_SHARED" in rw["sv.kha"], str(rw))
ok("TC-AS04-07b DB: một device_id cho hai SV", sql(f"select count(distinct device_id) from attendance_checkins where window_id='{w7}'")[0][0] == "1")
# 08: đua
time.sleep(62)
okr = []
for _ in range(4):
    sx, wx, dx = neww(C); tx = tokenof(dx); k = Key()
    with cf.ThreadPoolExecutor(2) as ex: list(ex.map(lambda u: checkin(u, C, {"token": tx, "device": k.dev(u, token=tx)}), ["sv.gioi", "sv.kha"]))
    f = [x[3] for x in dbrows(sx, wx)]; okr.append(all("DEVICE_SHARED" in x for x in f) and len(f) == 2)
ok("TC-AS04-08 đua 4 vòng: luôn cả hai DEVICE_SHARED", all(okr), str(okr))
# 09: hai lớp khác nhau cùng khoá → không SHARED (C và C2 — cần C2 bật self_checkin)
policy(C2, rotate_seconds=5)
s9, w9, d9 = neww(C2); t9 = tokenof(d9); k = Key()
c, _, _ = checkin("sv.gioi", C2, {"token": t9, "device": k.dev("sv.gioi", token=t9)}); s9b, w9b, d9b = neww(C); t9b = tokenof(d9b)
c2_, _, _ = checkin("sv.kha", C, {"token": t9b, "device": k.dev("sv.kha", token=t9b)})
f = sql(f"select coalesce(nullif(array_to_string(flags,','),''),'-') from attendance_checkins where window_id='{w9b}'")[0][0]
ok("TC-AS04-09 cùng khoá ở hai lớp: không DEVICE_SHARED", c == 201 and c2_ == 201 and "DEVICE_SHARED" not in f, f"{c} {c2_} flags={f!r}")
# 10: DEVICE_NEW
sA, wA, dA = neww(C); tA = tokenof(dA); K1, K2 = Key(), Key(); u = "sv07"; api(u, "POST", "/courses/join", {"code": "QCKASTV"})
checkin(u, C, {"token": tA, "device": K1.dev(u, token=tA)})
sB, wB, dB = neww(C); tB = tokenof(dB); checkin(u, C, {"token": tB, "device": K2.dev(u, token=tB)})
sC, wC, dC = neww(C); tC = tokenof(dC); checkin(u, C, {"token": tC, "device": K2.dev(u, token=tC)})
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
