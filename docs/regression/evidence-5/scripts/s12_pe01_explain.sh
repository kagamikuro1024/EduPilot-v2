#!/bin/bash
# TC-PE01-10: dữ liệu giả 20.000 câu / 2.000 lượt / 20.000 bài nộp trên DB tạm qc2_s5 (đã có nền của s12_pe01_constraints.py), EXPLAIN 9 truy vấn AC3.
P(){ docker exec -i ep-r-postgres-1 psql -U edupilot -d qc2_s5 -At "$@"; }
U=00000000-0000-7000-8000-000000000001; C=00000000-0000-7000-8000-0000000000c1
P <<SQL
insert into users(email,full_name,role) select 'qcx'||g||'@x.test','QC'||g,'STUDENT' from generate_series(1,100) g;
insert into question_bank(course_id,type,title,topic,stem,answer_key,review_status,reviewed_by,created_by,difficulty,archived_at)
 select '$C',(array['MCQ_SINGLE','MCQ_MULTI','TRUE_FALSE'])[1+g%3]::question_type,'q'||g,'topic'||(g%40),'stem','[0]',(array['DRAFT','PENDING','APPROVED','REJECTED'])[1+g%4]::question_review_status,
 case when g%4>=2 then '$U'::uuid end,'$U',(array['EASY','MEDIUM','HARD'])[1+g%3]::question_difficulty, null from generate_series(1,20000) g;
insert into exams(course_id,title,status,opens_at,closes_at,duration_minutes,created_by,published_at) select '$C','E'||g,(array['SCHEDULED','OPEN','CLOSED','PUBLISHED','DRAFT'])[1+g%5]::exam_status, now()+ (g||' hour')::interval - interval '30 hour', now()+(g||' hour')::interval-interval '29 hour',30,'$U',case when g%5=3 then now() end from generate_series(1,20) g;
insert into exam_items(course_id,exam_id,question_id,position) select '$C',e.id,'00000000-0000-7000-8000-000000001002',1 from exams e where e.title like 'E%' and not exists (select 1 from exam_items i where i.exam_id=e.id);
insert into exam_attempts(course_id,exam_id,student_id,status,started_at,deadline_at,submitted_at,submit_reason,auto_score,graded_at)
 select '$C',e.id,u.id,case when (row_number() over ())%3=0 then 'IN_PROGRESS' else 'GRADED' end::attempt_status, now()-interval '2 hour', now()+interval '1 hour', case when (row_number() over ())%3=0 then null else now() end, case when (row_number() over ())%3=0 then null else 'MANUAL'::attempt_submit_reason end, case when (row_number() over ())%3=0 then null else 5 end, case when (row_number() over ())%3=0 then null else now() end
 from exams e cross join (select id from users where email like 'qcx%') u where e.title like 'E%';
insert into code_submissions(course_id,exam_id,attempt_id,item_id,problem_id,student_id,kind,language,source,source_sha256,status,verdict,enqueued_at)
 select a.course_id,a.exam_id,a.id,i.id,'00000000-0000-7000-8000-000000001002',a.student_id,'SUBMIT','cpp17','x',repeat('a',64),(case when g%10=0 then 'QUEUED' else 'DONE' end)::submission_status,(case when g%10=0 then null else 'AC' end)::judge_verdict,(case when g%10=0 then null else now() end)
 from (select a.*, row_number() over () rn from exam_attempts a where a.exam_id in (select id from exams where title like 'E%')) a join exam_items i on i.exam_id=a.exam_id and i.question_id='00000000-0000-7000-8000-000000001002' cross join generate_series(1,10) g where true;
analyze;
select 'rows', (select count(*) from question_bank), (select count(*) from exam_attempts), (select count(*) from code_submissions);
SQL
EX(){ n=$1; shift; echo "== $n"; P -c "set enable_seqscan=off" -c "explain (analyze, timing off, summary on) $*" 2>&1 | grep -E 'Scan|Seq|Execution|Planning' | head -6; echo "-- không ép (planner tự chọn):"; P -c "explain (analyze, timing off, summary on) $*" 2>&1 | grep -E 'Scan|Seq|Execution' | head -4; }
EX 1_cau_review "select * from question_bank where course_id='$C' and archived_at is null and review_status='PENDING' order by created_at desc,id desc limit 30"
EX 2_cau_topic "select * from question_bank where course_id='$C' and archived_at is null and topic='topic7' and difficulty='EASY' and type='MCQ_SINGLE'"
EX 3_luot_exam_student "select * from exam_attempts where exam_id=(select id from exams where title='E3') and student_id=(select id from users where email='qcx5@x.test')"
EX 4_luot_dang_lam "select * from exam_attempts where student_id=(select id from users where email='qcx5@x.test') and status='IN_PROGRESS'"
EX 5_luot_den_han "select id from exam_attempts where status='IN_PROGRESS' and deadline_at<now()+interval '2 hour'"
EX 6_bai_nop_cuoi "select * from code_submissions where course_id='$C' and attempt_id=(select id from exam_attempts limit 1) and item_id=(select id from exam_items limit 1) and kind='SUBMIT' order by created_at desc,id desc limit 1"
EX 7_hang_cho "select id from code_submissions where status='QUEUED' and enqueued_at is null and next_attempt_at<=now() order by next_attempt_at,id limit 50"
EX 8_bang_diem "select * from exam_attempts where course_id='$C' and exam_id=(select id from exams where title='E3') and status='GRADED' order by student_id limit 50"
EX 9a_bai_den_han_mo "select id from exams where status='SCHEDULED' and opens_at<=now()"
EX 9b_bai_den_han_dong "select id from exams where status in ('SCHEDULED','OPEN') and closes_at<=now()"
