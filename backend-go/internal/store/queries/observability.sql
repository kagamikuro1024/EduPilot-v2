-- Quan sát AI (US-P10-01…03): chỉ đọc llm_audit (không cột nội dung), escalation_tickets, chat_messages, pii_events, grading_results.
-- Ngày theo múi giờ Asia/Ho_Chi_Minh. Lỗi = status ∉ {ok, degraded, cancelled}. Mọi truy vấn lọc theo created_at (chỉ mục), không OFFSET.

-- name: ObsWindowSummary :one
select count(*)::bigint as requests,
       coalesce(percentile_cont(0.5) within group (order by latency_ms), 0)::float8 as p50_ms,
       coalesce(percentile_cont(0.95) within group (order by latency_ms), 0)::float8 as p95_ms,
       (count(*) filter (where status not in ('ok', 'degraded', 'cancelled')))::bigint as errors,
       (count(*) filter (where fallback_index > 0))::bigint as fallbacks,
       coalesce(sum(cost_est), 0)::numeric(14,4) as cost,
       coalesce(sum(pii_masked_count), 0)::bigint as pii_masked
  from llm_audit
 where created_at >= sqlc.arg(from_ts) and created_at < sqlc.arg(to_ts)
   and (sqlc.narg(course_id)::uuid is null or course_id = sqlc.narg(course_id));

-- name: ObsDailySummary :many
select ((created_at at time zone 'Asia/Ho_Chi_Minh')::date)::date as day,
       count(*)::bigint as requests,
       coalesce(percentile_cont(0.95) within group (order by latency_ms), 0)::float8 as p95_ms,
       (count(*) filter (where status not in ('ok', 'degraded', 'cancelled')))::bigint as errors,
       coalesce(sum(pii_masked_count), 0)::bigint as pii_masked,
       coalesce(sum(cost_est), 0)::numeric(14,4) as cost
  from llm_audit
 where created_at >= sqlc.arg(from_ts) and created_at < sqlc.arg(to_ts)
   and (sqlc.narg(course_id)::uuid is null or course_id = sqlc.narg(course_id))
 group by 1 order by 1 desc;

-- name: ObsEscalationsDaily :many
select ((created_at at time zone 'Asia/Ho_Chi_Minh')::date)::date as day, count(*)::bigint as n
  from escalation_tickets
 where created_at >= sqlc.arg(from_ts) and created_at < sqlc.arg(to_ts)
   and (sqlc.narg(course_id)::uuid is null or course_id = sqlc.narg(course_id))
 group by 1;

-- name: ObsCostByTask :many
select task, count(*)::bigint as requests, coalesce(sum(cost_est), 0)::numeric(14,4) as cost,
       coalesce(sum(tokens_in), 0)::bigint as tokens_in, coalesce(sum(tokens_out), 0)::bigint as tokens_out
  from llm_audit
 where created_at >= sqlc.arg(from_ts) and created_at < sqlc.arg(to_ts)
 group by task order by cost desc, requests desc;

-- name: ObsRequestsPage :many
-- Bảng yêu cầu (ADMIN): KHÔNG cột nội dung. filter: all | problem (status ≠ ok) | privacy (pii_masked_count > 0).
select a.id, a.created_at, a.task, a.lane, a.provider, a.model, a.latency_ms, a.status, a.error_kind, a.degraded, a.fallback_index,
       a.pii_masked_count, a.course_id, c.class_code as course_code
  from llm_audit a left join courses c on c.id = a.course_id
 where (sqlc.narg(from_ts)::timestamptz is null or a.created_at >= sqlc.narg(from_ts))
   and (sqlc.narg(to_ts)::timestamptz is null or a.created_at < sqlc.narg(to_ts))
   and (sqlc.narg(course_id)::uuid is null or a.course_id = sqlc.narg(course_id))
   and (sqlc.narg(task)::text is null or a.task = sqlc.narg(task))
   and (sqlc.arg(filter)::text = 'all'
        or (sqlc.arg(filter)::text = 'problem' and a.status <> 'ok')
        or (sqlc.arg(filter)::text = 'privacy' and a.pii_masked_count > 0))
   and (sqlc.narg(cursor_at)::timestamptz is null or (a.created_at, a.id) < (sqlc.narg(cursor_at), sqlc.narg(cursor_id)::uuid))
 order by a.created_at desc, a.id desc
 limit sqlc.arg(page_limit);

