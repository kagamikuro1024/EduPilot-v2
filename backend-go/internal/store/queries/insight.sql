-- Báo cáo lỗ hổng kiến thức (US-P10-04/05). Mọi truy vấn lọc theo course_id; chỉ sinh viên ACTIVE của lớp; phiên chat đã xoá không vào báo cáo.

-- name: InsightChatQuestions :many
-- Câu hỏi chat riêng của sinh viên + cờ của câu trả lời (tự tin thấp / không có ngữ cảnh / Not helpful / escalate).
select u.id, u.user_id, u.content, u.created_at,
       coalesce(a.low_confidence, false)::bool as low_confidence,
       coalesce(a.no_context, false)::bool as no_context,
       coalesce(a.feedback = 'NOT_HELPFUL', false)::bool as not_helpful,
       exists (select 1 from escalation_tickets t where t.course_id = u.course_id and t.message_id = a.id)::bool as escalated
from chat_messages u
join chat_sessions s on s.id = u.session_id and s.deleted_at is null
join enrollments e on e.course_id = u.course_id and e.user_id = u.user_id and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE'
left join chat_messages a on a.course_id = u.course_id and a.reply_to = u.id and a.role = 'ASSISTANT'
where u.course_id = sqlc.arg(course_id) and u.role = 'USER' and u.content <> ''
  and a.intent is distinct from 'CRISIS' -- tin khủng hoảng không bao giờ vào báo cáo (P-11-34; mã còn kiểm DetectIntent trên chữ gốc)
  and u.created_at >= sqlc.arg(from_ts) and u.created_at < sqlc.arg(to_ts)
order by u.created_at, u.id
limit sqlc.arg(max_rows);

-- name: InsightThreadQuestions :many
-- Hỏi đáp công khai (đã qua tường lửa PII khi lưu): chủ thread của sinh viên, chưa ẩn / xoá.
select t.id, t.author_id as user_id, t.title, t.body, t.created_at,
       coalesce(t.ai_state = 'SKIPPED', false)::bool as ai_skipped
from forum_threads t
join enrollments e on e.course_id = t.course_id and e.user_id = t.author_id and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE'
where t.course_id = sqlc.arg(course_id) and t.deleted_at is null and t.hidden_at is null
  and t.created_at >= sqlc.arg(from_ts) and t.created_at < sqlc.arg(to_ts)
order by t.created_at, t.id
limit sqlc.arg(max_rows);

-- name: InsightPracticeTopics :many
-- Tín hiệu tổng hợp của luyện đề: theo chủ đề câu hỏi, số lượt trả lời và số sai. Không danh tính.
select coalesce(nullif(q.topic, ''), '')::text as topic, count(*)::int as answered, (count(*) filter (where a.is_correct = false))::int as wrong
from practice_answers a
join practice_items i on i.id = a.item_id and i.course_id = a.course_id
join question_bank q on q.id = i.question_id and q.course_id = a.course_id
where a.course_id = sqlc.arg(course_id) and a.answered_at >= sqlc.arg(from_ts) and a.answered_at < sqlc.arg(to_ts) and a.is_correct is not null
group by 1
having count(*) >= 5;

-- name: InsightWatermark :one
-- Mốc dữ liệu mới nhất của ba nguồn trong kỳ (câu hỏi chat, thread, lượt trả lời luyện đề).
select coalesce(greatest(
         (select max(m.created_at) from chat_messages m where m.course_id = sqlc.arg(course_id) and m.role = 'USER' and m.created_at >= sqlc.arg(from_ts) and m.created_at < sqlc.arg(to_ts)),
         (select max(t.created_at) from forum_threads t where t.course_id = sqlc.arg(course_id) and t.hidden_at is null and t.deleted_at is null and t.created_at >= sqlc.arg(from_ts) and t.created_at < sqlc.arg(to_ts)),
         (select max(a.answered_at) from practice_answers a where a.course_id = sqlc.arg(course_id) and a.answered_at >= sqlc.arg(from_ts) and a.answered_at < sqlc.arg(to_ts))
       ), 'epoch'::timestamptz)::timestamptz as watermark;

-- name: InsightNearestChunk :one
-- Chương / tuần của tài liệu có đoạn gần tâm cụm nhất (metadata tài liệu của lớp).
select d.title, d.week_no
from content_chunks c
join documents d on d.id = c.document_id
where sqlc.arg(course_id)::uuid = any (c.course_ids) and c.embedding is not null and d.category is distinct from 'THREAD_KNOWLEDGE'
order by c.embedding <=> sqlc.arg(centroid)::vector
limit 1;

-- name: InsightReportInsert :one
insert into insight_reports (course_id, kind, period_from, period_to, payload, data_watermark, generated_by)
values (sqlc.arg(course_id), 'KNOWLEDGE_GAP', sqlc.arg(period_from), sqlc.arg(period_to), sqlc.arg(payload), sqlc.arg(data_watermark), sqlc.arg(generated_by))
returning id, created_at;

-- name: InsightReportLatest :many
-- Hai báo cáo mới nhất của lớp (mới nhất trước): bản hiện tại + bản trước để so sánh.
select id, period_from, period_to, payload, data_watermark, generated_by, created_at
from insight_reports
where course_id = sqlc.arg(course_id) and kind = 'KNOWLEDGE_GAP' and coalesce((payload ->> 'schema')::int, 1) >= 2 -- báo cáo cũ (trước P-11-31/32/34, không có schema) coi như không có
order by created_at desc, id desc
limit 2;

-- name: InsightReportLatestFor :one
-- Báo cáo mới nhất cùng period_days (so data_watermark để quyết định `reused`).
select id, data_watermark
from insight_reports
where course_id = sqlc.arg(course_id) and kind = 'KNOWLEDGE_GAP' and (payload ->> 'period_days')::int = sqlc.arg(period_days)::int and coalesce((payload ->> 'schema')::int, 1) >= 2
order by created_at desc, id desc
limit 1;

-- name: InsightReportGet :one
select id, period_from, period_to, payload, data_watermark, generated_by, created_at
from insight_reports
where course_id = sqlc.arg(course_id) and kind = 'KNOWLEDGE_GAP' and id = sqlc.arg(id) and coalesce((payload ->> 'schema')::int, 1) >= 2;
