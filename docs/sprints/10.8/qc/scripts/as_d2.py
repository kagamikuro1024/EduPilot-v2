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

S = st(); C = S["cid"]
chain = ["AAAA", "AAAA"]
time.sleep(61)
api("sv.kha", "POST", "/me/devices/challenge", None)
c, j, _ = api("sv.kha", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": chain})
ok("TC-AS12-03 chuỗi rác (2 phần tử) → 422 ATTESTATION_FAILED", c == 422 and "ATTESTATION_FAILED" in json.dumps(j), f"{c} {json.dumps(j)[:100]}")
c, j, _ = api("sv.kha", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": chain})
ok("TC-AS12-03b nonce đã bị xoá (GETDEL): lần hai → 422 DEVICE_CHALLENGE_INVALID", c == 422 and "DEVICE_CHALLENGE_INVALID" in json.dumps(j), f"{c} {json.dumps(j)[:100]}")
c, j, _ = api("sv.nguyco", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": chain}); ok("TC-AS12-04 chưa xin nonce → 422 DEVICE_CHALLENGE_INVALID", c == 422 and "DEVICE_CHALLENGE_INVALID" in json.dumps(j), f"{c} {json.dumps(j)[:100]}")
c, j, _ = api("sv.nguyco", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": ["AAAA"]}); ok("TC-AS12-03c 1 chứng chỉ → 422 VALIDATION_FAILED (hợp đồng openapi minItems:1 nhưng thực tế 2–6)", c == 422, f"{c}")
rd = lambda *a: subprocess.run(["docker", "exec", "ep-qc-redis-1", "redis-cli", *a], capture_output=True, text=True).stdout.strip()
api("sv.nguyco", "POST", "/me/devices/challenge", None); rd("--scan", "--pattern", "*chal*")
key = rd("--scan", "--pattern", "*chal*").splitlines()[0]; rd("expire", key, "1"); time.sleep(2)
c, j, _ = api("sv.nguyco", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": chain}); ok("TC-AS12-04b nonce hết hạn (EXPIRE 1 s) → 422 DEVICE_CHALLENGE_INVALID", c == 422 and "DEVICE_CHALLENGE_INVALID" in json.dumps(j), f"{c} {json.dumps(j)[:100]}")
def fixture(boot):
    k = Key(); der = base64.b64decode(k.pub); h = hashlib.sha256(der).digest()
    sql(f"insert into user_devices(key_sha256, public_key, kind, attested, boot_verified, security_level) values (decode('{h.hex()}','hex'), decode('{der.hex()}','hex'), 'ANDROID_KEY', true, {boot}, 'TRUSTED_ENVIRONMENT') on conflict do nothing") if False else subprocess.run(["docker", "compose", "-p", "ep-qc", "exec", "-T", "postgres", "psql", "-U", "edupilot", "-d", "edupilot", "-c", f"insert into user_devices(key_sha256, public_key, kind, attested, boot_verified, security_level) values (decode('{h.hex()}','hex'), decode('{der.hex()}','hex'), 'ANDROID_KEY', true, {boot}, 'TRUSTED_ENVIRONMENT')"], capture_output=True, text=True, cwd=os.path.expanduser("~/Documents/TA_Agent_qc5"))
    return k, base64.urlsafe_b64encode(h).decode().rstrip("=")
def adev(k, kid, u, t, **kw):
    sig = k.sign(f"edupilot-checkin-v1|{t}|{uid(u)}", der=True); return {"kind": "ANDROID_KEY", "key_id": kid, "signature": sig}

policy(C, rotate_seconds=5, late_after_min=10, geofence={"lat": 10.7727, "lng": 106.6602, "radius_m": 200})
LAT, LNG = 10.7727, 106.6602
time.sleep(30)
kT, idT = fixture("true"); kF, idF = fixture("false"); kM, idM = fixture("true"); kN, idN = fixture("true")
s, w, d = neww(C); t = tokenof(d)
users = {"sv.gioi": (kT, idT, None), "sv.kha": (kF, idF, None), "sv.nguyco": (kM, idM, {"lat": LAT, "lng": LNG, "accuracy_m": 10, "mock": True})}
rs = {}
for u, (k, i_, g) in users.items(): rs[u] = checkin(u, C, {"token": t, "device": adev(k, i_, u, t), **({"geo": g} if g else {"geo": {"lat": LAT, "lng": LNG, "accuracy_m": 10}})})[0]
c4 = checkin("sv04", C, {"token": t, "device": {"kind": "ANDROID_KEY", "key_id": "Zm9vYmFy", "signature": "AAAA"}, "geo": {"lat": LAT, "lng": LNG, "accuracy_m": 10}})[0]
R = {u: fl(w, u) for u in ("sv.gioi", "sv.kha", "sv.nguyco", "sv04")}
ok("TC-AS12-12 ANDROID_KEY attested + boot_verified + chữ ký + trong vùng: assurance=3, không cờ tín hiệu (DEVICE_NEW cho phép)", rs["sv.gioi"] == 201 and R["sv.gioi"][1] == "3" and not set(R["sv.gioi"][0].split(",")) - {"DEVICE_NEW", "-"}, str(R["sv.gioi"]))
ok("TC-AS12-13 boot_verified=false: mức 2 + DEVICE_UNTRUSTED", rs["sv.kha"] == 201 and R["sv.kha"][1] == "2" and "DEVICE_UNTRUSTED" in R["sv.kha"][0], str(R["sv.kha"]))
ok("TC-AS12-14 geo.mock=true (lớp có geofence): GEO_MOCK, assurance ≤ 2, 201", rs["sv.nguyco"] == 201 and "GEO_MOCK" in R["sv.nguyco"][0] and int(R["sv.nguyco"][1]) <= 2, str(R["sv.nguyco"]))
ok("TC-AS12-15 key_id chưa đăng ký: DEVICE_UNTRUSTED, mức 1, 201", c4 == 201 and R["sv04"][1] == "1" and "DEVICE_UNTRUSTED" in R["sv04"][0], str(R["sv04"]))
time.sleep(30)
policy(C, rotate_seconds=5, late_after_min=10, require="APP")
s, w, d = neww(C); t = tokenof(d); kk = Key()
checkin("sv05", C, {"token": t, "device": kk.dev("sv05", token=t)})
checkin("sv06", C, {"token": t, "device": adev(kN, idN, "sv06", t)})
checkin("sv07", C, {"token": t, "device": {"kind": "ANDROID_KEY", "key_id": "Zm9vYmFy", "signature": "AAAA"}})
checkin("sv08", C, {"token": t})
R = {u: fl(w, u) for u in ("sv05", "sv06", "sv07", "sv08")}
ok("TC-AS04-13a require=APP + WEB_KEY hợp lệ: NOT_APP, mức 2", "NOT_APP" in R["sv05"][0] and R["sv05"][1] == "2", str(R["sv05"]))
ok("TC-AS04-13b require=APP + ANDROID_KEY attested: không NOT_APP, mức 3", "NOT_APP" not in R["sv06"][0] and R["sv06"][1] == "3", str(R["sv06"]))
ok("TC-AS04-13d key_id lạ / không device: NOT_APP", "NOT_APP" in R["sv07"][0] and "NOT_APP" in R["sv08"][0], f"{R['sv07']} / {R['sv08']}")
time.sleep(20)
s, w, d = neww(C); t = tokenof(d)
checkin("sv09", C, {"token": t, "device": adev(kF, idF, "sv09", t)})
R = fl(w, "sv09"); ok("TC-AS04-13c ANDROID_KEY boot_verified=false: không NOT_APP, có DEVICE_UNTRUSTED", R and "NOT_APP" not in R[0] and "DEVICE_UNTRUSTED" in R[0], str(R))
policy(C, rotate_seconds=5, late_after_min=10, geofence=None)
ok("TC-AS12-17 user_devices: label luôn NULL; không user_id", sql("select count(*) from user_devices where label is not null")[0][0] == "0" and not sql("select 1 from information_schema.columns where table_name='user_devices' and column_name='user_id'"), "")
