-- Chat riêng (US-P3-01; service ở US-P3-05). Mọi truy vấn lọc theo chủ phiên (user_id) — không có đường đọc chat của người khác.

-- name: InsertChatSession :one
insert into chat_sessions (course_id, user_id, title, document_id)
values (sqlc.arg(course_id), sqlc.arg(user_id), sqlc.narg(title), sqlc.narg(document_id))
returning *;

-- name: GetChatSession :one
select * from chat_sessions
where id = sqlc.arg(id) and user_id = sqlc.arg(user_id) and deleted_at is null;

-- name: ListChatSessions :many
select * from chat_sessions
where course_id = sqlc.arg(course_id) and user_id = sqlc.arg(user_id) and deleted_at is null
  and (sqlc.narg(cursor_at)::timestamptz is null or (last_message_at, id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by last_message_at desc, id desc
limit sqlc.arg(page_limit);

-- name: InsertChatMessage :one
insert into chat_messages (course_id, session_id, user_id, role, content, stream_status, client_msg_id, reply_to, intent, masked_count)
values (sqlc.arg(course_id), sqlc.arg(session_id), sqlc.arg(user_id), sqlc.arg(role), sqlc.arg(content), sqlc.arg(stream_status),
        sqlc.narg(client_msg_id), sqlc.narg(reply_to), sqlc.narg(intent), sqlc.arg(masked_count))
returning *;

-- name: ListChatMessages :many
select * from chat_messages
where session_id = sqlc.arg(session_id)
  and (sqlc.narg(cursor_at)::timestamptz is null or (created_at, id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by created_at desc, id desc
limit sqlc.arg(page_limit);

-- name: SetChatPartial :exec
update chat_messages set partial_content = sqlc.arg(partial_content)
where id = sqlc.arg(id) and stream_status = 'STREAMING';

-- name: FinishChatMessage :one
-- Ghi có điều kiện: chỉ tin còn STREAMING mới được đóng (tin đã bị Dừng / reaper đóng thì 0 hàng → pgx.ErrNoRows).
update chat_messages
set content = sqlc.arg(content), partial_content = null, stream_status = sqlc.arg(stream_status), completed_at = now(),
    citations = sqlc.arg(citations), blocks = sqlc.arg(blocks), confidence = sqlc.narg(confidence), low_confidence = sqlc.arg(low_confidence),
    no_context = sqlc.arg(no_context), degraded = sqlc.arg(degraded), error_code = sqlc.narg(error_code), trace_id = sqlc.narg(trace_id)
where id = sqlc.arg(id) and stream_status = 'STREAMING'
returning *;

-- name: ReapStaleChatMessages :many
-- Reaper: tin STREAMING quá hạn → FAILED, giữ phần đã có.
update chat_messages
set stream_status = 'FAILED', content = coalesce(partial_content, ''), partial_content = null, completed_at = now(), error_code = 'STALE'
where stream_status = 'STREAMING' and updated_at < sqlc.arg(before)
returning id, course_id, session_id;

-- name: TouchChatSession :exec
update chat_sessions set last_message_at = now() where id = sqlc.arg(id);

-- name: SoftDeleteChatSession :execrows
update chat_sessions set deleted_at = now() where id = sqlc.arg(id) and user_id = sqlc.arg(user_id) and deleted_at is null;

-- name: RestoreChatSession :execrows
update chat_sessions set deleted_at = null where id = sqlc.arg(id) and user_id = sqlc.arg(user_id) and deleted_at is not null;
