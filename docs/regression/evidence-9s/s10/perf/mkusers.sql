-- 100 SV hiệu năng QC (qcperf001..100@edupilot.local) trong QD10A; mật khẩu = của sv.kha (sao chép băm). Dọn: delete from users where email like 'qcperf%@edupilot.local'
with src as (select password_hash from users where email='sv.kha@edupilot.local'),
ins as (
  insert into users (email, password_hash, full_name, role, student_code, email_verified_at, status)
  select format('qcperf%s@edupilot.local', lpad(g::text,3,'0')), (select password_hash from src), format('QC Perf %s', g), 'STUDENT', format('QCP%s', lpad(g::text,5,'0')), now(), 'ACTIVE'
  from generate_series(1,100) g on conflict (email) do nothing returning id, student_code)
insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot)
select (select id from courses where class_code='QD10A'), id, 'STUDENT', 'ACTIVE', 'ROSTER', student_code from ins;
insert into user_settings (user_id) select id from users where email like 'qcperf%@edupilot.local' on conflict do nothing;
update users set tracking_notice_ack_at = now() where email like 'qcperf%@edupilot.local';
select count(*) from users where email like 'qcperf%@edupilot.local';
