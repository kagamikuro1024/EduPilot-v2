# US-AS-10 (vị trí / mạng) + US-AS-12 (thiết bị: nonce, mức 3 bằng fixture)
from asl import *
import math, re, subprocess
S = st(); C, C2 = S["cid"], S["cid3"]
LAT, LNG = 10.7727, 106.6602
def dest(d):  # điểm cách tâm d mét về phía bắc (haversine R=6371000)
    return LAT + math.degrees(d / 6371000.0), LNG
def hav(a, b):
    p1, p2 = math.radians(a[0]), math.radians(b[0]); dl = math.radians(b[1] - a[1]); dp = p2 - p1
    h = math.sin(dp / 2) ** 2 + math.cos(p1) * math.cos(p2) * math.sin(dl / 2) ** 2
    return 2 * 6371000 * math.asin(math.sqrt(h))
def geo(d, acc): la, ln = dest(d); return {"lat": la, "lng": ln, "accuracy_m": acc}
def fl(w, u): r = sql(f"select array_to_string(flags,','), assurance, geo_distance_m, geo_accuracy_m, in_campus from attendance_checkins where window_id='{w}' and student_id='{uid(u)}'"); return r[0] if r else None
# ---- AS10-01: lớp C2 không khai geofence: geo bị bỏ qua
policy(C2, rotate_seconds=5); s, w, d = neww(C2)
c, _, _ = checkin("sv.gioi", C2, {"token": tokenof(d), "geo": geo(5000, 10)}); r = fl(w, "sv.gioi")
ok("TC-AS10-01 lớp không khai geofence: geo bị bỏ qua (không geo_*, in_campus NULL, không GEO_*)", c == 201 and r[2] == "" and r[3] == "" and r[4] == "" and "GEO" not in r[0] and "OFF_CAMPUS" not in r[0], str(r))
# ---- geofence 200 m
time.sleep(1)
policy(C, rotate_seconds=5, late_after_min=10, geofence={"lat": LAT, "lng": LNG, "radius_m": 200})
s, w, d = neww(C); t = tokenof(d)
cases = [("sv.gioi", geo(50, 20)), ("sv.kha", geo(260, 50)), ("sv.nguyco", geo(249.5, 50)), ("sv04", geo(10, 101)), ("sv05", {"denied": True}), ("sv06", None)]
for u, g in cases: checkin(u, C, {"token": t, **({"geo": g} if g else {})})
R = {u: fl(w, u) for u, _ in cases}
ok("TC-AS10-03 D(50) acc20: geo_distance_m≈50, geo_accuracy_m=20, không cờ", R["sv.gioi"] and abs(int(R["sv.gioi"][2]) - 50) <= 1 and R["sv.gioi"][3] == "20" and "GEO" not in R["sv.gioi"][0], str(R["sv.gioi"]))
ok("TC-AS10-04 D(260) acc50: GEO_OUT, vẫn 201 (đã ghi)", "GEO_OUT" in R["sv.kha"][0], str(R["sv.kha"]))
ok("TC-AS10-05 biên D(249.5) acc50: KHÔNG GEO_OUT", "GEO_OUT" not in R["sv.nguyco"][0], str(R["sv.nguyco"]))
ok("TC-AS10-06 acc=101: GEO_WEAK", "GEO_WEAK" in R["sv04"][0], str(R["sv04"]))
ok("TC-AS10-07 denied=true và không có geo: GEO_DENIED (cả hai)", "GEO_DENIED" in R["sv05"][0] and "GEO_DENIED" in R["sv06"][0], f"{R['sv05']} {R['sv06']}")
ok("TC-AS10-04b lượt cờ GEO có assurance 1", R["sv.kha"][1] == "1", R["sv.kha"][1])
# acc=100 không cờ + 18 haversine (20 điểm)
s2, w2, d2 = neww(C); t2 = tokenof(d2)
checkin("sv.gioi", C, {"token": t2, "geo": geo(30, 100)}); r = fl(w2, "sv.gioi")
ok("TC-AS10-06b acc=100: không GEO_WEAK", "GEO_WEAK" not in r[0], str(r))
errs = []
for i, dd in enumerate([100, 150, 199, 201, 250, 300, 500, 1000]):
    if i > 7: break
    sx, wx, dx = neww(C); u = SV[i % 6]; checkin(u, C, {"token": tokenof(dx), "geo": geo(dd, 5)}); r = fl(wx, u); errs.append(abs(int(r[2]) - dd))
    if i == 5: time.sleep(31)  # tránh trần 10 lượt/phút
