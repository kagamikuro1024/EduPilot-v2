# s8core: ca ghi/chỉnh dữ liệu trên QD8A1 (lớp riêng): 01-21/22/23/26/27/30, 02-26/27, 01-32 làm lại, 04-16 làm lại; khôi phục sau mỗi ca
import math, re
from fractions import Fraction as F
from s8core_lib import *
A1 = open("/tmp/s8core/QD8A1.id").read().strip(); Q = f"/courses/{A1}"; C1 = cid("761987")
cache = json.load(open("/tmp/s8core/a1cache.json"))
code = {r[0]: r[1] for r in sql("select student_code,id from users where student_code is not null")}
acc = {r[0]: r[1] for r in sql("select split_part(email,'@',1),id from users where email like 'sv%@edupilot.local'")}
def uidA(a): return acc[a]
c, gb = call("GET", f"{Q}/gradebook?limit=1"); items = {x["key"]: x["item_id"] for x in gb["columns"]}
EXAM = sq1(f"select id from exams where course_id='{A1}'")
def preview(): return call("GET", f"{Q}/gradebook/preview?limit=100")[1]
def explain(a): return call("GET", f"{Q}/gradebook/students/{uidA(a)}/explain")[1]
def wkline(ex): return [l for l in ex["lines"] if l["label"].startswith("Kiểm tra hằng tuần")]
import csv
exp = {r['student_code']: r for r in csv.DictReader(open(REPO + '/seed/expected_final_grades.csv'))}
import csv
# ---- 01-30: study_time_daily không ảnh hưởng
pv0 = {r["student_id"]: (r["hp"], r["letter"]) for r in preview()["items"]}; ex0 = explain("sv.nguyco")
sql(f"insert into study_time_daily(course_id,student_id,date,active_seconds,area) select '{A1}','{uidA('sv.nguyco')}', current_date - g, 36000, 'READING' from generate_series(0,13) g") if False else None
cols = [r[0] for r in sql("select column_name from information_schema.columns where table_name='study_time_daily'")]; print(cols)
try:
    sql(f"insert into study_time_daily(course_id,student_id,date,active_seconds,area) select '{A1}','{uidA('sv.nguyco')}', current_date - g, 36000, a from generate_series(0,13) g, (values ('LIBRARY'),('PRACTICE'),('CHAT')) v(a)"); ins = True
except Exception as e: print("insert study_time", e); ins = False
pv1 = {r["student_id"]: (r["hp"], r["letter"]) for r in preview()["items"]}; ex1 = explain("sv.nguyco")
sql(f"delete from study_time_daily where course_id='{A1}'")
chk("TC-P6-01-30", ins and pv0 == pv1 and ex0["lines"] == ex1["lines"] and pv1[uidA("sv.nguyco")] == ("3", "F") and "thời gian" not in json.dumps(ex1, ensure_ascii=False).lower(), f"study_time_daily C = 600 phút/ngày (14 ngày × 3 khu vực, SQL tay): preview 30 SV & giải trình C không đổi (HP 3 · F); không dòng nào nhắc thời gian học; đã xoá", f"{ins} {pv0==pv1} {pv1[uidA('sv.nguyco')]}")
