-- Điểm danh và sự kiện tham gia (FEAT-crm-360 US-P5-02). Bảng dữ liệu điểm: mọi truy vấn có course_id; không cache phía máy chủ.

-- name: CrmGetSession :one
select * from class_sessions where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: CrmGridStudents :many
-- Lưới một buổi: mọi sinh viên ACTIVE của lớp LEFT JOIN dòng điểm danh của buổi + số lần phát biểu của buổi; tên không dấu, id ổn định.
select e.user_id as student_id, u.full_name, e.student_code_snapshot as student_code,
       a.status as rec_status, a.note as rec_note, a.version as rec_version, a.updated_at as rec_marked_at, mu.full_name as rec_marked_by,
       coalesce(p.n, 0)::int as speak_count
from enrollments e
join users u on u.id = e.user_id
left join attendance_records a on a.course_id = e.course_id and a.session_id = sqlc.arg(session_id) and a.student_id = e.user_id
left join users mu on mu.id = a.marked_by
left join (
    select sp.student_id, count(*) as n from participation_events sp
    where sp.course_id = sqlc.arg(course_id) and sp.session_id = sqlc.arg(session_id) and sp.type = 'SPEAK'
    group by sp.student_id
) p on p.student_id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE'
order by vn_fold(u.full_name), e.user_id;

-- name: CrmActiveStudentIDs :many
select user_id from enrollments where course_id = sqlc.arg(course_id) and role_in_course = 'STUDENT' and status = 'ACTIVE';

-- name: CrmSessionRecords :many
-- Mọi dòng điểm danh của buổi (kể cả sinh viên đã bị mời ra: F2 giữ nguyên dòng) kèm tên người điểm, để so bản và dựng chi tiết xung đột.
select a.student_id, a.status, a.note, a.version, a.updated_at, u.full_name as marked_by_name
from attendance_records a join users u on u.id = a.marked_by
where a.course_id = sqlc.arg(course_id) and a.session_id = sqlc.arg(session_id);

-- name: CrmInsertRecords :many
-- Chèn hàng loạt trong MỘT câu (version = 1). Danh sách song song: student_ids[i] ↔ statuses[i] ↔ notes[i] ('' = không có ghi chú).
insert into attendance_records (course_id, session_id, student_id, status, note, marked_by)
select sqlc.arg(course_id)::uuid, sqlc.arg(session_id)::uuid, i.sid::uuid, s.st::attendance_status, nullif(t.note, ''), sqlc.arg(marked_by)::uuid
from unnest(sqlc.arg(student_ids)::text[]) with ordinality as i(sid, n)
join unnest(sqlc.arg(statuses)::text[]) with ordinality as s(st, n) on s.n = i.n
join unnest(sqlc.arg(notes)::text[]) with ordinality as t(note, n) on t.n = i.n
returning student_id, version;

-- name: CrmUpdateRecords :many
-- Sửa hàng loạt trong MỘT câu, version + 1. Người gọi đã so version trong cùng giao dịch (buổi đang bị khoá).
update attendance_records a
set status = s.st::attendance_status, note = nullif(t.note, ''), marked_by = sqlc.arg(marked_by)::uuid, version = a.version + 1
from unnest(sqlc.arg(student_ids)::text[]) with ordinality as i(sid, n)
join unnest(sqlc.arg(statuses)::text[]) with ordinality as s(st, n) on s.n = i.n
join unnest(sqlc.arg(notes)::text[]) with ordinality as t(note, n) on t.n = i.n
where a.course_id = sqlc.arg(course_id) and a.session_id = sqlc.arg(session_id) and a.student_id = i.sid::uuid
returning a.student_id, a.version;

-- name: CrmCompleteSession :one
-- Đặt "đã hoàn tất điểm danh" CHỈ lần đầu (idempotent: lần sau không có hàng → ErrNoRows).
update class_sessions set attendance_done_at = sqlc.arg(at), attendance_done_by = sqlc.arg(by)
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and attendance_done_at is null
returning attendance_done_at;

-- name: CrmStudentActive :one
select exists (select 1 from enrollments where course_id = sqlc.arg(course_id) and user_id = sqlc.arg(student_id) and role_in_course = 'STUDENT' and status = 'ACTIVE') as ok;

-- name: CrmSessionInCourse :one
select exists (select 1 from class_sessions where course_id = sqlc.arg(course_id) and id = sqlc.arg(id)) as ok;

-- name: CrmInsertParticipation :one
-- Trùng client_event_id ⇒ không có hàng trả về (ErrNoRows); người gọi tra lại theo (client_event_id, course_id, created_by) — TLR-8.
insert into participation_events (course_id, student_id, session_id, type, points, note, client_event_id, created_by)
values (sqlc.arg(course_id), sqlc.arg(student_id), sqlc.narg(session_id), sqlc.arg(type), sqlc.arg(points), sqlc.narg(note), sqlc.arg(client_event_id), sqlc.arg(created_by))
on conflict (client_event_id) do nothing
returning id, course_id, student_id, session_id, type, points, note, client_event_id, created_by, created_at;

-- name: CrmParticipationByClientEvent :one
select id, course_id, student_id, session_id, type, points, note, client_event_id, created_by, created_at
from participation_events
where client_event_id = sqlc.arg(client_event_id) and course_id = sqlc.arg(course_id) and created_by = sqlc.arg(created_by);

-- name: CrmParticipationGet :one
select id, course_id, student_id, session_id, type, points, note, created_by
from participation_events where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: CrmParticipationDelete :execrows
delete from participation_events where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: CrmParticipationList :many
-- Mới nhất trước, con trỏ (created_at, id); lọc theo sinh viên / buổi (tuỳ chọn).
select pe.id, pe.student_id, u.full_name, pe.session_id, pe.type, pe.points, pe.note, pe.created_by, pe.created_at
from participation_events pe join users u on u.id = pe.student_id
where pe.course_id = sqlc.arg(course_id)
  and (sqlc.narg(student_id)::uuid is null or pe.student_id = sqlc.narg(student_id))
  and (sqlc.narg(session_id)::uuid is null or pe.session_id = sqlc.narg(session_id))
  and (sqlc.narg(cur_at)::timestamptz is null or (pe.created_at, pe.id) < (sqlc.narg(cur_at)::timestamptz, sqlc.narg(cur_id)::uuid))
order by pe.created_at desc, pe.id desc
limit sqlc.arg(lim);

-- name: CrmMyAttendanceSummary :many
-- Tool AI: CHỈ của (lớp, sinh viên) đã tin cậy. Không có note, marked_by.
select recorded, present, late, excused, absent from v_attendance_summary where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id);

-- name: CrmMyRecentAbsent :many
select s.session_no, s.starts_at
from attendance_records a join class_sessions s on s.course_id = a.course_id and s.id = a.session_id
where a.course_id = sqlc.arg(course_id) and a.student_id = sqlc.arg(student_id) and a.status = 'ABSENT'
order by s.starts_at desc, s.id desc
limit 5;

-- name: CrmMyParticipationSummary :one
select (count(*) filter (where type = 'SPEAK'))::int as speak_count, count(*)::int as events, coalesce(sum(points), 0)::numeric(8,2) as points_total
from participation_events where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id);

-- name: CrmMyParticipationRecent :many
select created_at, type, points from participation_events
where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id)
order by created_at desc, id desc
limit 5;
