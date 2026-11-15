# s8core: 01-34 (SV rời lớp giữ snapshot) + 04-27 (LOCKED → confirm 409) trên QD8A1
from s8core_lib import *
A1 = open("/tmp/s8core/QD8A1.id").read().strip(); Q = f"/courses/{A1}"
sv30 = sq1("select id from users where email='sv30@edupilot.local'")
print("gradebook:", sql(f"select status,snapshot_version from gradebooks where course_id='{A1}'"), "snap v1:", sq1(f"select count(*) from final_grade_snapshots where course_id='{A1}' and version=1"))
old = sql(f"select numeric_grade,letter,locked_at from final_grade_snapshots where course_id='{A1}' and student_id='{sv30}' and version=1")
sql(f"update enrollments set status='REMOVED', removed_at=now() where course_id='{A1}' and user_id='{sv30}'")
c, f2 = call("POST", f"{Q}/gradebook/finalize", "teacher", {}); print(c, f2)
new = sql(f"select count(*) from final_grade_snapshots where course_id='{A1}' and student_id='{sv30}' and version=2"), sq1(f"select count(*) from final_grade_snapshots where course_id='{A1}' and version=2")
old2 = sql(f"select numeric_grade,letter,locked_at from final_grade_snapshots where course_id='{A1}' and student_id='{sv30}' and version=1")
chk("TC-P6-01-34", c == 200 and old and old == old2 and new[0][0][0] == "0" and new[1] == "29", f"sv30 rời lớp rồi chốt lại: snapshot v1 của sv30 còn nguyên {old2[0][:2]}; v2 chỉ 29 SV, không có dòng v2 cho sv30", f"{c} {f2} {old} {old2} {new}")
# 04-27: LOCKED → confirm DRAFT → 409 GRADEBOOK_LOCKED
d = call("GET", f"{Q}/grade-scheme", "teacher")[1]["draft"]
st0 = sql(f"select status,version from grade_schemes where id='{d['id']}'")
cc, jj = call("POST", f"{Q}/grade-scheme/confirm", "teacher", {"version": d["version"]})
st1 = sql(f"select status,version from grade_schemes where id='{d['id']}'")
chk("TC-P6-04-27", cc == 409 and "GRADEBOOK_LOCKED" in json.dumps(jj) and st0 == st1 and st1[0][0] == "DRAFT", f"sổ LOCKED: confirm → {cc} {jj.get('code')}; DRAFT không đổi {st1}", f"{cc} {jj} {st0} {st1}")
# khôi phục
print(call("POST", f"{Q}/gradebook/unlock", "teacher", {"reason": "QC s8core mở khoá sau khi thử 01-34 và 04-27"}))
sql(f"update enrollments set status='ACTIVE', removed_at=null where course_id='{A1}' and user_id='{sv30}'")
print(sql(f"select status,snapshot_version from gradebooks where course_id='{A1}'"))
