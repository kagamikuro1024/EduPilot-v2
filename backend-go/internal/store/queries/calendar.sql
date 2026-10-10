-- US-P8-03 — lịch gộp lúc đọc (UNION, không nhân bản), sự kiện của Staff, token ICS (băm), nhắc 24 giờ. Mọi truy vấn bắt đầu từ course_id / user_id (luật 13).

-- name: CalList :many
-- Ba nguồn gộp: class_sessions, exams (không DRAFT), calendar_events. `personal_state` chỉ cho sinh viên và dòng bài thi, từ exam_attempts của CHÍNH người xem.
-- Khoảng [from, to): dòng giao với khoảng. Keyset (starts_at, id).
with u as (
  select 'class_session'::text as src, cs.id, 'CLASS_SESSION'::text as typ, ('Buổi ' || cs.session_no::text || coalesce(' · ' || nullif(cs.topic, ''), ''))::text as title,
         cs.starts_at, cs.ends_at, cs.room as location, ''::text as exam_status, cs.updated_at, 0::int as ver
  from class_sessions cs where cs.course_id = sqlc.arg(course_id)
  union all
  select 'weekly_exam', e.id, 'EXAM', e.title, e.opens_at, e.closes_at, null, e.status::text, e.updated_at, 0
  from exams e where e.course_id = sqlc.arg(course_id) and e.status <> 'DRAFT' and e.opens_at is not null
  union all
  select 'calendar_event', ce.id, ce.type::text, ce.title, ce.starts_at, ce.ends_at, ce.location, '', ce.updated_at, ce.version
  from calendar_events ce where ce.course_id = sqlc.arg(course_id)
)
select u.src, u.id, u.typ, u.title, u.starts_at, coalesce(u.ends_at, 'epoch'::timestamptz)::timestamptz as ends_at, (u.ends_at is not null)::bool as has_end, u.location, u.exam_status, u.updated_at, u.ver,
       case when u.src = 'weekly_exam' and sqlc.arg(is_student)::bool then coalesce((
         select case when a.status = 'IN_PROGRESS' then 'IN_PROGRESS' else 'SUBMITTED' end
         from exam_attempts a where a.exam_id = u.id and a.student_id = sqlc.arg(viewer_id) limit 1), 'NOT_STARTED') else '' end::text as personal_state
