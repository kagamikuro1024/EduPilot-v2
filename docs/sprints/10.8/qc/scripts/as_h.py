# Kiểm lại D-AS-01 (máy dùng chung qua PgBouncer), D-AS-02 (P-108-13), D-AS-03 (openapi 2–6) trên bản sửa
from asl import *
import concurrent.futures as cf, subprocess
S = st(); C = S["cid"]
policy(C, rotate_seconds=5, late_after_min=10, geofence=None, campus_cidrs=[], require="WEB")
time.sleep(61)
# D-AS-01a: hai SV cùng khoá
s, w, d = neww(C); t = tokenof(d); K = Key()
c1, j1, _ = checkin("sv.gioi", C, {"token": t, "device": K.dev("sv.gioi", token=t)}); c2, j2, _ = checkin("sv.kha", C, {"token": t, "device": K.dev("sv.kha", token=t)})
rw = {x[0].split("@")[0]: x[3] for x in dbrows(s, w)}
ok("TC-AS04-07 (kiểm lại) cùng khoá, hai SV: cả hai 201 và cả hai lượt có DEVICE_SHARED (cờ gắn cả lượt trước)", c1 == 201 and c2 == 201 and "DEVICE_SHARED" in rw.get("sv.gioi", "") and "DEVICE_SHARED" in rw.get("sv.kha", ""), f"{c1} {c2} {rw}")
# ba SV cùng khoá
c3, _, _ = checkin("sv.nguyco", C, {"token": t, "device": K.dev("sv.nguyco", token=t)}); rw = {x[0].split("@")[0]: x[3] for x in dbrows(s, w)}
ok("TC-AS04-07b SV thứ ba cùng khoá: 201; ba lượt đều DEVICE_SHARED", c3 == 201 and all("DEVICE_SHARED" in v for v in rw.values()) and len(rw) == 3, str(rw))
# đua
res = []
time.sleep(30)
for _ in range(4):
    sx, wx, dx = neww(C); tx = tokenof(dx); k = Key()
    with cf.ThreadPoolExecutor(2) as ex: rr = list(ex.map(lambda u: checkin(u, C, {"token": tx, "device": k.dev(u, token=tx)})[0], ["sv04", "sv05"]))
    f = [x[3] for x in dbrows(sx, wx)]; res.append((rr, all("DEVICE_SHARED" in x for x in f) and len(f) == 2))
ok("TC-AS04-08 (kiểm lại) đua 4 vòng: luôn hai 201 và cả hai DEVICE_SHARED", all(r == [201, 201] and b for r, b in res), str(res))
# cùng lớp khác buổi vẫn không SHARED? (phạm vi lớp): SV khác dùng cùng khoá ở buổi sau
s2, w2, d2 = neww(C); t2 = tokenof(d2); checkin("sv06", C, {"token": t2, "device": K.dev("sv06", token=t2)}); f = dbrows(s2, w2)[0][3]
ok("TC-AS04-09b cùng khoá ở buổi sau (phạm vi lớp): DEVICE_SHARED (SRS FR-12: cùng course_id)", "DEVICE_SHARED" in f, f)
# Hôm nay CHECKIN_REVIEW còn chạy (AttReviewSessions đổi text[])
close(C, w); time.sleep(2)
j = api("teacher", "GET", "/me/today")[1]; it = [x for x in (j.get("actions", []) + j.get("attention", [])) if x.get("kind") == "CHECKIN_REVIEW" and (x.get("course") or {}).get("id") == C]
time.sleep(1.5); j = api("teacher", "GET", "/me/today")[1]; it = [x for x in (j.get("actions", []) + j.get("attention", [])) if x.get("kind") == "CHECKIN_REVIEW" and (x.get("course") or {}).get("id") == C]
ok("TC-AS06-06 (kiểm lại sau đổi AttReviewSessions) /me/today vẫn có CHECKIN_REVIEW", len(it) >= 1, json.dumps([x["title"] for x in it], ensure_ascii=False))
# D-AS-02: P-108-13 (422 unknown)
c, j, _ = api("sv.kha", "POST", f"/courses/{C}/checkins", {"token": "x", "student_id": uid("sv.gioi")})
ok("TC-AS03-02b (theo P-108-13/SRS v1.6) trường lạ trong thân → 422 VALIDATION_FAILED code=unknown; không ghi lượt", c == 422 and j["details"][0]["code"] == "unknown", json.dumps(j.get("details"))[:100])
# D-AS-03: openapi + hành vi
oa = open(os.path.expanduser("~/Documents/TA_Agent_qc5/backend-go/api/openapi.yaml")).read()
i = oa.index("    DeviceRegisterInput:"); blk = oa[i:i + 900]
ok("TC-AS12-03c (D-AS-03) openapi attestation_chain minItems 2 / maxItems 6", "minItems: 2" in blk and "maxItems: 6" in blk, blk[blk.index("attestation_chain"):][:160].replace("\n", " "))
api("sv.kha", "POST", "/me/devices/challenge", None); a = api("sv.kha", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": ["AAAA"]})[0]
api("sv.kha", "POST", "/me/devices/challenge", None); b = api("sv.kha", "POST", "/me/devices", {"kind": "ANDROID_KEY", "public_key": Key().pub, "attestation_chain": ["AAAA"] * 7})[0]
ok("TC-AS12-03d 1 và 7 chứng chỉ → 422 (khớp openapi)", a == 422 and b == 422, f"{a} {b}")
