# s8core: dựng lớp QD8A1 = bản sao SẠCH của lớp 1 (cấu trúc + bài thi + công thức đã xác nhận + P5 trước 12:30 + điểm GK/CK đúng seed CSV)
import csv, sys
from s8core_lib import *
CODE = sys.argv[1] if len(sys.argv) > 1 else "QD8A1"
C1 = cid("761987")
TID = sq1("select id from users where email='teacher@edupilot.local'"); TAID = sq1("select id from users where email='ta@edupilot.local'")
c, r = call("POST", "/admin/courses", "admin", {"subject_code": "QCD", "class_code": CODE, "name": f"QC s8core {CODE}", "semester": "2026-2027-HK1", "capacity": 60, "teacher_id": TID})
assert c in (200, 201), (c, r)
NEW = r["id"]; print("course", NEW)
print(call("POST", f"/admin/courses/{NEW}/assign", "admin", {"ta_ids": [TAID]})[0])
seed_end = sq1(f"select max(created_at) from grade_entries where course_id='{C1}' and source='IMPORT'")
EXAM = sq1(f"select exam_id from grade_items where course_id='{C1}' and source='EXAM' limit 1")
sqlfile = f"""
\\set ON_ERROR_STOP 1
begin;
insert into enrollments (course_id,user_id,role_in_course,status,joined_via,student_code_snapshot)
 select '{NEW}', user_id,'STUDENT','ACTIVE','ROSTER',student_code_snapshot from enrollments where course_id='{C1}' and role_in_course='STUDENT' and status='ACTIVE';
create temp table smap as select id old_id, uuidv7() new_id from class_sessions where course_id='{C1}';
create temp table t_s as select * from class_sessions where course_id='{C1}';
update t_s set id=m.new_id, course_id='{NEW}' from smap m where m.old_id=t_s.id; insert into class_sessions select * from t_s;
create temp table t_a as select * from attendance_records where course_id='{C1}' and created_at <= '{seed_end}'::timestamptz + interval '3 minutes';
update t_a set id=uuidv7(), course_id='{NEW}', session_id=m.new_id from smap m where m.old_id=t_a.session_id; insert into attendance_records select * from t_a;
create temp table t_p as select * from participation_events where course_id='{C1}' and created_at <= '{seed_end}'::timestamptz + interval '3 minutes';
update t_p set id=uuidv7(), course_id='{NEW}', client_event_id=gen_random_uuid(), session_id=(select new_id from smap where old_id=t_p.session_id); insert into participation_events select * from t_p;
create temp table emap as select id old_id, uuidv7() new_id from exams where id='{EXAM}';
create temp table t_e as select * from exams where id='{EXAM}';
update t_e set id=(select new_id from emap), course_id='{NEW}'; insert into exams select * from t_e;
create temp table t_x as select * from exam_attempts where exam_id='{EXAM}' and status='GRADED';
update t_x set id=uuidv7(), course_id='{NEW}', exam_id=(select new_id from emap); insert into exam_attempts select * from t_x;
create temp table sc as select id old_id, uuidv7() new_id from grade_schemes where course_id='{C1}' and status='CONFIRMED';
create temp table t_g as select * from grade_schemes where id=(select old_id from sc);
update t_g set id=(select new_id from sc), course_id='{NEW}', source_document_id=null, extract_job_id=null; insert into grade_schemes select * from t_g;
create temp table cmap as select id old_id, uuidv7() new_id from grade_components where scheme_id=(select old_id from sc);
create temp table t_c as select * from grade_components where scheme_id=(select old_id from sc);
update t_c set id=m.new_id, course_id='{NEW}', scheme_id=(select new_id from sc) from cmap m where m.old_id=t_c.id; insert into grade_components select * from t_c;
create temp table t_r as select * from grade_rules where scheme_id=(select old_id from sc);
update t_r set id=uuidv7(), course_id='{NEW}', scheme_id=(select new_id from sc); insert into grade_rules select * from t_r;
create temp table imap as select id old_id, uuidv7() new_id from grade_items where scheme_id=(select old_id from sc);
create temp table t_i as select * from grade_items where scheme_id=(select old_id from sc);
update t_i set id=m.new_id, course_id='{NEW}', scheme_id=(select new_id from sc), component_id=(select new_id from cmap where old_id=t_i.component_id), exam_id=(select new_id from emap where old_id=t_i.exam_id) from imap m where m.old_id=t_i.id; insert into grade_items select * from t_i;
insert into gradebooks (course_id) values ('{NEW}') on conflict do nothing;
create temp table t_n as select * from grade_entries where course_id='{C1}';
update t_n set id=uuidv7(), course_id='{NEW}', item_id=(select new_id from imap where old_id=t_n.item_id); insert into grade_entries select * from t_n;
commit;
"""
open("/tmp/s8core/clone.sql", "w").write(sqlfile)
r = subprocess.run("docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot < /tmp/s8core/clone.sql", shell=True, capture_output=True, text=True); print(r.stdout[-300:], r.stderr[-600:])
# điểm GK/CK đúng seed CSV
rows = list(csv.DictReader(open(REPO + "/seed/expected_final_grades.csv")))
for it_key, col in (("midterm", "midterm"), ("final", "final_exam")):
    item = sq1(f"select id from grade_items where course_id='{NEW}' and key='{it_key}'")
    for r in rows:
        sql(f"update grade_entries e set score={r[col]}, source='IMPORT', version=1 from users u where u.id=e.student_id and u.student_code='{r['student_code']}' and e.course_id='{NEW}' and e.item_id='{item}'")
print("QD8A1", NEW); open("/tmp/s8core/QD8A1.id", "w").write(NEW) if CODE == "QD8A1" else None
