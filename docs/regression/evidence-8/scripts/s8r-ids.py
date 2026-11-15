# s8r-ids: xuất /tmp/qc2-s8r/ids.json (id lớp, id item, id sinh viên theo tài khoản CSV) cho spec UI
import sys, json, csv
from s8r_lib import *
CSVR = {r["account"]: r for r in csv.DictReader(open(REPO + "/seed/expected_final_grades.csv"))}
out = {}
for code in sys.argv[1:]:
    c0 = cid(code)
    if not c0: continue
    d = {"id": c0, "items": {r[0]: r[1] for r in sql(f"select key,id from grade_items where course_id='{c0}'")}, "students": {}}
    for a, r in CSVR.items():
        u = sq1(f"select u.id from users u join enrollments e on e.user_id=u.id where e.course_id='{c0}' and u.student_code='{r['student_code']}' limit 1")
        if u: d["students"][a] = u
    out[code] = d
old = json.load(open("/tmp/qc2-s8r/ids.json")) if os.path.exists("/tmp/qc2-s8r/ids.json") else {}
old.update(out); json.dump(old, open("/tmp/qc2-s8r/ids.json", "w"), ensure_ascii=False, indent=1); print(list(old))
