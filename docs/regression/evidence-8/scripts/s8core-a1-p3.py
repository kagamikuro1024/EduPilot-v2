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
# ---- 01-32 làm lại (item lớp 1 qua đường dẫn lớp có công thức)
item1 = sq1(f"select id from grade_items where course_id='{C1}' limit 1"); a0 = sq1(f"select count(*) from audit_log where course_id='{A1}'")
c1, _ = call("PUT", f"{Q}/grade-entries", "teacher", {"item_id": item1, "student_id": uidA("sv.kha"), "score": "5", "version": 1}); c2, _ = call("POST", f"{Q}/grade-items/{item1}/publish", "teacher", {"published": True})
a1 = sq1(f"select count(*) from audit_log where course_id='{A1}'")
chk("TC-P6-01-32", (c1, c2) == (404, 404) and a0 == a1, f"IDOR: item lớp 1 qua đường dẫn lớp có công thức (QD8A1): PUT grade-entries {c1}, publish {c2}; audit_log lớp không tăng {a0}→{a1}", f"{c1} {c2} audit {a0}->{a1}")
# ---- 04-16 làm lại với audit theo lớp
cf = json.load(open("/tmp/s8core/a1scheme.json"))
body = {"components": [{k: x[k] for k in ("key", "name", "grp", "weight")} | {"items": x["items"]} for x in cf["components"]], "rules": [{"kind": r["kind"], "applicable": r["applicable"], "params": r["params"]} for r in cf["rules"]]}
for r in body["rules"]:
    if r["kind"] == "ROUNDING": r["params"] = {"mode": "HALF_UP", "step": "0.1"}
