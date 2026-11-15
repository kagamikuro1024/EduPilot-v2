# smoke r10.8: chạy lại TC-AS03-17 / 04-14 / 10-13 / 12-16 (lần 1 test gửi thêm trường `channel` → 422 theo P-108-13 trường lạ)
from asl import *
import math
S=st(); C=S["cid"]
LAT, LNG = 10.7727, 106.6602
def geo(d, acc): return {"lat": LAT + math.degrees(d / 6371000.0), "lng": LNG, "accuracy_m": acc}
def fl(w, u):
    r = sql(f"select coalesce(nullif(array_to_string(flags,','),''),'-'), assurance, channel from attendance_checkins where window_id='{w}' and student_id='{uid(u)}'"); return r[0] if r else None
def fl(w,u):
    r = sql(f"select coalesce(nullif(array_to_string(flags,','),''),'-'), assurance, channel from attendance_checkins where window_id='{w}' and student_id='{uid(u)}'"); return r[0] if r else None
# --- 04-14c: require=APP, khai app.version bằng khoá WEB_KEY: vẫn NOT_APP
policy(C, rotate_seconds=60, late_after_min=10, require="APP", geofence=None, campus_cidrs=[])
s, w, d = neww(C); t = tokenof(d); k = Key()
c = checkin("sv12", C, {"token": t, "device": k.dev("sv12", token=t), "app": {"version": "1.0.0"}})[0]; f = fl(w, "sv12")
ok("TC-AS04-14c require=APP: WEB_KEY + khai app.version vẫn NOT_APP (khai báo máy khách không là căn cứ)", c == 201 and "NOT_APP" in f[0], str(f))
close(C, w); policy(C, rotate_seconds=20, require="WEB")
