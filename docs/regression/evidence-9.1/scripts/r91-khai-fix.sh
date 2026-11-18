#!/bin/bash
# r91-khai-fix.sh <MÃ_LỚP>: thêm lượt chấm kế tiếp cho Khải trên lớp sao = bản kết quả seed của 761987 (lượt 1 = 7,00 / lượt 2 = 8,50 ⇒ «Lệch 1,5 điểm so với lượt 2»); LLM thật cho 10/10 không lệch, và bản nháp AI bất biến nên chỉ thêm hàng
source /tmp/ep-r-lib.sh
COLS=$(psqlr -c "select string_agg(column_name, ',' order by ordinal_position) from information_schema.columns where table_name='grading_results' and is_generated='NEVER'")
SC=$(psqlr -c "select s.id from submissions s join users u on u.id=s.student_id where s.course_id=(select id from courses where class_code='$1') and u.full_name like '%Khải%' and s.assignment_id is not null limit 1")
docker exec -i ep-r-postgres-1 sh -c 'psql -U edupilot -d edupilot -q -v ON_ERROR_STOP=1; exit 0' <<SQL
begin;
create temp table t_g as select * from grading_results where id=(select g2.id from grading_results g2 join submissions s2 on s2.id=g2.submission_id join users u2 on u2.id=s2.student_id join courses c2 on c2.id=g2.course_id where c2.class_code='761987' and u2.full_name like '%Khải%' order by g2.round desc limit 1);
update t_g set id=uuidv7(), course_id=(select id from courses where class_code='$1'), submission_id='$SC', round=(select coalesce(max(round),0)+1 from grading_results where submission_id='$SC'), reviewed_by=null, reviewed_at=null, published_by=null, published_at=null, first_reviewed_at=null, first_reviewed_by=null;
insert into grading_results ($COLS) select $COLS from t_g;
commit;
SQL
psqlr -c "select round,status,total,pass_detail->'total' from grading_results where submission_id='$SC' order by round"