au0, en0 = sq1(f"select count(*) from audit_log where course_id='{A1}'"), sq1(f"select count(*) from grade_entries where course_id='{A1}'")
ca, ia = call("POST", f"{Q}/grade-scheme/impact", "teacher", body); cb, ib = call("POST", f"{Q}/grade-scheme/impact", "teacher", body)
au1, en1 = sq1(f"select count(*) from audit_log where course_id='{A1}'"), sq1(f"select count(*) from grade_entries where course_id='{A1}'")
chk("TC-P6-04-16", ca == cb == 200 and au0 == au1 and en0 == en1 and ia == ib and ia["students_total"] == 30 == ia["students_computable"] and ia["changed_vs_confirmed"] == 21, f"impact 2 lần: audit_log lớp {au0}→{au1}, grade_entries {en0}→{en1} (không ghi); N=21 khớp tính tay; khoá {sorted(ia)}", f"{au0}/{au1} {en0}/{en1} {ia.get('changed_vs_confirmed')}")
# ---- 02-26: xoá ô final sv30 → INCOMPLETE
s30 = next(a for a, i in acc.items() if a == "sv30"); e = sql(f"select e.version from grade_entries e where e.course_id='{A1}' and e.item_id='{items['final']}' and e.student_id='{uidA('sv30')}'")[0][0]
c, r = call("PUT", f"{Q}/grade-entries", "teacher", {"item_id": items["final"], "student_id": uidA("sv30"), "score": None, "version": int(e)})
pv = preview(); row = next(x for x in pv["items"] if x["student_id"] == uidA("sv30"))
chk("TC-P6-02-26", c == 200 and row["status"] == "INCOMPLETE" and "hp" not in row or (row.get("hp") in (None, "")), f"xoá ô final sv30 (PUT score null {c}): preview status INCOMPLETE, không hp/letter ({ {k: row.get(k) for k in ('status','hp','letter','verdict')} })", f"{c} {r} {row}")
print("entry sv30 final after:", sql(f"select score from grade_entries where course_id='{A1}' and item_id='{items['final']}' and student_id='{uidA('sv30')}'"))
call("PUT", f"{Q}/grade-entries", "teacher", {"item_id": items["final"], "student_id": uidA("sv30"), "score": "5", "version": None})  # khôi phục (giá trị seed của sv30 sẽ ghi lại bên dưới)
import csv
exp = {r["student_code"]: r for r in csv.DictReader(open(REPO + "/seed/expected_final_grades.csv"))}
v30 = next(r for r in exp.values() if r["account"] == "sv30")["final_exam"]
sql(f"update grade_entries set score={v30}, source='IMPORT', version=1 where course_id='{A1}' and item_id='{items['final']}' and student_id='{uidA('sv30')}'")
# ---- 02-27 / 01-21: bài thi chưa công bố
sql(f"update exams set status='CLOSED' where id='{EXAM}'")
pv = preview(); sm = pv["summary"]; ex = explain("sv.kha")
fz = call("POST", f"{Q}/gradebook/finalize", "teacher", {})
txt = json.dumps(ex, ensure_ascii=False)
chk("TC-P6-01-21", sm["incomplete"] == 30 and sm["complete"] == 0 and ex["result"]["status"] == "INCOMPLETE" and ("weekly" in json.dumps(ex["result"]["missing"])) and not [l for l in wkline(ex) if l["value"] == "0"], f"bài thi chuyển CLOSED: cả lớp INCOMPLETE ({sm['incomplete']}/30), missing {ex['result']['missing']}, không có số 0 bịa; giải trình: {[x for x in ex['lines'] if 'chưa công bố' in json.dumps(x, ensure_ascii=False)][:1]}", f"{sm} {ex['result']}")
chk("TC-P6-02-27", sm["incomplete"] == 30 and fz[0] == 409 and "GRADES_INCOMPLETE" in json.dumps(fz[1]), f"bài thi chưa công bố: cả lớp INCOMPLETE ({sm['incomplete']}); finalize → {fz[0]} {fz[1].get('code') if isinstance(fz[1], dict) else ''}", f"{sm} {fz}")
sql(f"update exams set status='PUBLISHED' where id='{EXAM}'")
# ---- 01-22: sửa điểm PE → ô đổi ngay; không có bản sao điểm thi trong grade_entries
bk = sql(f"select id,auto_score,adjusted_score from exam_attempts where exam_id='{EXAM}' and student_id='{uidA('sv.kha')}'")[0]
b0 = F(wkline(explain("sv.kha"))[0]["value"].replace(",", "."))
sql(f"update exam_attempts set adjusted_score=7.00, adjusted_reason='QC s8core', adjusted_at=now() where id='{bk[0]}'")
b1 = F(wkline(explain("sv.kha"))[0]["value"].replace(",", ".")); n_exam_entries = sq1(f"select count(*) from grade_entries e join grade_items i on i.id=e.item_id where i.source='EXAM' and e.course_id='{A1}'")
chk("TC-P6-01-22", b0 == 8 and b1 == 7 and n_exam_entries == "0", f"sửa adjusted_score B 8→7: giải trình ô weekly đổi ngay {b0}→{b1}; grade_entries nguồn EXAM = {n_exam_entries}", f"{b0} {b1} {n_exam_entries}")
# ---- 01-23: thang 10 → item 20 / 30
sql(f"update grade_items set max_score=20 where course_id='{A1}' and key='weekly' and scheme_id=(select id from grade_schemes where course_id='{A1}' and status='CONFIRMED')")
sql(f"update exam_attempts set adjusted_score=8 where id='{bk[0]}'")
l20 = wkline(explain("sv.kha"))[0]
sql(f"update grade_items set max_score=30 where course_id='{A1}' and key='weekly' and scheme_id=(select id from grade_schemes where course_id='{A1}' and status='CONFIRMED')")
sql(f"update exam_attempts set adjusted_score=7.33 where id='{bk[0]}'")
l30 = wkline(explain("sv.kha"))[0]
print(l20, l30)
chk("TC-P6-01-23", l20["expr"].startswith("16 × 10 ÷ 20") and l20["value"] == "8" and l30["expr"].replace(" ", "").startswith("21,99×10÷30") and l30["value"] == "7,33", f"quy thang: item 20, điểm thi 8,0 → ô 16 (expr '{l20['expr']}') → điểm QT 8; item 30, 7,33 → 21,99 ('{l30['expr']}') → 7,33; decimal không sai số", f"{l20} {l30}")
sql(f"update grade_items set max_score=10 where course_id='{A1}' and key='weekly' and scheme_id=(select id from grade_schemes where course_id='{A1}' and status='CONFIRMED')")
sql(f"update exam_attempts set adjusted_score=null, adjusted_reason=null, adjusted_at=null where id='{bk[0]}'")
assert F(wkline(explain("sv.kha"))[0]["value"].replace(",", ".")) == 8
# ---- 01-26: EXCUSED / LATE không tính vắng
b_abs0 = F(explain("sv.kha")["lines"][4]["expr"].split("−")[0].replace("max(0;", "").strip()) if False else None
rows = sql(f"select a.id from attendance_records a where a.course_id='{A1}' and a.student_id='{uidA('sv.kha')}' and a.status='PRESENT' limit 2")
sql(f"update attendance_records set status='EXCUSED' where id='{rows[0][0]}'"); sql(f"update attendance_records set status='LATE' where id='{rows[1][0]}'")
exb = explain("sv.kha"); pl = [l for l in exb["lines"] if l["label"] == "Điểm trừ"][0]
chk("TC-P6-01-26", pl["expr"].startswith("max(0; 2 − 2)") and pl["value"] == "0", f"B thêm 1 EXCUSED + 1 LATE: vắng vẫn 2 ('{pl['expr']}'), P=0", f"{pl}")
sql(f"update attendance_records set status='PRESENT' where id in ('{rows[0][0]}','{rows[1][0]}')")
# ---- 01-27: sinh viên rời lớp
sql(f"update enrollments set status='REMOVED' where course_id='{A1}' and user_id='{uidA('sv30')}'")
gb2 = call("GET", f"{Q}/gradebook?limit=100")[1]; pv2 = preview()
chk("TC-P6-01-27", len(gb2["rows"]) == 29 and uidA("sv30") not in [r["student"]["id"] for r in gb2["rows"]] and pv2["summary"]["students"] == 29, f"sv30 enrollment REMOVED: gradebook {len(gb2['rows'])} hàng, preview students {pv2['summary']['students']} (không tính sv30)", f"{len(gb2['rows'])} {pv2['summary']}")
sql(f"update enrollments set status='ACTIVE' where course_id='{A1}' and user_id='{uidA('sv30')}'")
# ---- 01-30: study_time_daily không ảnh hưởng
pv0 = {r["student_id"]: (r["hp"], r["letter"]) for r in preview()["items"]}; ex0 = explain("sv.nguyco")
sql(f"insert into study_time_daily(course_id,student_id,date,active_seconds,area) select '{A1}','{uidA('sv.nguyco')}', current_date - g, 36000, 'READING' from generate_series(0,13) g") if False else None
cols = [r[0] for r in sql("select column_name from information_schema.columns where table_name='study_time_daily'")]; print(cols)
try:
    sql(f"insert into study_time_daily(course_id,student_id,date,active_seconds,area) select '{A1}','{uidA('sv.nguyco')}', current_date - g, 36000, a from generate_series(0,13) g, (values ('KNOWLEDGE'),('PRACTICE')) v(a)"); ins = True
except Exception as e: print("insert study_time", e); ins = False
pv1 = {r["student_id"]: (r["hp"], r["letter"]) for r in preview()["items"]}; ex1 = explain("sv.nguyco")
sql(f"delete from study_time_daily where course_id='{A1}'")
chk("TC-P6-01-30", ins and pv0 == pv1 and ex0["lines"] == ex1["lines"] and pv1[uidA("sv.nguyco")] == ("3", "F") and "thời gian" not in json.dumps(ex1, ensure_ascii=False).lower(), f"study_time_daily C = 600 phút/ngày (14 ngày × 2 khu vực, SQL tay): preview 30 SV & giải trình C không đổi (HP 3 · F); không dòng nào nhắc thời gian học; đã xoá", f"{ins} {pv0==pv1} {pv1[uidA('sv.nguyco')]}")
