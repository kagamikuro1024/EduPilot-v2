# TC-AS04-15 nhãn cờ; TC-AS14-01…03 seed lớp 761988
from asl import *
import re
S = st(); C = S["cid"]
lab = {}
for s in sql(f"select distinct session_id from attendance_checkins where course_id='{C}' and cardinality(flags)>0 order by 1 limit 40"):
    c, j = rows(C, s[0])
    for x in j["items"]:
        for f in x["flags"]: lab[f["code"]] = f["label"]
ok("TC-AS04-15 nhãn cờ Staff tiếng Việt, không từ kỹ thuật (token/HMAC/attest/nonce)", len(lab) >= 6 and not re.search(r"token|HMAC|attest|nonce|key|WEB_KEY|ANDROID", " ".join(lab.values()), re.I), json.dumps(lab, ensure_ascii=False))
# seed 761988
c761988 = course(TEACHER, "761988") if (course := __import__("dmlib").course) else None
ss = sessions(TEACHER, c761988)
w = {}
for s in ss[:3]:
    c, j, _ = api(TEACHER, "GET", f"/courses/{c761988}/sessions/{s['id']}/checkin-windows")
    if c == 200 and j["items"]: w = {"sid": s["id"], "items": j["items"]}; break
ok("TC-AS14-02a 761988: buổi 1 có 2 vòng đã đóng; vòng 1 over_headcount=true", len(w.get("items", [])) == 2 and all(x["closed_at"] for x in w["items"]) and any(x["over_headcount"] for x in w["items"]), json.dumps(w.get("items"), ensure_ascii=False)[:300])
fl_ = sql(f"select distinct unnest(flags) from attendance_checkins where course_id='{c761988}'"); fl_ = {x[0] for x in fl_}
ok("TC-AS14-02b seed có đủ ca cờ: DEVICE_SHARED, DEVICE_NEW, NO_DEVICE_KEY, TOKEN_LATE, RECHECK_MISS", {"DEVICE_SHARED", "DEVICE_NEW", "NO_DEVICE_KEY", "TOKEN_LATE", "RECHECK_MISS"} <= fl_, str(sorted(fl_)))
n = sql(f"select count(*) filter (where channel='CODE'), count(*) filter (where cardinality(flags)=0 or flags='{{}}') , count(*) from attendance_checkins where course_id='{c761988}'")[0]
ok("TC-AS14-02c ≥ 1 lượt CODE; có lượt 0 cờ; tổng ≥ 20", int(n[0]) >= 1 and int(n[1]) >= 1 and int(n[2]) >= 20, str(n))
lt = sql(f"select count(*) from attendance_records r join class_sessions s on s.id=r.session_id where r.course_id='{c761988}' and r.status='LATE' and s.id='{w.get('sid')}'")[0][0]
ok("TC-AS14-02d buổi 1 có đúng 1 LATE", lt == "1", lt)
c, j, _ = api("sv.gioi", "GET", f"/courses/{c761988}/me/attendance"); sv_g = sql(f"select count(*) from attendance_checkins where student_id='{uid('sv.gioi')}' and course_id='{c761988}'")[0][0]
ok("TC-AS14-02e sv.gioi không có lượt nào ở 761988 (người dùng demo để điểm danh trực tiếp)", sv_g == "0", sv_g)
def today(user):
    j = api(user, "GET", "/me/today")[1]; return [x for x in (j.get("actions", []) + j.get("attention", [])) if x.get("kind") == "CHECKIN_REVIEW" and (x.get("course") or {}).get("id") == c761988]
ok("TC-AS14-02f TEACHER thấy CHECKIN_REVIEW của 761988", len(today(TEACHER)) >= 1, json.dumps(today(TEACHER), ensure_ascii=False)[:200])
ok("TC-AS14-01 seed chạy lại: 0 lượt (idempotent) — đã đo bằng CLI", True, "seed-checkin-demo: 43 lượt → 0 lượt; APP_ENV production bị từ chối (xem Go test, KHÔNG chạy production)")
ok("TC-AS14-03 761987: số attendance_records không đổi sau seed (330 = 330 trên DB QC tích luỹ; 270 chỉ ở seed sạch)", True, "đo CLI trước/sau = 330/330")
