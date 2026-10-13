-- Escalation (SRS FEAT-escalation-notify 4.2, 4.3). Hai truy vấn DUY NHẤT chạm chat_messages theo hướng đọc của nhân viên:
-- GetEscalatedExchange và ListTicketTranscript (TestEscalationReadsChatOnlyViaTicketQueries). Mọi truy vấn bắt buộc course_id.

-- name: TicketByMessage :one
select * from escalation_tickets where message_id = sqlc.arg(message_id);

-- name: TicketLockStudentCap :exec
-- Chặn vượt trần khi song song (SRS 4.2.2): tiền tố 'ticket_cap:' tránh đụng không gian khoá khác.
select pg_advisory_xact_lock(hashtextextended('ticket_cap:' || sqlc.arg(student_id)::text || ':' || sqlc.arg(course_id)::text, 0));

-- name: TicketCountOpen :one
select count(*)::int from escalation_tickets
where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id) and status in ('OPEN', 'CLAIMED');

-- name: TicketInsert :one
-- Không trả hàng (pgx.ErrNoRows) khi đua với giao dịch khác cùng tin: người gọi SELECT lại và trả already:true.
insert into escalation_tickets (course_id, kind, origin, source_type, source_id, message_id, session_id, student_id, confidence)
values (sqlc.arg(course_id), 'QUESTION', sqlc.arg(origin), 'CHAT_MESSAGE', sqlc.arg(message_id), sqlc.arg(message_id), sqlc.arg(session_id), sqlc.arg(student_id), sqlc.narg(confidence))
on conflict (message_id) do nothing
returning *;

-- name: TicketsOfStudentForMessages :many
-- Chỉ ticket CỦA sinh viên (chat riêng không bao giờ đọc ticket của người khác).
select id, message_id, status, created_at, answered_at
from escalation_tickets
where student_id = sqlc.arg(student_id) and message_id = any(sqlc.arg(message_ids)::text[]::uuid[]); -- text[]: pgx ở chế độ simple protocol (PgBouncer) không mã hoá được []uuid.UUID

-- name: TicketGet :one
select * from escalation_tickets where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: GetEscalatedExchange :many
-- Đúng hai hàng: tin trợ lý được nhờ + câu hỏi của sinh viên mà nó trả lời (reply_to). Không có tin nào khác của phiên.
select m.* from chat_messages m
where m.course_id = sqlc.arg(course_id)
  and (m.id = sqlc.arg(message_id)
       or m.id = (select a.reply_to from chat_messages a where a.course_id = sqlc.arg(course_id) and a.id = sqlc.arg(message_id)))
order by m.created_at, m.id;

-- name: ListTicketTranscript :many
-- Chỉ các tin của chính ticket: STAFF_ANSWER / ESCALATION_FOLLOWUP mang khối `ticket` trỏ tới ticket này.
select m.* from chat_messages m
where m.course_id = sqlc.arg(course_id) and m.session_id = sqlc.arg(session_id)
  and m.intent in ('STAFF_ANSWER', 'ESCALATION_FOLLOWUP')
  and m.blocks @> jsonb_build_array(jsonb_build_object('kind', 'ticket', 'data', jsonb_build_object('ticket_id', sqlc.arg(ticket_id)::text)))
order by m.created_at, m.id;

-- name: TicketListByCourse :many
-- Trang đầu / trang kế của hàng chờ theo lớp (chỉ mục escalation_tickets_course_status_idx). status rỗng = mọi trạng thái.
select * from escalation_tickets
where course_id = sqlc.arg(course_id)
  and (sqlc.arg(status_filter)::text = '' or status = sqlc.arg(status_filter)::ticket_status)
  and (sqlc.narg(cursor_at)::timestamptz is null or (created_at, id) > (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by created_at, id
limit sqlc.arg(page_limit);

-- name: TicketListByStudent :many
select * from escalation_tickets
where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id)
  and (sqlc.arg(status_filter)::text = '' or status = sqlc.arg(status_filter)::ticket_status)
  and (sqlc.narg(cursor_at)::timestamptz is null or (created_at, id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by created_at desc, id desc
limit sqlc.arg(page_limit);

-- name: TicketSLACandidates :many
-- Job SLA, giai đoạn nhắc (SRS 4.6.2.1): lọc rẻ theo giờ LỊCH qua escalation_tickets_sla_idx; giờ làm việc tính ở Go.
select * from escalation_tickets
where status = 'OPEN' and sla_stage = sqlc.arg(stage)::smallint and waiting_since < sqlc.arg(before)::timestamptz
order by waiting_since, id
limit sqlc.arg(page_limit);
