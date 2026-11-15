# smoke r10.8: TC-AS04-15 + TC-AS14-01…03 trên seed 761988 (đọc bằng tài khoản seed teacher@, chỉ GET) + đo thật CLI seed-checkin-demo (as_g.py cũ ghi PASS cứng cho 14-01/03)
from asl import *
import re, subprocess
SEEDT = "teacher"
c761988 = sql("select id from courses where class_code='761988'")[0][0]; c761987 = sql("select id from courses where class_code='761987'")[0][0]
lab = {}
for s in sql(f"select distinct session_id from attendance_checkins where course_id='{c761988}' and cardinality(flags)>0"):
    c, j, _ = api(SEEDT, "GET", f"/courses/{c761988}/sessions/{s[0]}/checkins?limit=100")
    for x in j["items"]:
        for f in x["flags"]: lab[f["code"]] = f["label"]
ok("TC-AS04-15 nhãn cờ Staff tiếng Việt, không từ kỹ thuật (761988)", len(lab) >= 5 and not re.search(r"token|HMAC|attest|nonce|key|WEB_KEY|ANDROID", " ".join(lab.values()), re.I), json.dumps(lab, ensure_ascii=False))
ss = sessions(SEEDT, c761988); w = {}
for s in ss[:3]:
    c, j, _ = api(SEEDT, "GET", f"/courses/{c761988}/sessions/{s['id']}/checkin-windows")
    if c == 200 and j["items"]: w = {"sid": s["id"], "items": j["items"]}; break
ok("TC-AS14-02a 761988: buổi 1 có 2 vòng đã đóng; vòng 1 over_headcount=true", len(w.get("items", [])) == 2 and all(x["closed_at"] for x in w["items"]) and any(x["over_headcount"] for x in w["items"]), json.dumps(w.get("items"), ensure_ascii=False)[:300])
fl_ = {x[0] for x in sql(f"select distinct unnest(flags) from attendance_checkins where course_id='{c761988}'")}
ok("TC-AS14-02b seed có đủ ca cờ: DEVICE_SHARED, DEVICE_NEW, NO_DEVICE_KEY, TOKEN_LATE, RECHECK_MISS", {"DEVICE_SHARED", "DEVICE_NEW", "NO_DEVICE_KEY", "TOKEN_LATE", "RECHECK_MISS"} <= fl_, str(sorted(fl_)))
n = sql(f"select count(*) filter (where channel='CODE'), count(*) filter (where cardinality(flags)=0), count(*) from attendance_checkins where course_id='{c761988}'")[0]
ok("TC-AS14-02c ≥ 1 lượt CODE; có lượt 0 cờ; tổng ≥ 20", int(n[0]) >= 1 and int(n[1]) >= 1 and int(n[2]) >= 20, str(n))
lt = sql(f"select count(*) from attendance_records r where r.course_id='{c761988}' and r.status='LATE' and r.session_id='{w.get('sid')}'")[0][0]
ok("TC-AS14-02d buổi 1 có đúng 1 LATE", lt == "1", lt)
sv_g = sql(f"select count(*) from attendance_checkins where student_id='{uid('sv.gioi')}' and course_id='{c761988}'")[0][0]
ok("TC-AS14-02e sv.gioi không có lượt nào ở 761988", sv_g == "0", sv_g)
a0 = sql(f"select count(*) from attendance_records where course_id='{c761987}'")[0][0]; k0 = sql(f"select count(*) from attendance_checkins where course_id='{c761988}'")[0][0]
run = lambda: subprocess.run("docker exec ep-r-gateway-1 /gateway admin seed-checkin-demo --course 761988", shell=True, capture_output=True, text=True)
r1 = run(); r2 = run()
a1 = sql(f"select count(*) from attendance_records where course_id='{c761987}'")[0][0]; k1 = sql(f"select count(*) from attendance_checkins where course_id='{c761988}'")[0][0]
ok("TC-AS14-01 seed chạy lại 2 lần: không nhân đôi lượt", r1.returncode == 0 and r2.returncode == 0 and k0 == k1, f"rc={r1.returncode},{r2.returncode} out={r1.stdout.strip()!r} {r2.stdout.strip()!r} lượt {k0}->{k1} err={(r1.stderr+r2.stderr)[:150]}")
ok("TC-AS14-03 761987: attendance_records không đổi sau seed (đo thật)", a0 == a1 == "270", f"{a0} -> {a1}")
