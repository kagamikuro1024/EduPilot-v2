-- US-P7-05 — cấu hình tích hợp theo lớp (IMAP; SMTP chỉ đọc; Teams ở US-P7-11) và hộp thư nhận bài.

-- name: IntegrationGet :one
select * from integration_configs where course_id = sqlc.arg(course_id) and kind = sqlc.arg(kind);

-- name: IntegrationInsert :one
insert into integration_configs (course_id, kind, config_public, config_enc, enabled, state)
values (sqlc.arg(course_id), sqlc.arg(kind), sqlc.arg(config_public), sqlc.arg(config_enc), sqlc.arg(enabled), '{}'::jsonb)
on conflict (course_id, kind) do nothing
returning *;

-- name: IntegrationUpdate :one
-- Khoá lạc quan theo version; ghi lại cấu hình mở khoá poll (state.paused / fail_count / next_at bị xoá, giữ uidvalidity + last_uid).
update integration_configs
set config_public = sqlc.arg(config_public), config_enc = coalesce(sqlc.narg(config_enc), config_enc), enabled = sqlc.arg(enabled),
    state = (state - 'paused' - 'fail_count' - 'next_at'), last_error = null, version = version + 1
where course_id = sqlc.arg(course_id) and kind = sqlc.arg(kind) and version = sqlc.arg(version)
returning *;

-- name: IntegrationResult :exec
-- Ghi kết quả một lần kết nối / đồng bộ: state (jsonb gộp), last_ok_at khi ok, last_error (chữ ngắn, không bí mật).
update integration_configs
set state = state || sqlc.arg(state_patch)::jsonb,
    last_ok_at = case when sqlc.arg(ok)::boolean then now() else last_ok_at end,
    last_error = sqlc.narg(last_error)
where course_id = sqlc.arg(course_id) and kind = sqlc.arg(kind);

-- name: IntegrationDue :many
-- Lớp có IMAP bật, chưa tạm dừng, đã tới hạn poll (state.next_at rỗng hoặc ≤ now).
select course_id, config_public, state from integration_configs
where kind = 'IMAP' and enabled and coalesce((state->>'paused')::boolean, false) = false
  and (state->>'next_at' is null or (state->>'next_at')::timestamptz <= sqlc.arg(now)::timestamptz)
order by course_id
limit sqlc.arg(page_limit);

-- name: MailInboundInsert :one
-- Ghi dòng thư; trùng (course, uidvalidity, uid) → không trả dòng (đã xử lý).
insert into mail_inbound (course_id, uidvalidity, message_uid, message_id, from_email, subject, received_at, status, reason, submission_id)
values (sqlc.arg(course_id), sqlc.arg(uidvalidity), sqlc.arg(message_uid), sqlc.narg(message_id), sqlc.arg(from_email), sqlc.arg(subject), sqlc.arg(received_at), sqlc.arg(status), sqlc.narg(reason), sqlc.narg(submission_id))
on conflict (course_id, uidvalidity, message_uid) do nothing
returning id;

-- name: MailInboundList :many
select id, from_email, subject, received_at, status, reason, submission_id, created_at
from mail_inbound
where course_id = sqlc.arg(course_id) and (sqlc.narg(status)::text is null or status = sqlc.narg(status)::text)
  and (sqlc.narg(cursor_at)::timestamptz is null or (received_at, id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by received_at desc, id desc
limit sqlc.arg(page_limit);

-- name: SubmissionStudentOf :one
select student_id, assignment_id from submissions where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: MailInboundSeen :one
-- Cùng Message-ID đã nhận ở lớp này (UIDVALIDITY đổi thì (uidvalidity, uid) mới nên khoá UNIQUE không bắt được thư cũ gặp lại).
select exists (select 1 from mail_inbound where course_id = sqlc.arg(course_id) and message_id = sqlc.arg(message_id))::boolean;
