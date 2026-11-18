#!/bin/bash
# r91-clone.sh <MÃ_LỚP>: s8flow-clone (30 SV, buổi/điểm danh, sổ) + sao bài tập của 761987 và các bài nộp CHƯA có kết quả chấm (dữ liệu sạch cho UX12/14/15/16)
source /tmp/ep-r-lib.sh
CC=$1
mkdir -p /tmp/qc2-s8f
bash $REPO/docs/regression/evidence-8/scripts/s8flow-clone.sh $CC
CID=$(cat /tmp/qc2-s8f/$CC.id)
cat > /tmp/qc2-s8f/r91a-$CC.sql <<SQL
\set ON_ERROR_STOP 1
\set cid '$CID'
begin;
create temp table amap as select id old_id, uuidv7() new_id from assignments where course_id=(select id from courses where class_code='761987');
create temp table t_a as select * from assignments where course_id=(select id from courses where class_code='761987');
update t_a set id=m.new_id, course_id=:'cid' from amap m where m.old_id=t_a.id; insert into assignments select * from t_a;
create temp table t_s as select * from submissions where course_id=(select id from courses where class_code='761987');
update t_s set id=uuidv7(), course_id=:'cid', assignment_id=(select new_id from amap where old_id=t_s.assignment_id);
insert into submissions select * from t_s;
commit;
SQL
docker exec -i ep-r-postgres-1 sh -c 'psql -U edupilot -d edupilot -q; exit 0' < /tmp/qc2-s8f/r91a-$CC.sql && echo "ASSIGN-CLONED $CC $CID"
psqlr -c "update enrollments set user_id=(select id from users where email='teacher@edupilot.local') where role_in_course='TEACHER' and course_id='$CID'"
