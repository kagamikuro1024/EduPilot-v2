-- Liêm chính (US-PE-07, SRS FEAT-weekly-exam 4.7, 5.12): khoá chat, sự kiện rời tab / dán, so độ giống.

-- name: LockLatestAttempt :one
-- Lượt IN_PROGRESS còn trong hạn + grace của sinh viên có `deadline_at` MUỘN nhất (mọi lớp). Dùng chỉ mục từng phần `exam_attempts_running_idx`.
select id, exam_id, deadline_at from exam_attempts
where student_id = sqlc.arg(student_id) and status = 'IN_PROGRESS'
  and deadline_at + make_interval(secs => sqlc.arg(grace_seconds)::int) > sqlc.arg(now)::timestamptz
order by deadline_at desc
limit 1;

-- name: LockRunningStudents :many
select student_id from exam_attempts where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) and status = 'IN_PROGRESS';

-- name: LockStillRunning :many
select id from exam_attempts where id = any(sqlc.arg(ids)::text[]::uuid[]) and status = 'IN_PROGRESS';

-- name: ExamEventCount :one
select count(*)::int from exam_events where course_id = sqlc.arg(course_id) and attempt_id = sqlc.arg(attempt_id);

-- name: ExamEventsInsert :exec
-- Một lệnh cho cả lô: `rows` = [{"type","client_at","meta"}]; `occurred_at` là giờ MÁY CHỦ.
insert into exam_events (course_id, exam_id, attempt_id, student_id, type, occurred_at, client_at, meta)
select sqlc.arg(course_id), sqlc.arg(exam_id), sqlc.arg(attempt_id), sqlc.arg(student_id), x.type::exam_event_type, sqlc.arg(at), x.client_at, x.meta
from jsonb_to_recordset(sqlc.arg(rows)::jsonb) as x(type text, client_at timestamptz, meta jsonb);

-- name: ExamEventsList :many
select id, type, occurred_at, client_at, meta from exam_events
where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) and attempt_id = sqlc.arg(attempt_id)
  and (sqlc.narg(cursor_at)::timestamptz is null or (occurred_at, id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by occurred_at desc, id desc
limit sqlc.arg(row_limit);

-- name: ExamEventsSummary :many
-- Tóm tắt mỗi lượt theo loại: số lần, tổng thời gian rời tab (ms), tổng ký tự dán.
select type, count(*)::int as n,
       coalesce(sum((meta->>'duration_ms')::bigint), 0)::bigint as duration_ms,
       coalesce(sum((meta->>'chars')::bigint), 0)::bigint as chars
from exam_events
where course_id = sqlc.arg(course_id) and attempt_id = sqlc.arg(attempt_id)
group by type;

-- name: AttemptOfExam :one
-- Lượt thuộc bài thi này (Giảng viên đọc log): không có → 404.
select id, student_id from exam_attempts where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) and id = sqlc.arg(id);