from u
where u.starts_at < sqlc.arg(to_at)::timestamptz and coalesce(u.ends_at, u.starts_at) >= sqlc.arg(from_at)::timestamptz
  and (not sqlc.arg(only_exam)::bool or u.typ = 'EXAM')
  and (sqlc.narg(cursor_at)::timestamptz is null or (u.starts_at, u.id) > (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by u.starts_at, u.id
limit sqlc.arg(page_limit);

-- name: CalFeed :many
-- Feed ICS: mọi lớp người đó ACTIVE; kèm mã lớp (tiền tố khi ≥ 2 lớp). Giới hạn do người gọi (≤ 2.000).
with feed as (
  select c.class_code, 'class_session'::text as src, cs.id, ('Buổi ' || cs.session_no::text || coalesce(' · ' || nullif(cs.topic, ''), ''))::text as title, cs.starts_at, cs.ends_at, cs.room as location, cs.updated_at
  from class_sessions cs join courses c on c.id = cs.course_id
  where cs.course_id in (select en.course_id from enrollments en where en.user_id = sqlc.arg(owner) and en.status = 'ACTIVE')
  union all
  select c.class_code, 'weekly_exam', e.id, e.title, e.opens_at, e.closes_at, null, e.updated_at
  from exams e join courses c on c.id = e.course_id
  where e.status <> 'DRAFT' and e.opens_at is not null
    and e.course_id in (select en.course_id from enrollments en where en.user_id = sqlc.arg(owner) and en.status = 'ACTIVE')
  union all
  select c.class_code, 'calendar_event', ce.id, ce.title, ce.starts_at, ce.ends_at, ce.location, ce.updated_at
  from calendar_events ce join courses c on c.id = ce.course_id
  where ce.course_id in (select en.course_id from enrollments en where en.user_id = sqlc.arg(owner) and en.status = 'ACTIVE')
)
select f.class_code, f.src, f.id, f.title, f.starts_at, coalesce(f.ends_at, 'epoch'::timestamptz)::timestamptz as ends_at, (f.ends_at is not null)::bool as has_end, f.location, f.updated_at,
       (select count(*) from enrollments en2 where en2.user_id = sqlc.arg(owner) and en2.status = 'ACTIVE')::int as course_count
from feed f where f.starts_at >= sqlc.arg(from_at)::timestamptz and f.starts_at <= sqlc.arg(to_at)::timestamptz
order by f.starts_at, f.id
limit sqlc.arg(page_limit);

-- name: CalEventInsert :one
insert into calendar_events (course_id, type, title, starts_at, ends_at, location, description, created_by)
values (sqlc.arg(course_id), sqlc.arg(type), sqlc.arg(title), sqlc.arg(starts_at), sqlc.narg(ends_at), sqlc.narg(location), sqlc.narg(description), sqlc.arg(created_by))
returning *;

-- name: CalEventGet :one
select * from calendar_events where id = sqlc.arg(id) and course_id = sqlc.arg(course_id);

-- name: CalEventUpdate :one
update calendar_events set type = sqlc.arg(type), title = sqlc.arg(title), starts_at = sqlc.arg(starts_at), ends_at = sqlc.narg(ends_at),
       location = sqlc.narg(location), description = sqlc.narg(description), version = version + 1
where id = sqlc.arg(id) and course_id = sqlc.arg(course_id) and version = sqlc.arg(version)
returning *;

-- name: CalEventDelete :execrows
delete from calendar_events where id = sqlc.arg(id) and course_id = sqlc.arg(course_id);

-- name: CalCourseStatus :one
select status from courses where id = sqlc.arg(id);

-- name: IcsTokenSet :execrows
update users u set ics_token = sqlc.arg(token_hash) where u.id = sqlc.arg(owner);

-- name: IcsTokenClear :execrows
update users u set ics_token = null where u.id = sqlc.arg(owner);

-- name: IcsTokenExists :one
select (u.ics_token is not null)::bool from users u where u.id = sqlc.arg(owner);

-- name: IcsUserByHash :one
-- Tra theo sha256(token) bằng chỉ mục duy nhất; tài khoản không ACTIVE → như không có.
select u.id from users u where u.ics_token = sqlc.arg(token_hash) and u.status = 'ACTIVE';

-- name: CalDue :many
-- Nguồn có starts_at ∈ (now, until] của lớp ACTIVE: buổi học, bài thi SCHEDULED / OPEN, sự kiện.
select d.src, d.id, d.course_id, d.typ, d.title, d.starts_at from (
  select 'CLASS_SESSION'::text as src, cs.id, cs.course_id, 'CLASS_SESSION'::text as typ, ('Buổi ' || cs.session_no::text || coalesce(' · ' || nullif(cs.topic, ''), ''))::text as title, cs.starts_at
  from class_sessions cs
  union all
  select 'WEEKLY_EXAM', e.id, e.course_id, 'EXAM', e.title, e.opens_at from exams e where e.status in ('SCHEDULED', 'OPEN') and e.opens_at is not null
  union all
  select 'CALENDAR_EVENT', ce.id, ce.course_id, ce.type::text, ce.title, ce.starts_at from calendar_events ce
) d join courses c on c.id = d.course_id and c.status = 'ACTIVE'
where d.starts_at > sqlc.arg(now_at)::timestamptz and d.starts_at <= sqlc.arg(until_at)::timestamptz
order by d.starts_at, d.id;

-- name: CalRecipients :many
-- Sinh viên ACTIVE của lớp, tài khoản ACTIVE; lô theo user_id (keyset). Chưa có user_settings = mặc định bật.
select e.user_id, u.email, u.full_name, (u.email_verified_at is not null)::bool as verified,
       coalesce(us.remind_deadline_by_mail, true)::bool as remind_mail, coalesce(us.preferences, '{}'::jsonb) as preferences
from enrollments e join users u on u.id = e.user_id
left join user_settings us on us.user_id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.status = 'ACTIVE' and e.role_in_course = 'STUDENT' and u.status = 'ACTIVE'
  and e.user_id > sqlc.arg(after_user)::uuid
order by e.user_id
limit sqlc.arg(page_limit);

-- name: CalInsertReminder :execrows
insert into reminder_log (user_id, course_id, source_type, source_id, starts_at, kind)
values (sqlc.arg(user_id), sqlc.arg(course_id), sqlc.arg(source_type), sqlc.arg(source_id), sqlc.arg(starts_at), 'T24H')
on conflict (user_id, source_type, source_id, starts_at, kind) do nothing;
