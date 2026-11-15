#!/bin/bash
# s8flow: dựng bản sao lớp 761987 (lớp 1) thành lớp riêng: s8flow-clone.sh <MÃ_LỚP> [no-grades]
#  - lớp tạo bằng API admin (GV teacher@, TA ta@), 30 SV ROSTER + MSSV snapshot, 15 buổi + điểm danh, sự kiện phát biểu,
#    bài thi tuần 9 (MCQ) + lượt GRADED, công thức CONFIRMED (3 thành phần, 3 mục, 7 luật), điểm midterm/final, sổ OPEN.
source /tmp/ep-r-lib.sh
CC=$1; MODE=${2:-}
C1=$(psqlr -c "select id from courses where class_code='761987'")
if [ -n "${CID:-}" ]; then :; else OUT=$(bash $REPO/docs/regression/evidence-9/scripts/mkcourse.sh $CC); echo "$OUT" | head -3
CID=$(echo "$OUT" | awk '/^course /{print $2}'); fi
#; [ -n "$CID" ] || { echo "no course"; exit 1; }
cat > /tmp/qc2-s8f/clone-$CC.sql <<SQL
\set ON_ERROR_STOP 1
\set c1 '$C1'
\set cid '$CID'
begin;
insert into enrollments (course_id,user_id,role_in_course,status,joined_via,student_code_snapshot)
 select :'cid', user_id,'STUDENT','ACTIVE','ROSTER',student_code_snapshot from enrollments where course_id=:'c1' and role_in_course='STUDENT' and status='ACTIVE';
create temp table smap as select id old_id, uuidv7() new_id from class_sessions where course_id=:'c1';
create temp table t_s as select * from class_sessions where course_id=:'c1';
update t_s set id=m.new_id, course_id=:'cid' from smap m where m.old_id=t_s.id; insert into class_sessions select * from t_s;
create temp table t_a as select * from attendance_records where course_id=:'c1';
update t_a set id=uuidv7(), course_id=:'cid', session_id=m.new_id from smap m where m.old_id=t_a.session_id; insert into attendance_records select * from t_a;
create temp table t_p as select * from participation_events where course_id=:'c1';
update t_p set id=uuidv7(), course_id=:'cid', client_event_id=gen_random_uuid(), session_id=(select new_id from smap where old_id=t_p.session_id); insert into participation_events select * from t_p;
create temp table emap as select id old_id, uuidv7() new_id from exams where id='01a1df28-122f-7580-b4c2-b7eb0c3506e9';
create temp table t_e as select * from exams where id='01a1df28-122f-7580-b4c2-b7eb0c3506e9';
update t_e set id=(select new_id from emap), course_id=:'cid'; insert into exams select * from t_e;
create temp table t_x as select * from exam_attempts where exam_id='01a1df28-122f-7580-b4c2-b7eb0c3506e9' and status='GRADED';
update t_x set id=uuidv7(), course_id=:'cid', exam_id=(select new_id from emap); insert into exam_attempts select * from t_x;
create temp table sc as select id old_id, uuidv7() new_id from grade_schemes where course_id=:'c1' and status='CONFIRMED';
create temp table t_g as select * from grade_schemes where id=(select old_id from sc);
update t_g set id=(select new_id from sc), course_id=:'cid', source_document_id=null, extract_job_id=null; insert into grade_schemes select * from t_g;
create temp table cmap as select id old_id, uuidv7() new_id from grade_components where scheme_id=(select old_id from sc);
create temp table t_c as select * from grade_components where scheme_id=(select old_id from sc);
update t_c set id=m.new_id, course_id=:'cid', scheme_id=(select new_id from sc) from cmap m where m.old_id=t_c.id; insert into grade_components select * from t_c;
create temp table t_r as select * from grade_rules where scheme_id=(select old_id from sc);
update t_r set id=uuidv7(), course_id=:'cid', scheme_id=(select new_id from sc); insert into grade_rules select * from t_r;
create temp table imap as select id old_id, uuidv7() new_id from grade_items where scheme_id=(select old_id from sc);
create temp table t_i as select * from grade_items where scheme_id=(select old_id from sc);
update t_i set id=m.new_id, course_id=:'cid', scheme_id=(select new_id from sc), component_id=(select new_id from cmap where old_id=t_i.component_id), exam_id=(select new_id from emap where old_id=t_i.exam_id) from imap m where m.old_id=t_i.id; insert into grade_items select * from t_i;
insert into gradebooks (course_id) values (:'cid') on conflict do nothing;
SQL
[ "$MODE" = no-grades ] || [ "$MODE" = draft ] || cat >> /tmp/qc2-s8f/clone-$CC.sql <<SQL
create temp table t_n as select * from grade_entries where course_id=:'c1';
update t_n set id=uuidv7(), course_id=:'cid', item_id=(select new_id from imap where old_id=t_n.item_id); insert into grade_entries select * from t_n;
SQL
[ "$MODE" = draft ] && echo "update grade_schemes set status='DRAFT', confirmed_by=null, confirmed_at=null where course_id=:'cid';" >> /tmp/qc2-s8f/clone-$CC.sql
echo "commit;" >> /tmp/qc2-s8f/clone-$CC.sql
docker exec -i ep-r-postgres-1 psql -U edupilot -d edupilot -q < /tmp/qc2-s8f/clone-$CC.sql && echo "CLONED $CC $CID"
echo $CID > /tmp/qc2-s8f/$CC.id
psqlr -c "update enrollments set user_id=(select id from users where email='gv.s8f@edupilot.local') where role_in_course='TEACHER' and course_id='$CID'"