-- name: ObsRequestDetail :one
-- Metadata của một yêu cầu + bản prompt đã che (nếu còn); độ tự tin lấy từ tin chat cùng trace khi bản ghi payload chưa có.
select a.id, a.created_at, a.task, a.lane, a.provider, a.model, a.latency_ms, a.status, a.error_kind, a.course_id, a.trace_id,
       (p.audit_id is not null)::boolean as has_payload, p.masked_prompt as masked_prompt, p.tools as tools,
       coalesce(coalesce(p.confidence, (select m.confidence from chat_messages m where m.trace_id = a.trace_id and m.confidence is not null limit 1))::text, '')::text as confidence
  from llm_audit a left join llm_request_payloads p on p.audit_id = a.id
 where a.id = sqlc.arg(id);

-- name: ObsPayloadsPurge :execrows
delete from llm_request_payloads where created_at < sqlc.arg(before);

-- name: ObsCourseQuality :one
-- Chất lượng một lớp trong cửa sổ: không danh tính, không nội dung.
select
  (select count(*) from chat_messages m where m.course_id = sqlc.arg(course_id) and m.feedback = 'HELPFUL' and m.created_at >= sqlc.arg(from_ts) and m.created_at < sqlc.arg(to_ts))::bigint as helpful,
  (select count(*) from chat_messages m where m.course_id = sqlc.arg(course_id) and m.feedback = 'NOT_HELPFUL' and m.created_at >= sqlc.arg(from_ts) and m.created_at < sqlc.arg(to_ts))::bigint as not_helpful,
  (select count(*) from escalation_tickets t where t.course_id = sqlc.arg(course_id) and t.created_at >= sqlc.arg(from_ts) and t.created_at < sqlc.arg(to_ts))::bigint as escalations,
  (select count(*) from forum_posts f where f.course_id = sqlc.arg(course_id) and f.kind = 'AI' and f.verification_state in ('VERIFIED', 'CORRECTED') and f.created_at >= sqlc.arg(from_ts) and f.created_at < sqlc.arg(to_ts))::bigint as thread_correct,
  (select count(*) from forum_posts f where f.course_id = sqlc.arg(course_id) and f.kind = 'AI' and f.verification_state = 'REJECTED' and f.created_at >= sqlc.arg(from_ts) and f.created_at < sqlc.arg(to_ts))::bigint as thread_reject,
  (select count(*) from grading_results g where g.course_id = sqlc.arg(course_id) and g.status in ('REVIEWED', 'PUBLISHED') and g.run_state = 'DONE' and g.created_at >= sqlc.arg(from_ts) and g.created_at < sqlc.arg(to_ts))::bigint as graded_reviewed,
  (select count(*) from grading_results g where g.course_id = sqlc.arg(course_id) and g.status in ('REVIEWED', 'PUBLISHED') and g.run_state = 'DONE' and (coalesce(g.deviation, 0) <> 0 or cardinality(g.teacher_changed_criteria) > 0) and g.created_at >= sqlc.arg(from_ts) and g.created_at < sqlc.arg(to_ts))::bigint as graded_changed,
  (select coalesce(avg(abs(g.deviation)), 0) from grading_results g where g.course_id = sqlc.arg(course_id) and g.status in ('REVIEWED', 'PUBLISHED') and g.run_state = 'DONE' and g.deviation is not null and g.created_at >= sqlc.arg(from_ts) and g.created_at < sqlc.arg(to_ts))::numeric(6,2) as mean_abs_deviation;

-- name: ObsAlertErrorWindow :one
-- Cảnh báo lỗi: cửa sổ 1 giờ gần nhất.
select count(*)::bigint as requests, (count(*) filter (where status not in ('ok', 'degraded', 'cancelled')))::bigint as errors
  from llm_audit where created_at >= sqlc.arg(from_ts) and created_at < sqlc.arg(to_ts);

-- name: ObsAlertBudgetUsage :many
-- Chi phí trong ngày so với daily_limit của llm_budgets: scope 'system' (course_id null) và từng lớp có hạn mức.
select b.scope, b.course_id, b.daily_limit,
       coalesce((select sum(a.cost_est) from llm_audit a
                  where a.created_at >= sqlc.arg(day_start) and a.created_at < sqlc.arg(day_end)
                    and (b.scope = 'system' or a.course_id = b.course_id)), 0)::numeric(14,4) as spent,
       coalesce(c.class_code, '') as course_code
  from llm_budgets b left join courses c on c.id = b.course_id
 where b.daily_limit is not null and b.daily_limit > 0;

-- name: ObsAlertRecent :one
-- Đã có thông báo cùng loại + phạm vi (khoá bắt đầu bằng prefix) trong khoảng gần đây chưa (khử trùng 6 giờ).
select exists (select 1 from notifications where type = sqlc.arg(type) and dedupe_key like sqlc.arg(prefix)::text || '%' and created_at >= sqlc.arg(since)) as recent;

-- name: ObsActiveAdmins :many
select id from users where role = 'ADMIN' and status = 'ACTIVE';
