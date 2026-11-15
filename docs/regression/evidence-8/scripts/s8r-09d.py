import csv
from s8r_lib import *
E = {r["account"]: r for r in csv.DictReader(open(REPO + "/seed/expected_risk.csv"))}
A1 = cid("QD8R71")
def risk(): return {r[0]: (r[1], json.loads(r[2])) for r in sql(f"select u.student_code, r.level, r.reasons::text from student_risk r join users u on u.id=r.student_id where r.course_id='{A1}'")}
def glc(): return sum(1 for v in risk().values() if any(x["code"] == "GRADE_LOW" for x in v[1]))
def ver(): return int(sq1(f"select version from courses where id='{A1}'"))
def setg(v): return call("PUT", f"/courses/{A1}/settings", "teacher", {"version": ver(), "risk": {"grade_low_below": v}})
c0, _ = setg("5.0"); time.sleep(8); before = glc()
c, r = setg("6.0"); print(c, str(r)[:200]); after = before
for _ in range(15):
    time.sleep(4); after = glc()
    if after != before: break
inv = [setg(v)[0] for v in ("10.5", "5.3", "-1", "abc")]
C("TC-P6-09-43", c == 200 and after > before and all(x == 422 for x in inv), f"QD8R71: grade_low_below 5,0→6,0 (PUT settings {c}): số SV có GRADE_LOW {before}→{after} (crm.risk chạy lại riêng lớp); ngưỡng 10,5 / 5,3 / -1 / abc → {inv}", f"{c0} {c} {r} {before} {after} {inv}")
setg("5.0")
# 09-44
s26 = sq1(f"select u.id from users u join enrollments e on e.user_id=u.id where e.course_id='{A1}' and u.student_code='{E['sv26']['student_code']}'")
def q26(): return [x for x in risk()[E["sv26"]["student_code"]][1] if x["code"] == "GRADE_LOW"]
q0 = q26()
sql(f"insert into study_time_daily (course_id,student_id,date,active_seconds,area) select '{A1}','{s26}',(current_date - g),40000,(select area from study_time_daily limit 1) from generate_series(0,6) g on conflict do nothing")
setg("5.5"); time.sleep(12); q1 = q26(); setg("5.0")
C("TC-P6-09-44", q0 and q1 and q0[0]["values"]["qt_milli"] == q1[0]["values"]["qt_milli"] == 2125, f"sv26: nhập study_time_daily 40.000 s/ngày × 7 ngày rồi chạy lại crm.risk: qt_milli GRADE_LOW {q0[0]['values']['qt_milli'] if q0 else None} → {q1[0]['values']['qt_milli'] if q1 else None} (vẫn 2,125: thời gian học không vào điểm)", f"{q0} {q1}")
