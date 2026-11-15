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

rd = lambda *x: subprocess.run(["docker", "exec", "ep-qc-redis-1", "redis-cli", *x], capture_output=True, text=True).stdout.strip()
time.sleep(61)
u = "sv.nguyco"; api(u, "POST", "/me/devices/challenge", None)
key = [k for k in rd("--scan", "--pattern", "*chal*").splitlines() if k.endswith(uid(u))][0]; rd("expire", key, "1"); time.sleep(2.5)
c, j, _ = api(u, "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": chain})
ok("TC-AS12-04b nonce hết hạn (EXPIRE 1 s) → 422 DEVICE_CHALLENGE_INVALID", c == 422 and "DEVICE_CHALLENGE_INVALID" in json.dumps(j), f"{c} {json.dumps(j)[:100]}")
def fixture(boot):
    k = Key(); der = base64.b64decode(k.pub); h = hashlib.sha256(der).digest()
    sql(f"insert into user_devices(key_sha256, public_key, kind, attested, boot_verified, security_level) values (decode('{h.hex()}','hex'), decode('{der.hex()}','hex'), 'ANDROID_KEY', true, {boot}, 'TRUSTED_ENVIRONMENT') on conflict do nothing") if False else subprocess.run(["docker", "compose", "-p", "ep-qc", "exec", "-T", "postgres", "psql", "-U", "edupilot", "-d", "edupilot", "-c", f"insert into user_devices(key_sha256, public_key, kind, attested, boot_verified, security_level) values (decode('{h.hex()}','hex'), decode('{der.hex()}','hex'), 'ANDROID_KEY', true, {boot}, 'TRUSTED_ENVIRONMENT')"], capture_output=True, text=True, cwd=os.path.expanduser("~/Documents/TA_Agent_qc5"))
    return k, base64.urlsafe_b64encode(h).decode().rstrip("=")
def adev(k, kid, u, t, **kw):
    sig = k.sign(f"edupilot-checkin-v1|{t}|{uid(u)}", der=True); return {"kind": "ANDROID_KEY", "key_id": kid, "signature": sig}


policy(C, rotate_seconds=5, late_after_min=10, require="APP")
kF2, idF2 = fixture("false"); s, w, d = neww(C); t = tokenof(d)
checkin("sv09", C, {"token": t, "device": adev(kF2, idF2, "sv09", t)}); R = fl(w, "sv09")
ok("TC-AS04-13c ANDROID_KEY boot_verified=false: không NOT_APP, có DEVICE_UNTRUSTED (chạy lại với khoá mới)", R and "NOT_APP" not in R[0] and "DEVICE_UNTRUSTED" in R[0], str(R))
policy(C, rotate_seconds=5, late_after_min=10)
