#!/bin/bash
# r91-fixseed.sh <MÃ_LỚP>: đưa điểm danh + phát biểu của lớp sao về bản seed gốc (smkpr = /tmp/smk91-pristine.dump): 761987 đã bị QC ghi thêm (Muộn/Vắng/Vắng phép/phát biểu)
source /tmp/ep-r-lib.sh
SQ(){ docker exec -i ep-r-postgres-1 sh -c "psql -U edupilot -d $1 -At -F '|' -c \"$2\"; exit 0"; }
SQ smkpr "select e.student_code_snapshot||'|'||s.session_no||'|'||a.status from attendance_records a join class_sessions s on s.id=a.session_id join enrollments e on e.user_id=a.student_id and e.course_id=a.course_id and e.role_in_course='STUDENT' where a.course_id=(select id from courses where class_code='761987')" > /tmp/smks/pristine-att.txt
SQ smkpr "select e.student_code_snapshot||'|'||s.session_no||'|'||p.type||'|'||p.points||'|'||coalesce(p.note,'') from participation_events p join class_sessions s on s.id=p.session_id join enrollments e on e.user_id=p.student_id and e.course_id=p.course_id and e.role_in_course='STUDENT' where p.course_id=(select id from courses where class_code='761987')" > /tmp/smks/pristine-par.txt
wc -l /tmp/smks/pristine-att.txt /tmp/smks/pristine-par.txt | head -2
CC=$1
{ cat <<SQL
begin;
create temp table pa(code text, sn int, st text);
copy pa from stdin with (format csv, delimiter '|');
SQL
cat /tmp/smks/pristine-att.txt; echo '\.'
cat <<SQL
create temp table pp(code text, sn int, ty text, pts numeric, note text);
copy pp from stdin with (format csv, delimiter '|');
SQL
cat /tmp/smks/pristine-par.txt; echo '\.'
cat <<SQL
create temp table cx as select (select id from courses where class_code='$CC') cid;
update attendance_records ar set status=pa.st::attendance_status from pa, class_sessions s, enrollments en, cx where s.id=ar.session_id and s.session_no=pa.sn and en.user_id=ar.student_id and en.course_id=ar.course_id and en.student_code_snapshot=pa.code and ar.course_id=cx.cid and ar.status::text<>pa.st;
delete from attendance_records ar using cx where ar.course_id=cx.cid and not exists (select 1 from pa, class_sessions s, enrollments en where s.id=ar.session_id and s.session_no=pa.sn and en.user_id=ar.student_id and en.course_id=ar.course_id and en.student_code_snapshot=pa.code);
delete from participation_events pe using cx where pe.course_id=cx.cid;
insert into participation_events (course_id, student_id, session_id, type, points, note, client_event_id, created_by)
 select cx.cid, en.user_id, s.id, pp.ty::participation_type, pp.pts, nullif(pp.note,''), gen_random_uuid(), (select user_id from enrollments where course_id=cx.cid and role_in_course='TEACHER' limit 1)
 from pp, cx, enrollments en, class_sessions s where en.course_id=cx.cid and en.student_code_snapshot=pp.code and en.role_in_course='STUDENT' and s.course_id=cx.cid and s.session_no=pp.sn;
commit;
SQL
} | docker exec -i ep-r-postgres-1 sh -c 'psql -U edupilot -d edupilot -q -v ON_ERROR_STOP=1; exit 0'
psqlr -c "select status, count(*) from attendance_records where course_id=(select id from courses where class_code='$CC') group by 1 order by 1"
psqlr -c "select count(*) from participation_events where course_id=(select id from courses where class_code='$CC')"
