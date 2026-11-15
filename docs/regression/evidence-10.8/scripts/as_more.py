# smoke r10.8: ca API / DB / DIFF / LOG còn lại (xem smoke-10.8.md)
from asl import *
import re, subprocess, math, datetime
from cryptography import x509
from cryptography.x509.oid import NameOID
R = os.path.abspath(os.path.join(os.path.dirname(__file__), "../../../.."))
S = st(); C = S["cid"]
sh = lambda c: subprocess.run(c, shell=True, capture_output=True, text=True, cwd=R)
LAT, LNG = 10.7727, 106.6602
def dest(d): return LAT + math.degrees(d / 6371000.0), LNG
def geo(d, acc): la, ln = dest(d); return {"lat": la, "lng": ln, "accuracy_m": acc}
def fl(w, u):
    r = sql(f"select coalesce(nullif(array_to_string(flags,','),''),'-'), assurance, channel from attendance_checkins where window_id='{w}' and student_id='{uid(u)}'"); return r[0] if r else None
policy(C, rotate_seconds=20, late_after_min=10, require="WEB", geofence=None, campus_cidrs=[])
# ---------- DB / DIFF ----------
oa = open(R + "/backend-go/api/openapi.yaml").read()
i = oa.index("operationId: listCheckinWindows"); seg = oa[i:i + 6000]
cons = {x[0]: x[1] for x in sql("select conname, pg_get_constraintdef(oid) from pg_constraint where conrelid='checkin_windows'::regclass")}
ok("TC-AS02-14 openapi list cửa sổ maxItems 3; DB CHECK round 1..3 + UNIQUE (course,session,round)", "maxItems: 3" in seg and "<= 3" in cons.get("checkin_windows_round_chk", "") and "course_id, session_id, round" in cons.get("checkin_windows_round_key", ""), f"{cons.get('checkin_windows_round_chk')} | {cons.get('checkin_windows_round_key')}")
cols = [x[0] for x in sql("select column_name from information_schema.columns where table_name='user_devices'")]
ok("TC-AS04-04 user_devices: không user_id, không bảng liên kết; label luôn NULL (CHECK)", "user_id" not in cols and not sql("select 1 from information_schema.columns where table_name in (select table_name from information_schema.tables where table_schema='public' and table_name ilike '%device%' and table_name<>'user_devices') and column_name in ('user_id','device_id') limit 1") and "label IS NULL" in sql("select pg_get_constraintdef(oid) from pg_constraint where conname='user_devices_label_chk'")[0][0], str(cols))
idx = {x[0]: x[1] for x in sql("select indexname, indexdef from pg_indexes where tablename='attendance_checkins'")}
cc = {x[0]: x[1] for x in sql("select conname, pg_get_constraintdef(oid) from pg_constraint where conrelid='attendance_checkins'::regclass")}
allc = [x[0] for x in sql("select column_name from information_schema.columns where table_name='attendance_checkins'")]
flags_codes = re.findall(r"'([A-Z_]+)'", cc.get("attendance_checkins_flags_chk", ""))
ok("TC-AS04-18 attendance_checkins: CHECK 12 mã cờ, 3 chỉ mục, UNIQUE (course,window,student), không cột toạ độ/IP", len(set(flags_codes)) == 12 and "(course_id, session_id, created_at, id)" in idx.get("attendance_checkins_session_idx", "") and "device_id IS NOT NULL" in idx.get("attendance_checkins_device_idx", "") and "(course_id, student_id, created_at)" in idx.get("attendance_checkins_student_idx", "") and "course_id, window_id, student_id" in idx.get("attendance_checkins_natural_key", "") and not [c for c in allc if re.search(r"lat|lng|lon|coord|^ip|_ip|ip$", c)], f"{len(set(flags_codes))} mã; cột={allc}")
pr = open(R + "/docs/PRODUCTION_READINESS.md").read()
ok("TC-AS13-06 PRODUCTION_READINESS §3 ghi job xoá tài khoản phải gọi attendance.ForgetStudent", "ForgetStudent" in pr, str([l[:140] for l in pr.splitlines() if "ForgetStudent" in l][:2]))
ok("TC-AS13-10 PRODUCTION_READINESS §1.1 có dòng retention cho lượt thô, thiết bị, cửa sổ (30 ngày / không dữ liệu cá nhân)", all(re.search(p, pr, re.I) for p in (r"attendance_checkins", r"user_devices", r"checkin_windows")) and "30 ngày" in pr, str([l[:150] for l in pr.splitlines() if re.search("checkin_windows|user_devices", l)][:3]))
gosrc = sh("grep -rn 'TOKEN_EXPIRED\\|TOKEN_INVALID' backend-go/internal/attendance backend-go/internal/httpapi/attendancehttp --include=*.go | grep -v _test | grep -v 'CHECKIN_TOKEN_' | head").stdout.strip()
ok("TC-AS03-25 openapi có CHECKIN_TOKEN_INVALID / CHECKIN_TOKEN_EXPIRED; route điểm danh không dùng TOKEN_EXPIRED/TOKEN_INVALID (xác thực)", "CHECKIN_TOKEN_INVALID" in oa and "CHECKIN_TOKEN_EXPIRED" in oa and not gosrc, gosrc[:200])
# ---------- API ----------
# AS04-03: mã ngắn ký "code:"
s, w, d = neww(C); code = d["short_code"].replace(" ", "")
K1, K2 = Key(), Key()
u1, u2 = "sv04", "sv05"
c1 = checkin(u1, C, {"code": code, "device": K1.dev(u1, code=code)})[0]
c2 = checkin(u2, C, {"code": code, "device": {**K2.dev(u2, token="whatever"), }})[0]
f1, f2 = fl(w, u1), fl(w, u2)
ok("TC-AS04-03 mã ngắn: ký code:CODE → 201 assurance 2; ký nhầm thông điệp token → DEVICE_UNTRUSTED", c1 == c2 == 201 and f1[1] == "2" and "DEVICE_UNTRUSTED" not in f1[0] and "DEVICE_UNTRUSTED" in f2[0] and f1[2] == "CODE", f"{c1} {f1} | {c2} {f2}")
# AS04-12 / 14 / 16 / 12-11 / 03-17 bảng ca (hình học + mạng)
policy(C, rotate_seconds=20, late_after_min=10, require="WEB", geofence={"lat": LAT, "lng": LNG, "radius_m": 200}, campus_cidrs=["203.0.113.0/24"])
s, w, d = neww(C); t = tokenof(d)
Ks = {u: Key() for u in ("sv.gioi", "sv.kha", "sv.nguyco", "sv04", "sv05", "sv06", "sv07", "sv08", "sv09", "sv10", "sv11", "sv12")}
case = {}
def go(u, **kw): case[u] = checkin(u, C, {"token": t, **kw})[0]
go("sv.gioi", geo=geo(10, 10))                                                                   # không chữ ký, trong vùng
go("sv.kha", device=Ks["sv.kha"].dev("sv.kha", token=t), geo=geo(10, 10))                       # WEB_KEY hợp lệ, trong vùng
go("sv.nguyco", device=Ks["sv.nguyco"].dev("sv.nguyco", token=t), geo=geo(1000, 10))           # WEB_KEY + GEO_OUT
bad = Ks["sv04"].dev("sv04", token=t); bad["signature"] = Key().sign("khac|thong|diep"); go("sv04", device=bad, geo=geo(10, 10))   # chữ ký sai
go("sv05", device={"kind": "WEB_KEY", "public_key": "###", "signature": "###"}, geo=geo(10, 10))                                  # device hỏng
shared = Key(); go("sv06", device=shared.dev("sv06", token=t), geo=geo(10, 10)); go("sv07", device=shared.dev("sv07", token=t), geo=geo(10, 10))  # máy dùng chung
go("sv08", device=Ks["sv08"].dev("sv08", token=t), geo={"denied": True})                          # geo.denied
go("sv09", device=Ks["sv09"].dev("sv09", token=t), geo={**geo(10, 10), "mock": True})              # geo.mock
go("sv10", device=Ks["sv10"].dev("sv10", token=t), geo=geo(10, 10), app={"version": "999.x-lạ"}, channel="APP")  # app.version lạ + channel tự khai
F = {u: fl(w, u) for u in case}
ok("TC-AS03-17 bảng ca tín hiệu bất thường: luôn 201, cờ tương ứng trong DB", all(v == 201 for v in case.values()) and "NO_DEVICE_KEY" in F["sv.gioi"][0] and "GEO_OUT" in F["sv.nguyco"][0] and "OFF_CAMPUS" in F["sv.kha"][0] and "DEVICE_UNTRUSTED" in F["sv04"][0] and ("NO_DEVICE_KEY" in F["sv05"][0] or "DEVICE_UNTRUSTED" in F["sv05"][0]) and "DEVICE_SHARED" in F["sv06"][0] and "DEVICE_SHARED" in F["sv07"][0] and "GEO_DENIED" in F["sv08"][0] and "GEO_MOCK" in F["sv09"][0] or False, f"{case} {F}")
ok("TC-AS04-12 assurance theo SRS 4.5: không chữ ký 1; WEB_KEY hợp lệ 2; WEB_KEY + GEO_OUT 1", F["sv.gioi"][1] == "1" and F["sv.kha"][1] == "2" and F["sv.nguyco"][1] == "1", f"{F['sv.gioi']} {F['sv.kha']} {F['sv.nguyco']}")
ok("TC-AS04-14 app.version lạ / channel tự khai APP không đổi cờ / mức (không NOT_APP, channel TOKEN)", case["sv10"] == 201 and F["sv10"][2] == "TOKEN" and "NOT_APP" not in F["sv10"][0] and F["sv10"][1] == "2", str(F["sv10"]))
ok("TC-AS04-16 public_key rác → 201 + NO_DEVICE_KEY hoặc DEVICE_UNTRUSTED, không lỗi", case["sv05"] == 201 and ("NO_DEVICE_KEY" in F["sv05"][0] or "DEVICE_UNTRUSTED" in F["sv05"][0]), str(F["sv05"]))
ok("TC-AS10-13 mọi cờ AC2–AC3: phản hồi SV vẫn 201", all(v == 201 for v in case.values()), "")
ok("TC-AS12-16 mọi ca thiết bị/chữ ký trên: 201", all(v == 201 for v in case.values()), "")
# AS12-11: khoá Keystore giả lập (DER) như WEB_KEY → mức 2, không DEVICE_UNTRUSTED
policy(C, rotate_seconds=20, late_after_min=10, require="WEB", geofence=None, campus_cidrs=[])
s, w, d = neww(C); t = tokenof(d); kd = Key()
c = checkin("sv.gioi", C, {"token": t, "device": kd.dev("sv.gioi", token=t, der=True)})[0]; f = fl(w, "sv.gioi")
ok("TC-AS12-11 emulator chưa chứng thực: WEB_KEY chữ ký DER → 201, mức 2, không DEVICE_UNTRUSTED", c == 201 and f[1] == "2" and "DEVICE_UNTRUSTED" not in f[0], str(f))
# AS10-12 SV GET course: geo_check true, không geofence / campus_cidrs
policy(C, rotate_seconds=20, geofence={"lat": LAT, "lng": LNG, "radius_m": 200}, campus_cidrs=["203.0.113.0/24"])
cc_, jj, _ = api("sv.gioi", "GET", f"/courses/{C}"); tx = json.dumps(jj)
ok("TC-AS10-12 SV GET /courses/{C}: geo_check true, không geofence / campus_cidrs / toạ độ", cc_ == 200 and jj["attendance"].get("geo_check") is True and not re.search(r"geofence|campus_cidrs|10\.7727|106\.66|203\.0\.113", tx), json.dumps(jj["attendance"]))
v = api(TEACHER, "GET", f"/courses/{C}")[1]["settings"]["version"]
r422 = [policy_ for policy_ in []]
bads = [{"self_checkin": True, "geofence": {"lat": LAT, "lng": LNG, "radius_m": 19}}, {"self_checkin": True, "geofence": {"lat": 91, "lng": LNG, "radius_m": 200}}, {"self_checkin": True, "geofence": {"lat": LAT, "lng": -181, "radius_m": 200}}, {"self_checkin": True, "geofence": {"lat": LAT, "lng": LNG, "radius_m": 2001}}]
codes = []
for b in bads:
    vv = api(TEACHER, "GET", f"/courses/{C}")[1]["settings"]["version"]; codes.append(api(TEACHER, "PUT", f"/courses/{C}/settings", {"attendance": b, "version": vv})[0])
