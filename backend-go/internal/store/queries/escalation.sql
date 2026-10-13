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

-- name: TicketCountOpenInCourse :one
-- Số ticket đang chờ (OPEN) của lớp: nguồn đếm của thông báo gộp TICKET_NEW.
select count(*)::int from escalation_tickets where course_id = sqlc.arg(course_id) and status = 'OPEN';

-- name: TicketStaffIDs :many
-- Người nhận thông báo ticket: TA + TEACHER ACTIVE của lớp.
select user_id from enrollments where course_id = sqlc.arg(course_id) and role_in_course in ('TEACHER', 'TA') and status = 'ACTIVE';

-- name: TicketListStaff :many
-- Hàng chờ của nhân viên (SRS 6.2 #4, #5): một lớp hoặc mọi lớp của mình (course_ids). Câu hỏi chỉ là ĐOẠN CẮT 140 ký tự của đúng câu đã nhờ (reply_to của tin được nhờ) — cùng phạm vi với
-- GetEscalatedExchange (TestEscalationReadsChatOnlyViaTicketQueries cho phép riêng hai truy vấn liệt kê này; proposals #13). status_filter: OPEN|CLAIMED|ANSWERED|ALL (ALL gồm CLOSED).
select t.id, t.course_id, t.kind, t.status, t.student_id, s.full_name as student_name, t.claimed_by, cb.full_name as claimed_name, t.answered_by, t.sla_stage, t.waiting_since,
       t.reopened_count, t.save_as_knowledge, t.version, t.created_at, c.class_code,
       left(coalesce(q.content, ''), 140)::text as question_preview
from escalation_tickets t
join users s on s.id = t.student_id
join courses c on c.id = t.course_id
left join users cb on cb.id = t.claimed_by
left join chat_messages a on a.course_id = t.course_id and a.id = t.message_id
left join chat_messages q on q.course_id = a.course_id and q.id = a.reply_to
where t.course_id = any(sqlc.arg(course_ids)::text[]::uuid[])
  and (sqlc.arg(status_filter)::text = 'ALL' or t.status = sqlc.arg(status_filter)::ticket_status)
  and (sqlc.narg(cursor_at)::timestamptz is null or (t.created_at, t.id) > (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by t.created_at, t.id
limit sqlc.arg(page_limit);

-- name: TicketStaffCourseIDs :many
-- Lớp mà người này đang ACTIVE là TEACHER / TA (cho "Tất cả lớp của tôi").
select course_id::text from enrollments where user_id = sqlc.arg(user_id) and role_in_course in ('TEACHER', 'TA') and status = 'ACTIVE';

-- name: TicketUserBrief :one
-- Tên + vai trong lớp của một nhân viên (hiển thị "Cô … đã nhận").
select u.id, u.full_name, e.role_in_course::text as role from users u join enrollments e on e.user_id = u.id and e.course_id = sqlc.arg(course_id) where u.id = sqlc.arg(user_id);

-- name: TicketViewGet :one
select t.*, s.full_name as student_name, cb.full_name as claimed_name, ab.full_name as answered_name, c.class_code
from escalation_tickets t
join users s on s.id = t.student_id
join courses c on c.id = t.course_id
left join users cb on cb.id = t.claimed_by
left join users ab on ab.id = t.answered_by
where t.course_id = sqlc.arg(course_id) and t.id = sqlc.arg(id);

-- name: TicketClaim :one
update escalation_tickets set status = 'CLAIMED', claimed_by = sqlc.arg(user_id), claimed_at = now(), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'OPEN' and (sqlc.narg(version)::int is null or version = sqlc.narg(version)::int)
returning *;

-- name: TicketRelease :one
-- waiting_since KHÔNG đổi: bỏ nhận không tính lại đồng hồ SLA.
update escalation_tickets set status = 'OPEN', claimed_by = null, claimed_at = null, version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'CLAIMED' and claimed_by = sqlc.arg(user_id) and (sqlc.narg(version)::int is null or version = sqlc.narg(version)::int)
returning *;

-- name: TicketAnswer :one
update escalation_tickets set status = 'ANSWERED', answer = sqlc.arg(body), answered_by = sqlc.arg(user_id), answered_at = now(),
       claimed_by = case when status = 'OPEN' then sqlc.arg(user_id)::uuid else claimed_by end, claimed_at = coalesce(claimed_at, now()), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id)
  and (status = 'OPEN' or (status = 'CLAIMED' and claimed_by = sqlc.arg(user_id)) or (status = 'ANSWERED' and answered_by = sqlc.arg(user_id)))
  and (sqlc.narg(version)::int is null or version = sqlc.narg(version)::int)
returning *;

-- name: TicketDismiss :one
update escalation_tickets set status = 'CLOSED', closed_at = now(), closed_reason = 'STAFF_DISMISSED', dismiss_reason = sqlc.arg(reason), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and (status = 'OPEN' or (status = 'CLAIMED' and claimed_by = sqlc.arg(user_id)))
  and (sqlc.narg(version)::int is null or version = sqlc.narg(version)::int)
returning *;

-- name: TicketStudentIdentity :one
-- Khoá nhận dạng của sinh viên hỏi (tên, MSSV, email) để câu trả lời được phép nhắc CHÍNH họ (SRS 4.2.6).
select u.full_name, u.email, coalesce(u.student_code, '') as student_code, coalesce(e.student_code_snapshot, '') as roster_code
from users u left join enrollments e on e.user_id = u.id and e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT'
where u.id = sqlc.arg(student_id);

-- name: TicketOtherStudentEmails :many
-- Email (trong danh sách đưa vào) thuộc một sinh viên KHÁC của lớp.
select lower(u.email)::text from users u join enrollments e on e.user_id = u.id
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.user_id <> sqlc.arg(student_id) and lower(u.email) = any(sqlc.arg(emails)::text[]);