ok("TC-AS10-18 khoảng cách haversine: sai số ≤ 1 m so với tham chiếu (8 điểm)", max(errs) <= 1, str(errs))
c, j, _ = checkin("sv.kha", C, {"token": t2, "geo": {"lat": 91, "lng": 0}}); c2, _, _ = checkin("sv.kha", C, {"token": t2, "geo": {"lat": 0, "lng": -181}}); c3, _, _ = checkin("sv.kha", C, {"token": t2, "geo": {"lat": 1, "lng": 1, "accuracy_m": -1}}); c4, _, _ = checkin("sv.kha", C, {"token": t2, "geo": {"lat": "x", "lng": 1}})
ok("TC-AS10-14 lat=91 / lng=-181 / acc=-1 / lat chuỗi → 422", [c, c2, c3, c4] == [422] * 4, str([c, c2, c3, c4]))
# 10-16 Staff thấy; SV không
c, j = rows(C, s); it = [x for x in j["items"] if x["student"]["full_name"]]; ok("TC-AS10-16a Staff thấy geo_distance_m & in_campus", c == 200 and any(x.get("geo_distance_m") is not None for x in it), json.dumps(it[0], ensure_ascii=False)[:250])
# 10-17 tắt geofence giữa buổi
time.sleep(20)
s3, w3, d3 = neww(C); t3 = tokenof(d3)
checkin("sv.gioi", C, {"token": t3, "geo": geo(10, 10)})
policy(C, rotate_seconds=5, late_after_min=10)  # tắt geofence
checkin("sv.kha", C, {"token": t3, "geo": geo(10, 10)})
a, b = fl(w3, "sv.gioi"), fl(w3, "sv.kha")
ok("TC-AS10-17 tắt geofence giữa buổi: lượt trước giữ geo_*, lượt sau không lưu", a[2] != "" and b[2] == "" and b[3] == "" and b[4] == "", f"{a} / {b}")
# ---- CIDR
policy(C, rotate_seconds=5, late_after_min=10, campus_cidrs=["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.0/8"])
s4, w4, d4 = neww(C); checkin("sv.gioi", C, {"token": tokenof(d4)}); a = fl(w4, "sv.gioi")
policy(C, rotate_seconds=5, late_after_min=10, campus_cidrs=["203.0.113.0/24"])
s5, w5, d5 = neww(C); checkin("sv.gioi", C, {"token": tokenof(d5)}); b = fl(w5, "sv.gioi")
ok("TC-AS10-09 mạng trường: IP nội bộ (bridge Docker) → in_campus=true, không OFF_CAMPUS; CIDR khác → false + OFF_CAMPUS", a[4] == "t" and "OFF_CAMPUS" not in a[0] and b[4] == "f" and "OFF_CAMPUS" in b[0], f"{a} / {b}")
time.sleep(5)
policy(C, rotate_seconds=5, late_after_min=10, geofence={"lat": LAT, "lng": LNG, "radius_m": 200}, campus_cidrs=["203.0.113.0/24"])
s6, w6, d6 = neww(C); checkin("sv.gioi", C, {"token": tokenof(d6), "geo": geo(1000, 10)}); r = fl(w6, "sv.gioi")
ok("TC-AS10-10 ngoài cả hai: GEO_OUT + OFF_CAMPUS", "GEO_OUT" in r[0] and "OFF_CAMPUS" in r[0], str(r))
# 10-08 / 12 không toạ độ ở DB/log
bad_cols = sql("select table_name||'.'||column_name from information_schema.columns where table_schema='public' and table_name in ('attendance_checkins','user_devices','checkin_windows') and (column_name ~* 'lat|lng|lon|coord|ip_|_ip|ip$|hostname|device_name|imei|serial')")
ok("TC-AS10-08a bảng điểm danh không có cột toạ độ / IP / tên máy", not bad_cols, str(bad_cols))
logs = subprocess.run("docker logs ep-r-gateway-1 2>&1 | grep -c -E '10\\.77|106\\.66|\"lat\"|\"lng\"'", shell=True, capture_output=True, text=True).stdout.strip()
ok("TC-AS10-08b log gateway không chứa toạ độ đã gửi", logs == "0", logs)
policy(C, rotate_seconds=5, late_after_min=10)
# ============ US-AS-12 ============
rd = lambda *a: subprocess.run(["docker", "exec", "ep-r-redis-1", "redis-cli", *a], capture_output=True, text=True).stdout.strip()
time.sleep(61)
c, j, _ = api("sv.gioi", "POST", "/me/devices/challenge", None); keys = rd("--scan", "--pattern", "*chal*"); ttl = rd("ttl", keys.splitlines()[0]) if keys else "-"
ok("TC-AS12-01 challenge: 200 {nonce}; Redis có khoá, TTL ≤ 300", c == 200 and list(j) == ["nonce"] and keys and 0 < int(ttl) <= 300, f"{c} len(nonce)={len(j.get('nonce',''))} ttl={ttl} key={'…'+keys[-8:] if keys else ''}")
n1 = j["nonce"]; c, j2, _ = api("sv.gioi", "POST", "/me/devices/challenge", None); ok("TC-AS12-01b gọi lại: nonce mới", c == 200 and j2["nonce"] != n1)
rr = [api("sv.gioi", "POST", "/me/devices/challenge", None)[0] for _ in range(5)]
ok("TC-AS12-02 6 lần/phút: có 429 (đã 2 lần + 5 = 7)", 429 in rr, str(rr))
time.sleep(61)
api("sv.kha", "POST", "/me/devices/challenge", None)
c, j, _ = api("sv.kha", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": ["AAAA"]})
ok("TC-AS12-03 chuỗi rác → 422 ATTESTATION_FAILED", c == 422 and "ATTESTATION_FAILED" in json.dumps(j), f"{c} {json.dumps(j)[:100]}")
c, j, _ = api("sv.kha", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": ["AAAA"]})
ok("TC-AS12-03b nonce đã bị xoá (GETDEL): lần hai → 422 DEVICE_CHALLENGE_INVALID", c == 422 and "DEVICE_CHALLENGE_INVALID" in json.dumps(j), f"{c} {json.dumps(j)[:100]}")
c, j, _ = api("sv.nguyco", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": ["AAAA"]}); ok("TC-AS12-04 chưa xin nonce → 422 DEVICE_CHALLENGE_INVALID", c == 422 and "DEVICE_CHALLENGE_INVALID" in json.dumps(j), f"{c}")
r09 = {u: (api(u, "POST", "/me/devices/challenge", None)[0], api(u, "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": "AA", "attestation_chain": ["AA"]})[0]) for u in (TA, TEACHER, ADMIN)}
c0 = (raw("POST", "/me/devices/challenge", {})[0])
ok("TC-AS12-09 TA/TEACHER/ADMIN → 403; không JWT → 401", all(v == (403, 403) for v in r09.values()) and c0 == 401, f"{r09} {c0}")
c, j, _ = api("sv.04" if False else "sv04", "POST", "/me/devices/challenge", None); big = ["A" * 4000] * 11
c2, j2, _ = api("sv04", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": big})
ok("TC-AS12-08 chuỗi vượt maxItems (11 phần tử) → 413/422, không treo", c2 in (413, 422), f"{c2}")
# ---- mức 3 bằng fixture (Q-S108-2): khoá ANDROID_KEY attested giả
def fixture(boot):
    k = Key(); der = base64.b64decode(k.pub); h = hashlib.sha256(der).digest()
    sql(f"insert into user_devices(key_sha256, public_key, kind, attested, boot_verified, security_level) values (decode('{h.hex()}','hex'), decode('{der.hex()}','hex'), 'ANDROID_KEY', true, {boot}, 'TRUSTED_ENVIRONMENT') on conflict do nothing") if False else subprocess.run(["docker", "exec", "-i", "ep-r-postgres-1", "psql", "-U", "edupilot", "-d", "edupilot", "-c", f"insert into user_devices(key_sha256, public_key, kind, attested, boot_verified, security_level) values (decode('{h.hex()}','hex'), decode('{der.hex()}','hex'), 'ANDROID_KEY', true, {boot}, 'TRUSTED_ENVIRONMENT')"], capture_output=True, text=True)
    return k, base64.urlsafe_b64encode(h).decode().rstrip("=")
def adev(k, kid, u, t, **kw):
    sig = k.sign(f"edupilot-checkin-v1|{t}|{uid(u)}", der=True); return {"kind": "ANDROID_KEY", "key_id": kid, "signature": sig}
time.sleep(15)
policy(C, rotate_seconds=5, late_after_min=10)
kT, idT = fixture("true"); kF, idF = fixture("false"); kM, idM = fixture("true")
s, w, d = neww(C); t = tokenof(d)
c1, j1, _ = checkin("sv.gioi", C, {"token": t, "device": adev(kT, idT, "sv.gioi", t), "app": {"version": "1.0.0"}})
c2, j2, _ = checkin("sv.kha", C, {"token": t, "device": adev(kF, idF, "sv.kha", t)})
c3, j3, _ = checkin("sv.nguyco", C, {"token": t, "device": adev(kM, idM, "sv.nguyco", t), "geo": {"lat": LAT, "lng": LNG, "accuracy_m": 10, "mock": True}})
c4, j4, _ = checkin("sv04", C, {"token": t, "device": {"kind": "ANDROID_KEY", "key_id": "Zm9vYmFy", "signature": "AAAA"}})
R = {u: fl(w, u) for u in ("sv.gioi", "sv.kha", "sv.nguyco", "sv04")}
ok("TC-AS12-12 ANDROID_KEY attested+boot_verified, chữ ký đúng: assurance=3, không cờ", c1 == 201 and R["sv.gioi"][1] == "3" and R["sv.gioi"][0] == "", f"{c1} {R['sv.gioi']} {json.dumps(j1)[:100]}")
ok("TC-AS12-13 boot_verified=false: mức 2 + DEVICE_UNTRUSTED", c2 == 201 and R["sv.kha"][1] == "2" and "DEVICE_UNTRUSTED" in R["sv.kha"][0], f"{c2} {R['sv.kha']}")
ok("TC-AS12-14 geo.mock=true: GEO_MOCK, assurance ≤ 2, vẫn 201", c3 == 201 and "GEO_MOCK" in R["sv.nguyco"][0] and int(R["sv.nguyco"][1]) <= 2, f"{c3} {R['sv.nguyco']}")
ok("TC-AS12-15 key_id chưa đăng ký: DEVICE_UNTRUSTED, mức 1, 201", c4 == 201 and R["sv04"][1] == "1" and "DEVICE_UNTRUSTED" in R["sv04"][0], f"{c4} {R['sv04']}")
# NOT_APP (P-108-6) lớp require=APP
policy(C, rotate_seconds=5, late_after_min=10, require="APP")
s, w, d = neww(C); t = tokenof(d); kk = Key()
checkin("sv.gioi", C, {"token": t, "device": kk.dev("sv.gioi", token=t)})                              # (a) WEB_KEY hợp lệ
checkin("sv.kha", C, {"token": t, "device": adev(kT, idT, "sv.kha", t)})                              # (b) ANDROID_KEY attested boot ok
checkin("sv.nguyco", C, {"token": t, "device": adev(kF, idF, "sv.nguyco", t)})                        # (c) boot_verified=false
checkin("sv04", C, {"token": t, "device": {"kind": "ANDROID_KEY", "key_id": "Zm9vYmFy", "signature": "AAAA"}})  # (d) key_id lạ
checkin("sv05", C, {"token": t})                                                                      # không device
R = {u: fl(w, u) for u in ("sv.gioi", "sv.kha", "sv.nguyco", "sv04", "sv05")}
ok("TC-AS04-13a require=APP + WEB_KEY hợp lệ: NOT_APP, mức 2", "NOT_APP" in R["sv.gioi"][0] and R["sv.gioi"][1] == "2", str(R["sv.gioi"]))
ok("TC-AS04-13b require=APP + ANDROID_KEY attested + chữ ký đúng: không NOT_APP, mức 3", "NOT_APP" not in R["sv.kha"][0] and R["sv.kha"][1] == "3", str(R["sv.kha"]))
ok("TC-AS04-13c ANDROID_KEY boot_verified=false: không NOT_APP (có DEVICE_UNTRUSTED)", "NOT_APP" not in R["sv.nguyco"][0] and "DEVICE_UNTRUSTED" in R["sv.nguyco"][0], str(R["sv.nguyco"]))
ok("TC-AS04-13d key_id chưa đăng ký / không device: NOT_APP", "NOT_APP" in R["sv04"][0] and "NOT_APP" in R["sv05"][0], f"{R['sv04']} / {R['sv05']}")
policy(C, rotate_seconds=5, late_after_min=10)
# 12-17 user_devices
ok("TC-AS12-17 user_devices: label luôn NULL; không user_id", sql("select count(*) from user_devices where label is not null")[0][0] == "0" and not sql("select 1 from information_schema.columns where table_name='user_devices' and column_name='user_id'"), "")
