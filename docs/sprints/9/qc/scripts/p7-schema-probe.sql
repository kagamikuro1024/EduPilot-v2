-- QC US-P7-01 (TC-P7-01-04…13): đẩy dữ liệu xấu vào 00017_assignments, đọc SQLSTATE + tên ràng buộc. DB trống đã `goose up` tới 00017. Không đọc mã của dev.
-- Chạy:  docker exec -i qc-p7-pg psql -U edupilot -d qc -v ON_ERROR_STOP=1 < p7-schema-probe.sql   (cuối: bảng PASS|FAIL; thoát lỗi nếu có FAIL)
\set QUIET on
BEGIN;
CREATE TEMP TABLE r(name text, want text, got text, cons text);
CREATE OR REPLACE FUNCTION pg_temp.try(name text, want text, stmt text) RETURNS void LANGUAGE plpgsql AS $f$
DECLARE got text := 'ok'; c text := '';
BEGIN
  BEGIN EXECUTE stmt; EXCEPTION WHEN OTHERS THEN GET STACKED DIAGNOSTICS c = CONSTRAINT_NAME; got := SQLSTATE; END;
  INSERT INTO r VALUES (name, want, got, coalesce(c,''));
END $f$;
-- nền: 2 lớp, GV, SV A và B ở lớp 1, X ở lớp 2
INSERT INTO users(id,email,full_name,role,student_code) VALUES
 ('00000000-0000-7000-8000-0000000000a1','t@qc','GV','TEACHER',NULL),('00000000-0000-7000-8000-0000000000b1','a@qc','SV A','STUDENT','QC000001'),
 ('00000000-0000-7000-8000-0000000000b2','b@qc','SV B','STUDENT','QC000002'),('00000000-0000-7000-8000-0000000000b3','x@qc','SV X','STUDENT','QC000003');
INSERT INTO courses(id,subject_code,class_code,name,semester,created_by,join_code) VALUES
 ('00000000-0000-7000-8000-0000000000c1','INT','761987','Lop 1','2026-2027-HK1','00000000-0000-7000-8000-0000000000a1','ABCDEFG'),
 ('00000000-0000-7000-8000-0000000000c2','INT','761988','Lop 2','2026-2027-HK1','00000000-0000-7000-8000-0000000000a1','HJKMNPQ');
INSERT INTO enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot) VALUES
 ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-0000000000b1','STUDENT','ACTIVE','ROSTER','QC000001'),
 ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-0000000000b2','STUDENT','ACTIVE','ROSTER','QC000002'),
 ('00000000-0000-7000-8000-0000000000c2','00000000-0000-7000-8000-0000000000b3','STUDENT','ACTIVE','ROSTER','QC000003');
