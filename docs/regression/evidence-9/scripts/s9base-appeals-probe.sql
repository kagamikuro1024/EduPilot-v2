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
 ('00000000-0000-7000-8000-0000000000c1','INT','QPRB1','Lop 1','2026-2027-HK1','00000000-0000-7000-8000-0000000000a1','ABCDEFG'),
 ('00000000-0000-7000-8000-0000000000c2','INT','QPRB2','Lop 2','2026-2027-HK1','00000000-0000-7000-8000-0000000000a1','HJKMNPQ');
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
INSERT INTO assignments(id,course_id,title,due_at,submission_code,created_by) VALUES ('00000000-0000-7000-8000-0000000000d1','00000000-0000-7000-8000-0000000000c1','BT03',now(),'BT03','00000000-0000-7000-8000-0000000000a1'),
 ('00000000-0000-7000-8000-0000000000d2','00000000-0000-7000-8000-0000000000c2','BT03',now(),'BT03','00000000-0000-7000-8000-0000000000a1');
INSERT INTO submissions(id,course_id,source,blob_key,file_name,file_size,file_mime,sha256,assignment_id,student_id,match_status,match_method,version) VALUES ('00000000-0000-7000-8000-000000000e01','00000000-0000-7000-8000-0000000000c1','IN_APP','k','f',1,'text/plain','s','00000000-0000-7000-8000-0000000000d1','00000000-0000-7000-8000-0000000000b2','MATCHED','IN_APP',1);
;
INSERT INTO grading_results(id,course_id,submission_id,run_state,status) VALUES ('00000000-0000-7000-8000-000000000f01','00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000e01','DONE','DRAFT');
INSERT INTO submissions(id,course_id,source,blob_key,file_name,file_size,file_mime,sha256,assignment_id,student_id,match_status,match_method,version) VALUES ('00000000-0000-7000-8000-000000000e02','00000000-0000-7000-8000-0000000000c2','IN_APP','k','f',1,'text/plain','s','00000000-0000-7000-8000-0000000000d2','00000000-0000-7000-8000-0000000000b3','MATCHED','IN_APP',1);
INSERT INTO grading_results(id,course_id,submission_id,run_state,status) VALUES ('00000000-0000-7000-8000-000000000f02','00000000-0000-7000-8000-0000000000c2','00000000-0000-7000-8000-000000000e02','DONE','DRAFT');
SELECT pg_temp.try('07 phúc khảo hợp lệ lần 1','ok',$$INSERT INTO grade_appeals(course_id,assignment_id,student_id,submission_id,result_id,criteria_ids,reason,score_before) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-0000000000d1','00000000-0000-7000-8000-0000000000b2','00000000-0000-7000-8000-000000000e01','00000000-0000-7000-8000-000000000f01','{c1}','lý do',5)$$);
SELECT pg_temp.try('07 hai phúc khảo cùng (lớp,bài,sv)','23505',$$INSERT INTO grade_appeals(course_id,assignment_id,student_id,submission_id,result_id,criteria_ids,reason,score_before) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-0000000000d1','00000000-0000-7000-8000-0000000000b2','00000000-0000-7000-8000-000000000e01','00000000-0000-7000-8000-000000000f01','{c1}','lý do 2',5)$$);
SELECT pg_temp.try('07 result_id của lớp khác (FK phức hợp)','23503',$$INSERT INTO grade_appeals(course_id,assignment_id,student_id,submission_id,result_id,criteria_ids,reason,score_before) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-0000000000d1','00000000-0000-7000-8000-0000000000b1','00000000-0000-7000-8000-000000000e01','00000000-0000-7000-8000-000000000f02','{c1}','lý do',5)$$);
SELECT pg_temp.try('07 status lạ','23514',$$INSERT INTO grade_appeals(course_id,assignment_id,student_id,submission_id,result_id,criteria_ids,reason,score_before,status) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-0000000000d1','00000000-0000-7000-8000-0000000000b1','00000000-0000-7000-8000-000000000e01','00000000-0000-7000-8000-000000000f01','{c1}','x',5,'WAT')$$);
SELECT CASE WHEN want = got THEN 'PASS' ELSE 'FAIL' END AS kq, name, want, got, cons FROM r ORDER BY 1 DESC, name;
ROLLBACK;
