# smoke r10.8: chạy lại TC-AS03-17 / 04-14 / 10-13 / 12-16 (lần 1 test gửi thêm trường `channel` → 422 theo P-108-13 trường lạ)
from asl import *
import math
S=st(); C=S["cid"]
LAT, LNG = 10.7727, 106.6602
def geo(d, acc): return {"lat": LAT + math.degrees(d / 6371000.0), "lng": LNG, "accuracy_m": acc}
def fl(w, u):
    r = sql(f"select coalesce(nullif(array_to_string(flags,','),''),'-'), assurance, channel from attendance_checkins where window_id='{w}' and student_id='{uid(u)}'"); return r[0] if r else None
policy(C, rotate_seconds=60, late_after_min=10, require="WEB", geofence={"lat": LAT, "lng": LNG, "radius_m": 200}, campus_cidrs=["203.0.113.0/24"])
s, w, d = neww(C); t = tokenof(d); Ks = {}
def dev(u): Ks[u] = Key(); return Ks[u].dev(u, token=t)
case = {}
def go(u, **kw):
    r_ = checkin(u, C, {"token": t, **kw}); case[u] = r_[0]
    if r_[0] != 201: print("DBG", u, r_[0], json.dumps(r_[1])[:300])
go("sv.gioi", geo=geo(10, 10))
go("sv.kha", device=dev("sv.kha"), geo=geo(10, 10))
go("sv.nguyco", device=dev("sv.nguyco"), geo=geo(1000, 10))
bad = dev("sv04"); bad["signature"] = Key().sign("khac|thong|diep"); go("sv04", device=bad, geo=geo(10, 10))
go("sv05", device={"kind": "WEB_KEY", "public_key": "###", "signature": "###"}, geo=geo(10, 10))
shared = Key(); go("sv06", device=shared.dev("sv06", token=t), geo=geo(10, 10)); go("sv07", device=shared.dev("sv07", token=t), geo=geo(10, 10))
go("sv08", device=dev("sv08"), geo={"denied": True})
go("sv09", device=dev("sv09"), geo={**geo(10, 10), "mock": True})
go("sv10", device=dev("sv10"), geo=geo(10, 10), app={"version": "999.x-lạ"})
F = {u: fl(w, u) for u in case}
ok("TC-AS03-17 bảng ca tín hiệu bất thường (9 ca): luôn 201, cờ tương ứng trong DB", all(v == 201 for v in case.values()) and "NO_DEVICE_KEY" in F["sv.gioi"][0] and "GEO_OUT" in F["sv.nguyco"][0] and "OFF_CAMPUS" in F["sv.kha"][0] and "DEVICE_UNTRUSTED" in F["sv04"][0] and ("NO_DEVICE_KEY" in F["sv05"][0] or "DEVICE_UNTRUSTED" in F["sv05"][0]) and "DEVICE_SHARED" in F["sv06"][0] and "DEVICE_SHARED" in F["sv07"][0] and "GEO_DENIED" in F["sv08"][0] and "GEO_MOCK" in F["sv09"][0], f"{case} {F}")
ok("TC-AS04-14 app.version lạ không đổi cờ / mức (không NOT_APP; mức 2)", case["sv10"] == 201 and "NOT_APP" not in F["sv10"][0] and F["sv10"][1] == "2", str(F["sv10"]))
c, j, _ = checkin("sv11", C, {"token": t, "channel": "APP"})
ok("TC-AS04-14b `channel` tự khai trong thân → 422 trường lạ (P-108-13), không ghi lượt", c == 422 and not fl(w, "sv11"), f"{c} {json.dumps(j)[:120]}")
ok("TC-AS10-13 mọi cờ AC2–AC3: phản hồi SV vẫn 201", all(v == 201 for v in case.values()), str(case))
ok("TC-AS12-16 mọi ca thiết bị / chữ ký: 201", all(v == 201 for v in case.values()), "")
close(C, w); policy(C, rotate_seconds=60, geofence=None, campus_cidrs=[])
# --- 04-14c: require=APP, khai app.version bằng khoá WEB_KEY: vẫn NOT_APP
policy(C, rotate_seconds=60, late_after_min=10, require="APP", geofence=None, campus_cidrs=[])
s, w, d = neww(C); t = tokenof(d); k = Key()
c = checkin("sv12", C, {"token": t, "device": k.dev("sv12", token=t), "app": {"version": "1.0.0"}})[0]; f = fl(w, "sv12")
ok("TC-AS04-14c require=APP: WEB_KEY + khai app.version vẫn NOT_APP (khai báo máy khách không là căn cứ)", c == 201 and "NOT_APP" in f[0], str(f))
close(C, w); policy(C, rotate_seconds=20, require="WEB")
