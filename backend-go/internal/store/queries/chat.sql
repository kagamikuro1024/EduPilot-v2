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

-- name: SetChatPartial :execrows
-- 0 hàng = tin đã bị Dừng / reaper / retry đóng: G phải dừng.
update chat_messages set partial_content = sqlc.arg(partial_content)
where id = sqlc.arg(id) and attempt = sqlc.arg(attempt) and stream_status = 'STREAMING';

-- name: FinishChatMessage :execrows
-- Ghi cuối của G: chỉ tin còn STREAMING mới được đóng (đã bị Dừng / reaper đóng thì 0 hàng — G dừng lặng lẽ).
update chat_messages
set content = sqlc.arg(content), partial_content = null, stream_status = 'DONE', completed_at = now(),
    citations = sqlc.arg(citations), blocks = sqlc.arg(blocks), confidence = sqlc.narg(confidence), low_confidence = sqlc.arg(low_confidence),
    no_context = sqlc.arg(no_context), degraded = sqlc.arg(degraded), masked_count = sqlc.arg(masked_count), intent = sqlc.narg(intent),
    trace_id = sqlc.narg(trace_id)
where id = sqlc.arg(id) and attempt = sqlc.arg(attempt) and stream_status = 'STREAMING';

-- name: FailChatMessage :execrows
-- Lỗi / gián đoạn: giữ partial_content; có điều kiện STREAMING và đúng lượt (attempt).
update chat_messages
set stream_status = 'FAILED', error_code = sqlc.arg(error_code), completed_at = now(), partial_content = coalesce(sqlc.narg(partial_content), partial_content),
    masked_count = sqlc.arg(masked_count), intent = coalesce(sqlc.narg(intent), intent), trace_id = coalesce(sqlc.narg(trace_id), trace_id)
where id = sqlc.arg(id) and attempt = sqlc.arg(attempt) and stream_status = 'STREAMING';

-- name: CancelChatMessage :execrows
-- Dừng: giữ nguyên partial_content (nếu G còn sống truyền phần mới nhất thì ghi kèm).
update chat_messages
set stream_status = 'CANCELLED', completed_at = now(), partial_content = coalesce(sqlc.narg(partial_content), partial_content)
where id = sqlc.arg(id) and attempt = sqlc.arg(attempt) and stream_status = 'STREAMING';

-- name: RetryChatMessage :one
-- Đặt lại MỌI cột của lượt trước ở CÙNG hàng (kể cả completed_at); chỉ ASSISTANT FAILED / CANCELLED của chính mình.
update chat_messages
set content = '', partial_content = null, stream_status = 'STREAMING', completed_at = null, error_code = null, citations = '[]', blocks = '[]',
    confidence = null, low_confidence = false, no_context = false, degraded = false, masked_count = 0, intent = null, trace_id = null,
    feedback = null, attempt = attempt + 1
where id = sqlc.arg(id) and user_id = sqlc.arg(user_id) and role = 'ASSISTANT' and stream_status in ('FAILED', 'CANCELLED')
returning *;

-- name: ReapStaleChatMessages :many
-- Reaper: tin STREAMING quá hạn → FAILED INTERRUPTED, giữ partial_content. `updated_at` do trigger.
update chat_messages
set stream_status = 'FAILED', error_code = 'INTERRUPTED', completed_at = now()
where stream_status = 'STREAMING' and updated_at < sqlc.arg(before)
returning id, user_id, attempt;

-- name: GetChatMessageByID :one
-- Chỉ dùng nội bộ sau khi đã kiểm chủ tin (người xem SSE đã qua s.message).
select * from chat_messages where id = sqlc.arg(id);

-- name: GetChatMessageOwned :one
select * from chat_messages where id = sqlc.arg(id) and user_id = sqlc.arg(user_id);

-- name: GetChatUserMessageByClientID :one
select * from chat_messages where session_id = sqlc.arg(session_id) and client_msg_id = sqlc.arg(client_msg_id) and role = 'USER';

-- name: GetChatAssistantFor :one
select * from chat_messages where session_id = sqlc.arg(session_id) and reply_to = sqlc.arg(reply_to) and role = 'ASSISTANT';

-- name: GetChatUserMessageOf :one
select * from chat_messages where id = sqlc.arg(id) and role = 'USER';

-- name: SetChatTitle :exec
update chat_sessions set title = sqlc.arg(title) where id = sqlc.arg(id) and title is null;

-- name: SetChatFeedback :execrows
update chat_messages set feedback = sqlc.narg(feedback)
where id = sqlc.arg(id) and user_id = sqlc.arg(user_id) and role = 'ASSISTANT' and stream_status = 'DONE';

-- name: ChatHistory :many
-- Các lượt gần nhất ĐÃ XONG trước tin hiện tại (mới nhất trước; người gọi đảo lại).
select role, content from chat_messages
where session_id = sqlc.arg(session_id) and stream_status = 'DONE' and (created_at, id) < (sqlc.arg(before_at)::timestamptz, sqlc.arg(before_id)::uuid)
order by created_at desc, id desc
limit sqlc.arg(page_limit);

-- name: ChatDocumentUsable :one
-- "Hỏi AI về tài liệu này": tài liệu READY, dùng cho RAG, sinh viên được thấy, thuộc lớp.
select d.id from documents d
where d.id = sqlc.arg(id) and d.status = 'READY' and d.visible_to_students and d.use_for_rag and d.type <> 'ANSWER_KEY'
  and (d.course_id = sqlc.arg(course_id) or exists (select 1 from document_courses dc where dc.document_id = d.id and dc.course_id = sqlc.arg(course_id)))
  and exists (select 1 from content_chunks c where c.document_id = d.id and c.course_ids @> array[sqlc.arg(course_id)::uuid] and c.embedding is not null);

-- name: ChatCourseStatus :one
select status from courses where id = sqlc.arg(id);

-- name: ExamChatBlockedInsert :exec
-- exam.RecordChatBlocked: một dòng CHAT_BLOCKED cho lượt đang IN_PROGRESS của đúng sinh viên (không nội dung).
insert into exam_events (course_id, exam_id, attempt_id, student_id, type, occurred_at, meta)
select a.course_id, a.exam_id, a.id, a.student_id, 'CHAT_BLOCKED', sqlc.arg(at), '{}'::jsonb
from exam_attempts a where a.id = sqlc.arg(attempt_id) and a.student_id = sqlc.arg(student_id) and a.status = 'IN_PROGRESS';

-- name: TouchChatSession :exec
update chat_sessions set last_message_at = now() where id = sqlc.arg(id);

-- name: SoftDeleteChatSession :execrows
update chat_sessions set deleted_at = now() where id = sqlc.arg(id) and user_id = sqlc.arg(user_id) and deleted_at is null;

-- name: RestoreChatSession :execrows
update chat_sessions set deleted_at = null where id = sqlc.arg(id) and user_id = sqlc.arg(user_id) and deleted_at is not null;
