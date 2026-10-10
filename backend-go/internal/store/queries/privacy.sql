-- Sự kiện PII (US-P3-01): chỉ loại + số lượng, không văn bản. Chỉ INSERT (trigger chặn UPDATE/DELETE).

-- name: InsertPIIEvent :exec
insert into pii_events (course_id, session_id, user_id, channel, pii_type, count, action)
values (sqlc.arg(course_id), sqlc.narg(session_id), sqlc.arg(user_id), sqlc.arg(channel), sqlc.arg(pii_type), sqlc.arg(count), sqlc.arg(action));

-- name: CountPIIEventsByCourse :many
select pii_type, action, sum(count)::bigint as total
from pii_events
where course_id = sqlc.arg(course_id) and created_at >= sqlc.arg(since)
group by pii_type, action
order by pii_type, action;

-- name: PrivacyRoster :many
-- Từ điển PII của một lớp: chỉ sinh viên ACTIVE (giảng viên / TA không vào từ điển — Q8).
select u.full_name, e.student_code_snapshot
from enrollments e join users u on u.id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE';
