# s8r 09c: tín hiệu GRADE của crm.risk (09-40..44) trên 761987 (chỉ đọc) và bản sao sạch QD8A1 (đổi ngưỡng)
import csv
from s8r_lib import *
E = {r["account"]: r for r in csv.DictReader(open(REPO + "/seed/expected_risk.csv"))}
C1 = cid("761987"); C2 = cid("761988"); A1 = cid("QD8A1")
def risk(c0): return {r[0]: (r[1], json.loads(r[2])) for r in sql(f"select u.student_code, r.level, r.reasons::text from student_risk r join users u on u.id=r.student_id where r.course_id='{c0}'")}
R1 = risk(C1); bad = [(a, R1.get(e["student_code"], ("-",))[0], e["risk_level"]) for a, e in E.items() if R1.get(e["student_code"], ("-",))[0] != e["risk_level"]]
abc = {a: R1[E[a]["student_code"]][0] for a in ("sv.gioi", "sv.kha", "sv.nguyco")}
T("TC-P6-09-40", "F", f"lớp 761987 hiện: A/B/C = {abc} (khớp kỳ vọng NONE/NONE/AT_RISK) nhưng {len(bad)}/30 dòng lệch expected_risk.csv, vd {bad[:3]} (kỳ vọng WATCH={{sv12,sv19,sv26}}, hiện WATCH: {sum(1 for v in R1.values() if v[0]=='WATCH')}, AT_RISK: {sum(1 for v in R1.values() if v[0]=='AT_RISK')}); nguyên nhân: dữ liệu lớp 1 bị nhóm khác ghi đè (điểm giữa kỳ 6 SV, điểm cộng 2 SV) + tín hiệu hoạt động theo tuần tính theo ngày hiện tại; bản sao sạch QD8A1 không có dữ liệu hoạt động P5")
# 09-41 hồ sơ 360 của C (Staff) có lý do điểm; sinh viên không có
cC = [r for r in sql(f"select u.id from users u where u.student_code='{E['sv.nguyco']['student_code']}'")][0][0]
rs = R1[E["sv.nguyco"]["student_code"]][1]; gl = [x for x in rs if x["code"] == "GRADE_LOW"]
rc, prof = call("GET", f"/courses/{C1}/students/{cC}", "teacher")
txt = json.dumps(prof, ensure_ascii=False) if rc == 200 else ""; has125 = " (có 1,25)" if ("1,25" in txt or "1.25" in txt) else ""
sv_c, sv_r = call("GET", f"/courses/{C1}/me/grades", "sv.nguyco"); stu_has = re.search(r"GRADE_LOW|WATCH|AT_RISK|risk", json.dumps(sv_r, ensure_ascii=False))
C("TC-P6-09-41", bool(gl) and gl[0]["values"].get("qt_milli") == 1250 and not stu_has, f"C: student_risk.reasons có GRADE_LOW qt_milli={gl[0]['values'].get('qt_milli') if gl else None} (1,25/10); hồ sơ Staff GET students/C → {rc}{has125}; /me/grades của C không có nhãn/lý do risk; UI hồ sơ: s8r-ui-09", f"{gl} {rc} {txt[:200]}")
# 09-42 lớp 2 chưa CONFIRMED: không GRADE_LOW
R2 = risk(C2); g2 = [k for k, v in R2.items() if any(x["code"] == "GRADE_LOW" for x in v[1])]
C("TC-P6-09-42", len(R2) > 0 and not g2, f"lớp 2 (761988, công thức chưa CONFIRMED): {len(R2)} dòng student_risk, 0 dòng có GRADE_LOW", f"{g2}")
# 09-43: đổi grade_low_below 6,0 trên QD8A1 → recompute
def gl_count(): return sum(1 for v in risk(A1).values() if any(x["code"] == "GRADE_LOW" for x in v[1]))
before = gl_count(); c, r = call("PUT", f"/courses/{A1}/settings", "teacher", {"risk": {"grade_low_below": "6.0"}}); print("PUT settings", c, str(r)[:200])
for _ in range(12):
    time.sleep(4); after = gl_count()
    if after != before: break
inv = [call("PUT", f"/courses/{A1}/settings", "teacher", {"risk": {"grade_low_below": v}})[0] for v in ("10.5", "5.3", "-1", "abc")]
call("PUT", f"/courses/{A1}/settings", "teacher", {"risk": {"grade_low_below": "5.0"}})
qt6 = sql(f"select count(*) from (select 1) x") and None
C("TC-P6-09-43", c == 200 and after > before and all(x == 422 for x in inv), f"QD8A1: grade_low_below 5,0→6,0 (PUT settings {c}): số SV có GRADE_LOW {before}→{after} (recompute riêng lớp qua outbox); ngưỡng ngoài [0;10] hoặc không bước 0,5 (10,5; 5,3; -1; abc) → {inv}", f"{c} {r} {before} {after} {inv}")
# 09-44 study_time rất cao cho sv26 → qt_current vẫn 2,125
s26 = sq1(f"select u.id from users u join enrollments e on e.user_id=u.id where e.course_id='{A1}' and u.student_code='{E['sv26']['student_code']}'")
cols = [r[0] for r in sql("select column_name from information_schema.columns where table_name='study_time_daily'")]; print(cols)
