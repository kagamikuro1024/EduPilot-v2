-- QC P6 — dựng lớp tải 1.000 sinh viên (LOAD1000) bằng SQL trên stack thử, sao chép công thức CONFIRMED của lớp 761987 (weekly đổi thành MANUAL).
-- Dùng:  docker exec -i edupilot100-postgres-1 psql -U edupilot -d edupilot -v ON_ERROR_STOP=1 < p6-load1000.sql
-- Sau đó: GV (teacher@) gọi các route trên lớp mới (QC lấy id: select id from courses where class_code='LOAD1000').
begin;
with c as (select * from courses where class_code = '761987'),
  nc as (insert into courses (subject_code, class_code, name, semester, status, created_by, join_code)
         select subject_code, 'LOAD1000', 'Lớp tải 1.000', semester, status, created_by, 'DKMPQRS' from c returning id),
  t as (select id from users where email = 'teacher@edupilot.local'),
  _te as (insert into enrollments (course_id, user_id, role_in_course, status, joined_via)
          select nc.id, t.id, 'TEACHER', 'ACTIVE', 'ADMIN' from nc, t returning 1),
  ns as (insert into grade_schemes (course_id, scheme_no, status, source_title, extraction, confirmed_by, confirmed_at, created_by, unclear_count)
         select nc.id, 1, 'CONFIRMED', s.source_title, s.extraction, s.confirmed_by, now(), s.created_by, 0
         from nc, grade_schemes s join c on s.course_id = c.id where s.status = 'CONFIRMED' returning id, course_id),
  _gb as (insert into gradebooks (course_id) select nc.id from nc returning 1)
select (select id from nc) as course, (select id from ns) as scheme \gset
-- thành phần / mục / luật
create temp table cmap as select gc.id old_id, uuidv7() new_id from grade_components gc join grade_schemes s on s.id = gc.scheme_id join courses c on c.id = s.course_id where c.class_code = '761987' and s.status = 'CONFIRMED';
insert into grade_components (id, course_id, scheme_id, key, name, grp, weight, position, citations)
  select m.new_id, :'course', :'scheme', gc.key, gc.name, gc.grp, gc.weight, gc.position, gc.citations from grade_components gc join cmap m on m.old_id = gc.id;
insert into grade_items (course_id, scheme_id, component_id, key, name, max_score, source, published_at, position)
  select :'course', :'scheme', m.new_id, gi.key, gi.name, gi.max_score, 'MANUAL', now(), gi.position from grade_items gi join cmap m on m.old_id = gi.component_id;
insert into grade_rules (course_id, scheme_id, kind, applicable, params, source_quote, source_page)
  select :'course', :'scheme', gr.kind, gr.applicable, gr.params, gr.source_quote, gr.source_page from grade_rules gr join grade_schemes s on s.id = gr.scheme_id join courses c on c.id = s.course_id where c.class_code = '761987' and s.status = 'CONFIRMED';
-- 1.000 sinh viên + ghi danh ROSTER
create temp table su as select uuidv7() id, g n from generate_series(1, 1000) g;
insert into users (id, email, full_name, role, student_code, email_verified_at) select id, 'load' || n || '@load.local', 'Tải ' || lpad(n::text, 4, '0'), 'STUDENT', '3' || lpad(n::text, 7, '0'), now() from su;
insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot) select :'course', id, 'STUDENT', 'ACTIVE', 'ROSTER', '3' || lpad(n::text, 7, '0') from su;
-- điểm: mọi mục 6 + (n % 4)
insert into grade_entries (course_id, item_id, student_id, score, source, entered_by)
  select :'course', i.id, su.id, 6 + (su.n % 4), 'MANUAL', (select id from users where email = 'teacher@edupilot.local') from su, grade_items i where i.course_id = :'course';
commit;
select 'LOAD1000', (select id from courses where class_code = 'LOAD1000'), (select count(*) from enrollments e join courses c on c.id = e.course_id where c.class_code = 'LOAD1000' and e.role_in_course = 'STUDENT'), (select count(*) from grade_entries e join courses c on c.id = e.course_id where c.class_code = 'LOAD1000');
