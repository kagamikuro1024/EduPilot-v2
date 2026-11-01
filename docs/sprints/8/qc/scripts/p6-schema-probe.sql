-- QC US-P6-01 (TC-06…11, 14…16): đẩy dữ liệu xấu vào 00016_gradebook, đọc SQLSTATE. DB trống đã `goose up`. Không đọc mã của dev.
-- Chạy:  docker exec -i qc-p6-pg psql -U edupilot -d qc -v ON_ERROR_STOP=1 < p6-schema-probe.sql    (mỗi dòng NOTICE: PASS|FAIL tên | mong | thật)
\set QUIET on
SET client_min_messages = notice;
BEGIN;
CREATE TEMP TABLE r(name text, want text, got text);
CREATE OR REPLACE FUNCTION pg_temp.try(name text, want text, stmt text) RETURNS void LANGUAGE plpgsql AS $f$
DECLARE got text := 'ok';
BEGIN
  BEGIN EXECUTE stmt; EXCEPTION WHEN OTHERS THEN got := SQLSTATE; END;
  -- mỗi ca chạy trong savepoint ngầm của khối BEGIN…EXCEPTION nên ca lỗi không làm hỏng giao dịch
  INSERT INTO r VALUES (name, want, got);
END $f$;

-- dữ liệu nền: 2 lớp, 1 giảng viên, 3 sinh viên (A, B ở lớp 1; X chỉ ở lớp 2), 1 bài thi mỗi lớp
INSERT INTO users(id,email,full_name,role,student_code) VALUES
 ('00000000-0000-7000-8000-0000000000a1','t@qc','GV','TEACHER',NULL),
 ('00000000-0000-7000-8000-0000000000b1','a@qc','SV A','STUDENT','QC000001'),
 ('00000000-0000-7000-8000-0000000000b2','b@qc','SV B','STUDENT','QC000002'),
 ('00000000-0000-7000-8000-0000000000b3','x@qc','SV X','STUDENT','QC000003');
INSERT INTO courses(id,subject_code,class_code,name,semester,created_by,join_code) VALUES
 ('00000000-0000-7000-8000-0000000000c1','INT','761987','Lop 1','2026-2027-HK1','00000000-0000-7000-8000-0000000000a1','ABCDEFG'),
 ('00000000-0000-7000-8000-0000000000c2','INT','761988','Lop 2','2026-2027-HK1','00000000-0000-7000-8000-0000000000a1','HJKMNPQ');
INSERT INTO enrollments(course_id,user_id,role_in_course,status,joined_via,student_code_snapshot)
SELECT c, u, 'STUDENT','ACTIVE', (SELECT enumlabel::enrollment_joined_via FROM pg_enum e JOIN pg_type t ON t.oid=e.enumtypid WHERE t.typname='enrollment_joined_via' ORDER BY enumsortorder LIMIT 1), s
FROM (VALUES ('00000000-0000-7000-8000-0000000000c1'::uuid,'00000000-0000-7000-8000-0000000000b1'::uuid,'QC000001'),
             ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-0000000000b2','QC000002'),
             ('00000000-0000-7000-8000-0000000000c2','00000000-0000-7000-8000-0000000000b3','QC000003')) v(c,u,s);
INSERT INTO exams(id,course_id,title,created_by,status,published_at,opens_at,closes_at,duration_minutes) VALUES
 ('00000000-0000-7000-8000-0000000000e1','00000000-0000-7000-8000-0000000000c1','Thi lop 1','00000000-0000-7000-8000-0000000000a1','PUBLISHED',now(),now()-interval '2 day',now()-interval '1 day',30),
 ('00000000-0000-7000-8000-0000000000e2','00000000-0000-7000-8000-0000000000c2','Thi lop 2','00000000-0000-7000-8000-0000000000a1','PUBLISHED',now(),now()-interval '2 day',now()-interval '1 day',30);

