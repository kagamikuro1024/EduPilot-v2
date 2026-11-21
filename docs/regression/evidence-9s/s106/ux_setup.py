import sys, os; sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
from gdlib import *
W = "/Users/kuro/.config/edupilot-team/qc3-work/"
a = mk_assignment("ux60", 1)
# thêm 59 bài nộp từ 59 SV qcgd (đã ghi danh lớp) — bản sao một bài mẫu
sql(f"""create temp table t1 as select * from submissions where assignment_id='{a}' limit 1;
create temp table st as select u.id from users u where u.email like 'qcgd%@edupilot.local' and exists (select 1 from enrollments e where e.user_id=u.id and e.course_id='{CG0}') order by u.email limit 59;
create temp table tn as select t1.*, st.id as new_student from t1 cross join st;
update tn set id=uuidv7(), student_id=new_student, candidate_student_id=null, sha256=md5(random()::text)||md5(random()::text), blob_key='qc106/'||md5(random()::text), external_ref=null, created_at=now(), updated_at=now();
alter table tn drop column new_student;
insert into submissions select * from tn;""")
n = sql(f"select count(*) from submissions where assignment_id='{a}'")[0][0]; print("a", a, "bài nộp", n)
json.dump({"a": a}, open(W + "ux60.json", "w"))
