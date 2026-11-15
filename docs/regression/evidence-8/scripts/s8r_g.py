# s8r_g: ngữ cảnh một lớp gradebook (CODE qua env S8CODE) + hàm tiện: cell/put/snap/gb/notif.
import csv, hashlib, uuid, time, subprocess, concurrent.futures as cf
from s8r_lib import *
CODE = os.environ.get("S8CODE", "QD8R8"); C0 = cid(CODE); Q = f"/courses/{C0}"
ITEM = {k: sq1(f"select id from grade_items where course_id='{C0}' and key='{k}'") for k in ("midterm", "weekly", "final")}
CSV = {r["account"]: r for r in csv.DictReader(open(REPO + "/seed/expected_final_grades.csv"))}
TCH = sq1("select id from users where email='teacher@edupilot.local'")
def SV(a): return sq1("select u.id from users u join enrollments e on e.user_id=u.id where e.course_id='%s' and e.role_in_course='STUDENT' and u.student_code='%s'" % (C0, CSV[a]["student_code"]))
def cell(a, k):
    r = sql(f"select score,version from grade_entries where course_id='{C0}' and item_id='{ITEM[k]}' and student_id='{SV(a)}'"); return (r[0][0], int(r[0][1])) if r else (None, None)
def put(a, k, score, ver, who="teacher"): return call("PUT", f"{Q}/grade-entries", who, {"item_id": ITEM[k], "student_id": SV(a), "score": score, "version": ver})
def setc(a, k, score):
    s, v = cell(a, k); return put(a, k, score, v)
def snap(v): return sql(f"select student_id,numeric_grade,letter,coalesce(failed_reason,''),coalesce(unlock_reason,''),breakdown::text from final_grade_snapshots where course_id='{C0}' and version={v} order by student_id")
def h(v): return hashlib.md5(json.dumps(snap(v)).encode()).hexdigest()
def gb(): return sql(f"select status,snapshot_version,coalesce(unlock_reason,'') from gradebooks where course_id='{C0}'")[0]
def notif(t, cc=None): return int(sq1(f"select count(*) from notifications n where n.type='{t}' and n.course_id='{cc or C0}'"))
def fin(key=None, who="teacher"): return call("POST", f"{Q}/gradebook/finalize", who, {}, key=key)
def unl(reason, who="teacher", key=None): return call("POST", f"{Q}/gradebook/unlock", who, {"reason": reason}, key=key)