-- ---- TC-06 một DRAFT / một CONFIRMED mỗi lớp
INSERT INTO grade_schemes(id,course_id,scheme_no,status,created_by) VALUES ('00000000-0000-7000-8000-000000000501','00000000-0000-7000-8000-0000000000c1',1,'DRAFT','00000000-0000-7000-8000-0000000000a1');
SELECT pg_temp.try('06a hai DRAFT cùng lớp','23505',$$INSERT INTO grade_schemes(course_id,scheme_no,status,created_by) VALUES ('00000000-0000-7000-8000-0000000000c1',2,'DRAFT','00000000-0000-7000-8000-0000000000a1')$$);
SELECT pg_temp.try('06b CONFIRMED đầu','ok',$$INSERT INTO grade_schemes(id,course_id,scheme_no,status,confirmed_at,confirmed_by,created_by) VALUES ('00000000-0000-7000-8000-000000000502','00000000-0000-7000-8000-0000000000c1',3,'CONFIRMED',now(),'00000000-0000-7000-8000-0000000000a1','00000000-0000-7000-8000-0000000000a1')$$);
SELECT pg_temp.try('06c hai CONFIRMED','23505',$$INSERT INTO grade_schemes(course_id,scheme_no,status,confirmed_at,confirmed_by,created_by) VALUES ('00000000-0000-7000-8000-0000000000c1',4,'CONFIRMED',now(),'00000000-0000-7000-8000-0000000000a1','00000000-0000-7000-8000-0000000000a1')$$);
SELECT pg_temp.try('06d nhiều SUPERSEDED','ok',$$INSERT INTO grade_schemes(course_id,scheme_no,status,confirmed_at,confirmed_by,created_by) VALUES ('00000000-0000-7000-8000-0000000000c1',5,'SUPERSEDED',now(),'00000000-0000-7000-8000-0000000000a1','00000000-0000-7000-8000-0000000000a1'),('00000000-0000-7000-8000-0000000000c1',6,'SUPERSEDED',now(),'00000000-0000-7000-8000-0000000000a1','00000000-0000-7000-8000-0000000000a1')$$);
SELECT pg_temp.try('06e scheme lớp 2','ok',$$INSERT INTO grade_schemes(id,course_id,scheme_no,status,created_by) VALUES ('00000000-0000-7000-8000-000000000511','00000000-0000-7000-8000-0000000000c2',1,'DRAFT','00000000-0000-7000-8000-0000000000a1')$$);

