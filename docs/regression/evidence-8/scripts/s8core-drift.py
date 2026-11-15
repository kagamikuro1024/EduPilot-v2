# s8core TC-P6-02-03 / điều tra 7 lệch: so preview (API) vs (a) expected_final_grades.csv và (b) tính tay Fraction từ ĐẦU VÀO THẬT hiện có của lớp 1 trong DB
import csv, importlib.util, sys
from fractions import Fraction as F
from s8core_lib import *
spec = importlib.util.spec_from_file_location("hc", REPO + "/docs/sprints/8/qc/scripts/p6-hand-calc.py"); hc = importlib.util.module_from_spec(spec); spec.loader.exec_module(hc)
C1 = cid("761987")
c, pv = call("GET", f"/courses/{C1}/gradebook/preview?limit=100"); assert c == 200
api = {r["student_code"]: r for r in pv["items"]}
exp = {r["student_code"]: r for r in csv.DictReader(open(REPO + "/seed/expected_final_grades.csv"))}
inp = {r["student_code"]: r for r in csv.DictReader(open(REPO + "/docs/specs/FEAT-gradebook/fixtures/inputs-lop1.csv"))}
c, gb = call("GET", f"/courses/{C1}/gradebook?limit=100"); cols = {x["key"]: x["item_id"] for x in gb["columns"]}
ents = {r["student"]["student_code"]: r["entries"] for r in gb["rows"]}
# P5 đầu vào thật
pe = {}
for code, typ, pts in sql(f"select u.student_code,p.type,p.points from participation_events p join users u on u.id=p.student_id where p.course_id='{C1}'"): pe.setdefault(code, []).append((typ, F(pts)))
ab = dict(sql(f"select u.student_code,count(*) from attendance_records a join users u on u.id=a.student_id where a.course_id='{C1}' and a.status='ABSENT' group by 1"))
uids = {r[0]: r[1] for r in sql(f"select student_code,id from users where student_code is not null")}
n_csv = n_live = 0; diffs = []; EX = {}
for code, e in exp.items():
    a = api[code]; ok_csv = (a["hp"] == e["hp"] or F(a["hp"]) == F(e["hp"])) and a["letter"] == e["letter"] and F(a["qt"]) == F(e["qt"]) and F(a["hp_raw"]) == F(e["hp_raw"])
    n_csv += ok_csv
    cc, ex = call("GET", f"/courses/{C1}/gradebook/students/{uids[code]}/explain"); EX[code] = ex; wk = [l for l in ex["lines"] if l["label"].startswith("Kiểm tra hằng tuần")][0]["value"].replace(",", ".")
    en = ents[code]
    gk = F(en[cols["midterm"]]["score"]); ck = F(en[cols["final"]]["score"]); kt = F(wk)
    ev = pe.get(code, []); bonus = sum((p for t, p in ev if t in ("SPEAK", "BONUS")), F(0)); pen = sum((p for t, p in ev if t == "PENALTY"), F(0))
    qt, raw, hp, letter, verdict, _ = hc.compute(gk, kt, ck, bonus, pen, int(ab.get(code, 0)))
    ok_live = F(a["qt"]) == qt and F(a["hp_raw"]) == raw and F(a["hp"]) == hp and a["letter"] == letter and (a["verdict"] == "OK") == (verdict == "OK")
    n_live += ok_live
    if not ok_csv:
        i = inp[code]; why = []
        if F(i["midterm"]) != gk: why.append(f"midterm seed {i['midterm']} → DB {gk} ({en[cols['midterm']]['source']} v{en[cols['midterm']]['version']})")
        if F(i["final_exam"]) != ck: why.append(f"final seed {i['final_exam']} → DB {ck}")
        if F(i["speak_bonus_points"]) != bonus: why.append(f"bonus seed {i['speak_bonus_points']} → DB {bonus}")
        if F(i["penalty_event_points"]) != pen: why.append(f"penalty seed {i['penalty_event_points']} → DB {pen}")
        if int(i["absent"]) != int(ab.get(code, 0)): why.append(f"absent seed {i['absent']} → DB {ab.get(code,0)}")
        if F(i["weekly_exam_t9"] or 0) != kt: why.append(f"weekly seed {i['weekly_exam_t9']} → {kt}")
        diffs.append((e["account"], code, "; ".join(why) or "đầu vào giống seed?!"))
json.dump({"preview": api, "explain": EX, "gradebook": gb}, open("/tmp/s8core/c1cache.json", "w"), ensure_ascii=False)
print(f"preview API vs CSV BA: {n_csv}/30 khớp"); print(f"preview API vs tính tay (Fraction) từ đầu vào THẬT trong DB: {n_live}/30 khớp")
for d in diffs: print("LỆCH CSV:", *d)
c = sq1(f"select count(*) from grade_schemes where course_id='{C1}'"); print("số bản công thức lớp 1:", c, sql(f"select scheme_no,status,version from grade_schemes where course_id='{C1}'"))
print("entries không phải IMPORT/seed:", sql(f"select u.student_code,i.key,e.source,e.version,e.updated_at from grade_entries e join grade_items i on i.id=e.item_id join users u on u.id=e.student_id where e.course_id='{C1}' and e.source<>'IMPORT' order by 5"))
print("participation sau seed:", sql(f"select u.student_code,p.type,p.points,p.created_at from participation_events p join users u on u.id=p.student_id where p.course_id='{C1}' and p.created_at > (select min(created_at) from grade_entries where course_id='{C1}') + interval '1 hour' order by 4"))
rec("TC-P6-02-03", "F" if n_csv != 30 else "P", f"preview vs CSV {n_csv}/30; vs tính tay đầu vào thật {n_live}/30")