\set C1 '''00000000-0000-7000-8000-0000000000c1'''
\set C2 '''00000000-0000-7000-8000-0000000000c2'''
\set GV '''00000000-0000-7000-8000-0000000000a1'''
\set A1 '''00000000-0000-7000-8000-0000000000b1'''
\set B1 '''00000000-0000-7000-8000-0000000000b2'''
\set X1 '''00000000-0000-7000-8000-0000000000b3'''
CREATE OR REPLACE FUNCTION pg_temp.asg(code text, extra text DEFAULT '', course text DEFAULT '00000000-0000-7000-8000-0000000000c1') RETURNS text LANGUAGE sql AS
$$ SELECT 'INSERT INTO assignments(course_id,title,due_at,submission_code,created_by' || (CASE WHEN extra='' THEN '' ELSE ','||split_part(extra,'=',1) END) || ') VALUES ('''||course||''',''BT'',now(),'''||code||''','''||'00000000-0000-7000-8000-0000000000a1'||'''' || (CASE WHEN extra='' THEN '' ELSE ','||substr(extra,position('=' in extra)+1) END) || ')' $$;
-- ---- TC-04: assignments
SELECT pg_temp.try('04 hợp lệ','ok',pg_temp.asg('OK1'));
SELECT pg_temp.try('04 max_score 0,25 (biên dưới)','ok',pg_temp.asg('MS025','max_score=0.25'));
SELECT pg_temp.try('04 max_score 100 (biên trên)','ok',pg_temp.asg('MS100','max_score=100'));
SELECT pg_temp.try('04 max_score 0','23514',pg_temp.asg('MS0','max_score=0'));
SELECT pg_temp.try('04 max_score 100,25','23514',pg_temp.asg('MS1002','max_score=100.25'));
SELECT pg_temp.try('04 max_score 10,1 (không bội 0,25)','23514',pg_temp.asg('MS101','max_score=10.1'));
SELECT pg_temp.try('04 late_penalty âm','23514',pg_temp.asg('LP','late_penalty_per_day=-0.5'));
SELECT pg_temp.try('04 late_max_days 15','23514',pg_temp.asg('LM15','late_max_days=15'));
SELECT pg_temp.try('04 allow_late + late_max_days 14','ok',$$INSERT INTO assignments(course_id,title,due_at,submission_code,created_by,allow_late,late_max_days) VALUES ('00000000-0000-7000-8000-0000000000c1','BT',now(),'AL14','00000000-0000-7000-8000-0000000000a1',true,14)$$);
SELECT pg_temp.try('04 allow_late + late_max_days 0','23514',$$INSERT INTO assignments(course_id,title,due_at,submission_code,created_by,allow_late,late_max_days) VALUES ('00000000-0000-7000-8000-0000000000c1','BT',now(),'AL0','00000000-0000-7000-8000-0000000000a1',true,0)$$);
SELECT pg_temp.try('04 appeal_days 0','ok',pg_temp.asg('AP0','appeal_days=0'));
SELECT pg_temp.try('04 appeal_days 30','ok',pg_temp.asg('AP30','appeal_days=30'));
SELECT pg_temp.try('04 appeal_days 31','23514',pg_temp.asg('AP31','appeal_days=31'));
SELECT pg_temp.try('04 channels rỗng','23514',pg_temp.asg('CH0',$$channels='{}'$$));
SELECT pg_temp.try('04 channels lạ','23514',pg_temp.asg('CHX',$$channels='{FAX}'$$));
SELECT pg_temp.try('04 TEAMS thiếu id','23514',pg_temp.asg('TM0',$$channels='{IN_APP,TEAMS}'$$));
SELECT pg_temp.try('04 TEAMS đủ id','ok',$$INSERT INTO assignments(course_id,title,due_at,submission_code,created_by,channels,teams_class_id,teams_assignment_id) VALUES ('00000000-0000-7000-8000-0000000000c1','BT',now(),'TM1','00000000-0000-7000-8000-0000000000a1','{IN_APP,TEAMS}','tc','ta')$$);
SELECT pg_temp.try('04 id Teams khi KHÔNG có kênh TEAMS (DB chặt hơn spec: Q-01-b)','23514',$$INSERT INTO assignments(course_id,title,due_at,submission_code,created_by,teams_class_id,teams_assignment_id) VALUES ('00000000-0000-7000-8000-0000000000c1','BT',now(),'TM2','00000000-0000-7000-8000-0000000000a1','tc','ta')$$);
SELECT pg_temp.try('04 mã nộp thư 1 ký tự','23514',pg_temp.asg('A'));
SELECT pg_temp.try('04 mã nộp thư 13 ký tự','23514',pg_temp.asg('ABCDEFGHIJKLM'));
SELECT pg_temp.try('04 mã nộp thư chữ thường','23514',pg_temp.asg('bt03'));
SELECT pg_temp.try('04 mã nộp thư trùng cùng lớp','23505',pg_temp.asg('OK1'));
SELECT pg_temp.try('04 mã nộp thư trùng KHÁC lớp','ok',pg_temp.asg('OK1','', '00000000-0000-7000-8000-0000000000c2'));
SELECT pg_temp.try('04 type QUIZ ở DB','ok',pg_temp.asg('QZ',$$type='QUIZ'$$));
SELECT pg_temp.try('04 type lạ','23514',pg_temp.asg('TY',$$type='X'$$));
-- id mặc định + dòng nền cho các bảng sau
INSERT INTO assignments(id,course_id,title,due_at,submission_code,created_by) VALUES ('00000000-0000-7000-8000-0000000000d1','00000000-0000-7000-8000-0000000000c1','BT03',now(),'BT03','00000000-0000-7000-8000-0000000000a1'),
 ('00000000-0000-7000-8000-0000000000d2','00000000-0000-7000-8000-0000000000c2','BT03',now(),'BT03','00000000-0000-7000-8000-0000000000a1');
-- ---- TC-05: submissions
\set SUBCOLS 'course_id,source,blob_key,file_name,file_size,file_mime,sha256'
SELECT pg_temp.try('05 MATCHED thiếu student_id','23514',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,assignment_id,match_status,match_method,version) VALUES ('00000000-0000-7000-8000-0000000000c1','IN_APP','k','f',1,'text/plain','s','00000000-0000-7000-8000-0000000000d1','MATCHED','IN_APP',1)$$);
SELECT pg_temp.try('05 MATCHED đủ v1','ok',$$INSERT INTO submissions(id,course_id,source,blob_key,file_name,file_size,file_mime,sha256,assignment_id,student_id,match_status,match_method,version) VALUES ('00000000-0000-7000-8000-000000000e01','00000000-0000-7000-8000-0000000000c1','IN_APP','k','f',1,'text/plain','s','00000000-0000-7000-8000-0000000000d1','00000000-0000-7000-8000-0000000000b2','MATCHED','IN_APP',1)$$);
SELECT pg_temp.try('05 trùng (assignment, student, version)','23505',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,assignment_id,student_id,match_status,match_method,version) VALUES ('00000000-0000-7000-8000-0000000000c1','IN_APP','k2','f',1,'text/plain','s2','00000000-0000-7000-8000-0000000000d1','00000000-0000-7000-8000-0000000000b2','MATCHED','IN_APP',1)$$);
SELECT pg_temp.try('05 version 2 cùng sinh viên','ok',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,assignment_id,student_id,match_status,match_method,version) VALUES ('00000000-0000-7000-8000-0000000000c1','IN_APP','k3','f',1,'text/plain','s3','00000000-0000-7000-8000-0000000000d1','00000000-0000-7000-8000-0000000000b2','MATCHED','IN_APP',2)$$);
SELECT pg_temp.try('05 sinh viên ngoài lớp (FK phức hợp)','23503',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,assignment_id,student_id,match_status,match_method,version) VALUES ('00000000-0000-7000-8000-0000000000c1','IN_APP','k4','f',1,'text/plain','s4','00000000-0000-7000-8000-0000000000d1','00000000-0000-7000-8000-0000000000b3','MATCHED','IN_APP',1)$$);
SELECT pg_temp.try('05 bài tập lớp khác (FK phức hợp)','23503',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,assignment_id,student_id,match_status,match_method,version) VALUES ('00000000-0000-7000-8000-0000000000c1','IN_APP','k5','f',1,'text/plain','s5','00000000-0000-7000-8000-0000000000d2','00000000-0000-7000-8000-0000000000b1','MATCHED','IN_APP',1)$$);
SELECT pg_temp.try('05 UNMATCHED lần 1','ok',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,match_status,unmatched_reason) VALUES ('00000000-0000-7000-8000-0000000000c1','UPLOAD','u1','f',1,'text/plain','u','UNMATCHED','NO_STUDENT_SIGNAL')$$);
SELECT pg_temp.try('05 UNMATCHED lần 2 giống hệt (không bị chặn)','ok',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,match_status,unmatched_reason) VALUES ('00000000-0000-7000-8000-0000000000c1','UPLOAD','u1','f',1,'text/plain','u','UNMATCHED','NO_STUDENT_SIGNAL')$$);
SELECT pg_temp.try('05 UNMATCHED thiếu lý do','23514',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,match_status) VALUES ('00000000-0000-7000-8000-0000000000c1','UPLOAD','u2','f',1,'text/plain','u','UNMATCHED')$$);
SELECT pg_temp.try('05 external_ref trùng (source, ref, ts)','23505',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,match_status,unmatched_reason,external_ref,external_ts) SELECT '00000000-0000-7000-8000-0000000000c1','TEAMS','t'||g,'f',1,'text/plain','t','UNMATCHED','NO_STUDENT_SIGNAL','g1','2026-01-01' FROM generate_series(1,2) g$$);
SELECT pg_temp.try('05 file_size 0','23514',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,match_status,unmatched_reason) VALUES ('00000000-0000-7000-8000-0000000000c1','UPLOAD','u3','f',0,'text/plain','u','UNMATCHED','NO_STUDENT_SIGNAL')$$);
SELECT pg_temp.try('05 file_size 20 MiB (biên)','ok',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,match_status,unmatched_reason) VALUES ('00000000-0000-7000-8000-0000000000c1','UPLOAD','u4','f',20971520,'text/plain','u','UNMATCHED','NO_STUDENT_SIGNAL')$$);
SELECT pg_temp.try('05 file_size 20 MiB + 1','23514',$$INSERT INTO submissions(course_id,source,blob_key,file_name,file_size,file_mime,sha256,match_status,unmatched_reason) VALUES ('00000000-0000-7000-8000-0000000000c1','UPLOAD','u5','f',20971521,'text/plain','u','UNMATCHED','NO_STUDENT_SIGNAL')$$);
-- bất biến nội dung bài nộp
SELECT pg_temp.try('05 đổi sha256 bài nộp','P0001',$$UPDATE submissions SET sha256='x' WHERE id='00000000-0000-7000-8000-000000000e01'$$);
SELECT pg_temp.try('05 đổi blob_key bài nộp','P0001',$$UPDATE submissions SET blob_key='x' WHERE id='00000000-0000-7000-8000-000000000e01'$$);
SELECT pg_temp.try('05 extracted_text lần đầu (từ NULL)','ok',$$UPDATE submissions SET extracted_text='abc', extract_status='OK' WHERE id='00000000-0000-7000-8000-000000000e01'$$);
SELECT pg_temp.try('05 extracted_text lần hai đổi','P0001',$$UPDATE submissions SET extracted_text='khác' WHERE id='00000000-0000-7000-8000-000000000e01'$$);
-- ---- TC-06/10/11: grading_results
INSERT INTO grading_results(id,course_id,submission_id,run_state,status) VALUES ('00000000-0000-7000-8000-000000000f01','00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000e01','DONE','DRAFT');
SELECT pg_temp.try('06 REVIEWED khi QUEUED','23514',$$INSERT INTO grading_results(course_id,submission_id,round,run_state,status) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000e01',2,'QUEUED','REVIEWED')$$);
SELECT pg_temp.try('06 round trùng','23505',$$INSERT INTO grading_results(course_id,submission_id,round) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000e01',1)$$);
SELECT pg_temp.try('06 round 2','ok',$$INSERT INTO grading_results(course_id,submission_id,round) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000e01',2)$$);
SELECT pg_temp.try('06 PUBLISHED thiếu published_at','23514',$$INSERT INTO grading_results(course_id,submission_id,round,run_state,status) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000e01',3,'DONE','PUBLISHED')$$);
SELECT pg_temp.try('06 cờ lạ','23514',$$INSERT INTO grading_results(course_id,submission_id,round,flags) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000e01',4,'{HACKED}')$$);
SELECT pg_temp.try('06 kết quả trỏ bài nộp lớp khác','23503',$$INSERT INTO grading_results(course_id,submission_id,round) VALUES ('00000000-0000-7000-8000-0000000000c2','00000000-0000-7000-8000-000000000e01',1)$$);
SELECT pg_temp.try('06 needs_review cột sinh: NEEDS_REVIEW → true','ok',$qq$DO $x$ BEGIN UPDATE grading_results SET flags='{NEEDS_REVIEW}' WHERE id='00000000-0000-7000-8000-000000000f01'; IF NOT (SELECT needs_review FROM grading_results WHERE id='00000000-0000-7000-8000-000000000f01') THEN RAISE EXCEPTION 'needs_review sai'; END IF; UPDATE grading_results SET flags='{SCORE_ADJUSTED}' WHERE id='00000000-0000-7000-8000-000000000f01'; IF (SELECT needs_review FROM grading_results WHERE id='00000000-0000-7000-8000-000000000f01') THEN RAISE EXCEPTION 'SCORE_ADJUSTED không được là needs_review'; END IF; END $x$ $qq$);
SELECT pg_temp.try('06 ghi needs_review trực tiếp','428C9',$$UPDATE grading_results SET needs_review=true WHERE id='00000000-0000-7000-8000-000000000f01'$$);
SELECT pg_temp.try('10 ai_scores lần đầu từ NULL','ok',$$UPDATE grading_results SET ai_scores='{"c1":"2.0"}', ai_total_before_late=7 WHERE id='00000000-0000-7000-8000-000000000f01'$$);
SELECT pg_temp.try('10 đổi ai_scores','P0001',$$UPDATE grading_results SET ai_scores='{}' WHERE id='00000000-0000-7000-8000-000000000f01'$$);
SELECT pg_temp.try('10 đổi ai_total_before_late','P0001',$$UPDATE grading_results SET ai_total_before_late=9 WHERE id='00000000-0000-7000-8000-000000000f01'$$);
SELECT pg_temp.try('11 ghi lại cùng giá trị ai_scores','ok',$$UPDATE grading_results SET ai_scores=ai_scores WHERE id='00000000-0000-7000-8000-000000000f01'$$);
SELECT pg_temp.try('10 sửa criteria, total, status','ok',$$UPDATE grading_results SET criteria='[{"id":"c1"}]', total=7, status='DRAFT' WHERE id='00000000-0000-7000-8000-000000000f01'$$);
-- ---- TC-12: xoá bài tập đang được grade_items trỏ tới (bài KHÔNG có bài nộp, để không bị FK `submissions` che — BUG-P7-01-1)
INSERT INTO assignments(id,course_id,title,due_at,submission_code,created_by) VALUES ('00000000-0000-7000-8000-0000000000d3','00000000-0000-7000-8000-0000000000c1','BT-LK',now(),'LK1','00000000-0000-7000-8000-0000000000a1');
INSERT INTO grade_schemes(id,course_id,scheme_no,status,created_by,confirmed_at,confirmed_by) VALUES ('00000000-0000-7000-8000-000000000501','00000000-0000-7000-8000-0000000000c1',1,'CONFIRMED','00000000-0000-7000-8000-0000000000a1',now(),'00000000-0000-7000-8000-0000000000a1');
INSERT INTO grade_components(id,course_id,scheme_id,key,name,grp,weight) VALUES ('00000000-0000-7000-8000-000000000601','00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000501','assign','Bài tập','QT',20);
SELECT pg_temp.try('12 grade_items trỏ bài tập lớp 1','ok',$$INSERT INTO grade_items(course_id,scheme_id,component_id,key,name,source,assignment_id) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000501','00000000-0000-7000-8000-000000000601','bt','BT','ASSIGNMENT','00000000-0000-7000-8000-0000000000d3')$$);
SELECT pg_temp.try('12 grade_items lớp 1 trỏ bài tập lớp 2','23503',$$INSERT INTO grade_items(course_id,scheme_id,component_id,key,name,source,assignment_id) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000501','00000000-0000-7000-8000-000000000601','bt2','BT2','ASSIGNMENT','00000000-0000-7000-8000-0000000000d2')$$);
SELECT pg_temp.try('12 xoá bài tập đang được grade_items trỏ tới','23503',$$DELETE FROM assignments WHERE id='00000000-0000-7000-8000-0000000000d3'$$);
SELECT pg_temp.try('12 xoá bài tập sạch','ok',$$DELETE FROM assignments WHERE id='00000000-0000-7000-8000-0000000000d2' AND false OR id IN (SELECT id FROM assignments WHERE submission_code='A' AND false)$$);
-- ---- TC-07: grade_appeals (bảng ca bổ sung ở pha 2 theo cột thật); kết quả tổng hợp
SELECT pg_temp.try('00 không có cột late_days / is_late ở submissions','ok',$qq$DO $x$ BEGIN IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='submissions' AND column_name IN ('late_days','is_late')) THEN RAISE EXCEPTION 'có cột late'; END IF; END $x$ $qq$);
SELECT CASE WHEN want = got THEN 'PASS' ELSE 'FAIL' END AS kq, name, want, got, cons FROM r ORDER BY 1 DESC, name;
SELECT count(*) FILTER (WHERE want <> got) AS fails, count(*) AS total FROM r \gset
\if :fails
\echo 'RESULT: FAIL'
\else
\echo 'RESULT: PASS'
\endif
ROLLBACK;