-- ---- TC-07/08 trọng số
INSERT INTO grade_components(id,course_id,scheme_id,key,name,grp,weight) VALUES ('00000000-0000-7000-8000-000000000601','00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','midterm','GK','QT',20);
SELECT pg_temp.try('07a weight -0.01','23514',$$INSERT INTO grade_components(course_id,scheme_id,key,name,grp,weight) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','k1','x','QT',-0.01)$$);
SELECT pg_temp.try('07b weight 100.01','23514',$$INSERT INTO grade_components(course_id,scheme_id,key,name,grp,weight) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','k2','x','QT',100.01)$$);
SELECT pg_temp.try('07c weight 0','ok',$$INSERT INTO grade_components(course_id,scheme_id,key,name,grp,weight) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','k3','x','QT',0)$$);
SELECT pg_temp.try('07d weight 100','ok',$$INSERT INTO grade_components(course_id,scheme_id,key,name,grp,weight) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','k4','x','CK',100)$$);
SELECT pg_temp.try('08 (scheme,key) trùng','23505',$$INSERT INTO grade_components(course_id,scheme_id,key,name,grp,weight) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','midterm','x','QT',5)$$);
SELECT pg_temp.try('08b component gắn scheme lớp khác','23503',$$INSERT INTO grade_components(course_id,scheme_id,key,name,grp,weight) VALUES ('00000000-0000-7000-8000-0000000000c2','00000000-0000-7000-8000-000000000502','zz','x','QT',5)$$);

-- ---- TC-09 grade_items
SELECT pg_temp.try('09a EXAM thiếu exam_id','23514',$$INSERT INTO grade_items(course_id,scheme_id,component_id,key,name,source) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','00000000-0000-7000-8000-000000000601','i1','x','EXAM')$$);
SELECT pg_temp.try('09b MANUAL có exam_id','23514',$$INSERT INTO grade_items(course_id,scheme_id,component_id,key,name,source,exam_id) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','00000000-0000-7000-8000-000000000601','i2','x','MANUAL','00000000-0000-7000-8000-0000000000e1')$$);
SELECT pg_temp.try('09c EXAM hợp lệ','ok',$$INSERT INTO grade_items(id,course_id,scheme_id,component_id,key,name,source,exam_id) VALUES ('00000000-0000-7000-8000-000000000701','00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','00000000-0000-7000-8000-000000000601','i3','x','EXAM','00000000-0000-7000-8000-0000000000e1')$$);
SELECT pg_temp.try('09d max_score 0','23514',$$INSERT INTO grade_items(course_id,scheme_id,component_id,key,name,max_score) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','00000000-0000-7000-8000-000000000601','i4','x',0)$$);
SELECT pg_temp.try('09e max_score -1','23514',$$INSERT INTO grade_items(course_id,scheme_id,component_id,key,name,max_score) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','00000000-0000-7000-8000-000000000601','i5','x',-1)$$);
INSERT INTO grade_items(id,course_id,scheme_id,component_id,key,name) VALUES ('00000000-0000-7000-8000-000000000702','00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000502','00000000-0000-7000-8000-000000000601','m1','Tay');

-- ---- TC-10 grade_entries
SELECT pg_temp.try('10a score -0.01','23514',$$INSERT INTO grade_entries(course_id,item_id,student_id,score,entered_by) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000702','00000000-0000-7000-8000-0000000000b1',-0.01,'00000000-0000-7000-8000-0000000000a1')$$);
SELECT pg_temp.try('10b score 0','ok',$$INSERT INTO grade_entries(course_id,item_id,student_id,score,entered_by) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000702','00000000-0000-7000-8000-0000000000b1',0,'00000000-0000-7000-8000-0000000000a1')$$);
SELECT pg_temp.try('10c trùng (item,student)','23505',$$INSERT INTO grade_entries(course_id,item_id,student_id,score,entered_by) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000702','00000000-0000-7000-8000-0000000000b1',5,'00000000-0000-7000-8000-0000000000a1')$$);

-- ---- TC-11 FK phức hợp chéo lớp
SELECT pg_temp.try('11a entry lớp2 → item lớp1','23503',$$INSERT INTO grade_entries(course_id,item_id,student_id,score,entered_by) VALUES ('00000000-0000-7000-8000-0000000000c2','00000000-0000-7000-8000-000000000702','00000000-0000-7000-8000-0000000000b3',5,'00000000-0000-7000-8000-0000000000a1')$$);
SELECT pg_temp.try('11b item lớp2 → exam lớp1','23503',$$INSERT INTO grade_items(course_id,scheme_id,component_id,key,name,source,exam_id) VALUES ('00000000-0000-7000-8000-0000000000c2','00000000-0000-7000-8000-000000000511','00000000-0000-7000-8000-000000000601','z1','x','EXAM','00000000-0000-7000-8000-0000000000e1')$$);
SELECT pg_temp.try('11c item lớp2 → scheme lớp1','23503',$$INSERT INTO grade_items(course_id,scheme_id,component_id,key,name) VALUES ('00000000-0000-7000-8000-0000000000c2','00000000-0000-7000-8000-000000000502','00000000-0000-7000-8000-000000000601','z2','x')$$);
SELECT pg_temp.try('11d entry lớp1 → sv chỉ ở lớp2','23503',$$INSERT INTO grade_entries(course_id,item_id,student_id,score,entered_by) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-000000000702','00000000-0000-7000-8000-0000000000b3',5,'00000000-0000-7000-8000-0000000000a1')$$);

-- ---- TC-14/16 snapshot bất biến
INSERT INTO final_grade_snapshots(id,course_id,student_id,version,scheme_id,breakdown,numeric_grade,letter,locked_by) VALUES ('00000000-0000-7000-8000-000000000801','00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-0000000000b1',1,'00000000-0000-7000-8000-000000000502','{}',7,'B','00000000-0000-7000-8000-0000000000a1');
SELECT pg_temp.try('14a UPDATE snapshot','P0001',$$UPDATE final_grade_snapshots SET numeric_grade=1$$);
SELECT pg_temp.try('14b DELETE snapshot','P0001',$$DELETE FROM final_grade_snapshots$$);
SELECT pg_temp.try('14c TRUNCATE snapshot','P0001',$$TRUNCATE final_grade_snapshots$$);
SELECT pg_temp.try('16a trùng (course,student,version)','23505',$$INSERT INTO final_grade_snapshots(course_id,student_id,version,scheme_id,breakdown,numeric_grade,letter,locked_by) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-0000000000b1',1,'00000000-0000-7000-8000-000000000502','{}',8,'B+','00000000-0000-7000-8000-0000000000a1')$$);
SELECT pg_temp.try('16b version mới','ok',$$INSERT INTO final_grade_snapshots(course_id,student_id,version,scheme_id,breakdown,numeric_grade,letter,locked_by) VALUES ('00000000-0000-7000-8000-0000000000c1','00000000-0000-7000-8000-0000000000b1',2,'00000000-0000-7000-8000-000000000502','{}',8,'B+','00000000-0000-7000-8000-0000000000a1')$$);
SELECT pg_temp.try('16c UPDATE bản mới vẫn chặn','P0001',$$UPDATE final_grade_snapshots SET letter='A' WHERE version=2$$);

SELECT CASE WHEN want=got THEN 'PASS' ELSE 'FAIL' END AS kq, name, want AS mong, got AS that FROM r ORDER BY name;
-- TC-15: trigger mức dòng và mức câu lệnh BEFORE TRUNCATE
SELECT 'TC-15 trigger' AS ten, tgname, (tgtype & 1)::int AS row_level, (tgtype & 32)::int AS truncate_bit, (tgtype & 2)::int AS before_bit FROM pg_trigger WHERE tgrelid='final_grade_snapshots'::regclass AND NOT tgisinternal;
ROLLBACK;