ok("TC-AS10-15 geofence sai (radius 19/2001, lat 91, lng −181) → 422", codes == [422] * 4, str(codes))
policy(C, rotate_seconds=20, late_after_min=10, require="WEB", geofence=None, campus_cidrs=[])
# AS07-05 spot-check: 12 lượt, 3 còn cờ → 3 FLAGGED trước + 2 RANDOM
s, w, d = neww(C); t = tokenof(d)
us = ["sv.gioi", "sv.kha", "sv.nguyco"] + [f"sv{n:02d}" for n in range(4, 13)]
for k, u in enumerate(us): checkin(u, C, {"token": t, **({"device": Key().dev(u, token=t)} if k >= 3 else {})})   # 3 đầu NO_DEVICE_KEY (có cờ); 9 sau có khoá
sql(f"update attendance_checkins set flags='{{}}' where window_id='{w}' and cardinality(flags)>0 and student_id not in (select id from users where email in ('sv.gioi@edupilot.local','sv.kha@edupilot.local','sv.nguyco@edupilot.local'))")
nf = sql(f"select count(*) from attendance_checkins where window_id='{w}' and cardinality(flags)>0")[0][0]
a0 = sql(f"select count(*) from audit_log where course_id='{C}'")[0][0]; rc0 = sql(f"select count(*) from attendance_checkins where window_id='{w}'")[0][0]
c, j, _ = api(TEACHER, "POST", f"/courses/{C}/checkin-windows/{w}/spot-check", {"n": 5})
a1 = sql(f"select count(*) from audit_log where course_id='{C}'")[0][0]
st_ = j["students"] if c == 200 else []
ok("TC-AS07-05 spot-check n=5 với 3 lượt còn cờ: 3 FLAGGED trước + 2 RANDOM, không trùng, đủ trường, DB/audit không đổi", c == 200 and len(st_) == 5 and [x["reason"] for x in st_] == ["FLAGGED"] * 3 + ["RANDOM"] * 2 and len({x["student_id"] for x in st_}) == 5 and all(set(x) >= {"student_id", "full_name", "student_code", "reason"} for x in st_) and a0 == a1, f"nf={nf} {[x['reason'] for x in st_]} audit {a0}->{a1}")
close(C, w)
# ---------- LOG ----------
time.sleep(1)
gl = sh("docker logs ep-r-gateway-1 2>&1 | grep -E 'public_key|signature|\"token\"|token=|nonce|attestation' | grep -v '\"path\":\"/healthz' | grep -ciE 'checkin|device' ").stdout.strip()
gl2 = sh("docker logs ep-r-gateway-1 2>&1 | grep -E '10\\.7727|106\\.6602|lat=|lng=|\"lat\"|\"lng\"' | wc -l").stdout.strip()
ok("TC-AS03-23 log gateway: 0 dòng public_key/signature/token=/\"token\" thuộc điểm danh; không toạ độ", gl == "0" and gl2.strip() == "0", f"secrets={gl} coords={gl2}")
ok("TC-AS12-19 log gateway: nonce / public_key / chuỗi chứng chỉ = 0", sh("docker logs ep-r-gateway-1 2>&1 | grep -ciE 'attestation_chain|\"nonce\"|nonce=|BEGIN CERTIFICATE|MII[A-Za-z0-9+/]{40}'").stdout.strip() == "0", "")
s, w, d = neww(C); tk = tokenof(d)
for _ in range(3): display(C, s, w)
lg = sh(f"docker logs ep-r-gateway-1 2>&1 | grep -c -F '{tk[-24:]}'").stdout.strip() + "/" + sh(f"docker logs ep-r-caddy-1 2>&1 | grep -c -F '{tk[-24:]}'").stdout.strip()
hd = raw("GET", f"/courses/{C}/sessions/{s}/checkin-windows/{w}/display", None, {}, tok(TEACHER))[3]
ok("TC-AS02-12 token (phần sau #) không vào log gateway / Caddy; ghi nhận Referrer-Policy", lg == "0/0", f"gateway/caddy={lg}; Referrer-Policy={hd.get('Referrer-Policy') or hd.get('referrer-policy')}")
close(C, w)
