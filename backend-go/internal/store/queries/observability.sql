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

-- name: ObsRequestsPage :many
-- Bảng yêu cầu (ADMIN): KHÔNG cột nội dung. filter: all | problem (status ∉ ok, degraded, cancelled — Q-S11-1: nhất quán với tỷ lệ lỗi và cảnh báo) | privacy (pii_masked_count > 0).
select a.id, a.created_at, a.task, a.lane, a.provider, a.model, a.latency_ms, a.status, a.error_kind, a.degraded, a.fallback_index,
       a.pii_masked_count, a.course_id, c.class_code as course_code
  from llm_audit a left join courses c on c.id = a.course_id
 where (sqlc.narg(from_ts)::timestamptz is null or a.created_at >= sqlc.narg(from_ts))
   and (sqlc.narg(to_ts)::timestamptz is null or a.created_at < sqlc.narg(to_ts))
   and (sqlc.narg(course_id)::uuid is null or a.course_id = sqlc.narg(course_id))
   and (sqlc.narg(task)::text is null or a.task = sqlc.narg(task))
   and (sqlc.arg(filter)::text = 'all'
        or (sqlc.arg(filter)::text = 'problem' and a.status not in ('ok', 'degraded', 'cancelled'))
        or (sqlc.arg(filter)::text = 'privacy' and a.pii_masked_count > 0))
   and (sqlc.narg(cursor_at)::timestamptz is null or (a.created_at, a.id) < (sqlc.narg(cursor_at), sqlc.narg(cursor_id)::uuid))
 order by a.created_at desc, a.id desc
 limit sqlc.arg(page_limit);

-- name: ObsRequestDetail :one
-- Metadata của một yêu cầu + bản prompt đã che (nếu còn); độ tự tin lấy từ tin chat cùng trace khi bản ghi payload chưa có.
select a.id, a.created_at, a.task, a.lane, a.provider, a.model, a.latency_ms, a.status, a.error_kind, a.course_id, a.trace_id,
       (p.audit_id is not null)::boolean as has_payload, p.created_at as payload_at, p.masked_prompt as masked_prompt, p.tools as tools,
       coalesce(coalesce(p.confidence, (select m.confidence from chat_messages m where m.trace_id = a.trace_id and m.confidence is not null limit 1))::text, '')::text as confidence
  from llm_audit a left join llm_request_payloads p on p.audit_id = a.id
 where a.id = sqlc.arg(id);

-- name: ObsPayloadsPurge :execrows
delete from llm_request_payloads where created_at < sqlc.arg(before);

-- name: ObsCourseQuality :one
-- Chất lượng một lớp trong cửa sổ: không danh tính, không nội dung.
-- hidden-ok: chỉ đếm bài AI theo trạng thái xác minh; bài AI bị từ chối thường đã ẩn nhưng vẫn là số đo chất lượng.
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

-- Tổng hợp toàn hệ thống theo (giờ, tác vụ) của cửa sổ [from, to): giờ đủ [lo, hi) từ llm_audit_hourly + phần lẻ thô [from, min(lo, to)) và [hi, to). Mọi hàng gốc thuộc đúng MỘT nguồn.
-- lo / hi do Go tính (giờ nguyên đầu tiên ≥ from; min(watermark, giờ nguyên ≤ to)) để bộ lập kế hoạch thấy khoảng thật: ước lượng sai làm Postgres bật JIT ~100 ms cho truy vấn nhỏ.
-- name: ObsRollupParts :many
select h.hour::timestamptz as hour, h.task, h.requests, h.errors, h.fallbacks, h.cost, h.tokens_in, h.tokens_out, h.pii_masked
  from llm_audit_hourly h where h.hour >= sqlc.arg(lo) and h.hour < sqlc.arg(hi)
union all
select date_trunc('hour', a.created_at), a.task, count(*), count(*) filter (where a.status not in ('ok', 'degraded', 'cancelled')), count(*) filter (where a.fallback_index > 0),
       coalesce(sum(a.cost_est), 0)::numeric(14,4), coalesce(sum(a.tokens_in), 0), coalesce(sum(a.tokens_out), 0), coalesce(sum(a.pii_masked_count), 0)
  from (
    select x.created_at, x.task, x.status, x.fallback_index, x.cost_est, x.tokens_in, x.tokens_out, x.pii_masked_count
      from llm_audit x where x.created_at >= sqlc.arg(from_ts) and x.created_at < sqlc.arg(head_to)
    union all
    select y.created_at, y.task, y.status, y.fallback_index, y.cost_est, y.tokens_in, y.tokens_out, y.pii_masked_count
      from llm_audit y where y.created_at >= sqlc.arg(hi) and y.created_at < sqlc.arg(to_ts)
  ) a
 group by 1, 2;

-- Histogram độ trễ cùng cửa sổ và cùng cách chia hai nguồn như ObsRollupParts.
-- name: ObsRollupLatency :many
select h.hour::timestamptz as hour, h.bucket, h.n
  from llm_audit_hourly_lat h where h.hour >= sqlc.arg(lo) and h.hour < sqlc.arg(hi)
union all
select date_trunc('hour', a.created_at), obs_lat_bucket(a.latency_ms), count(*)
  from (
    select x.created_at, x.latency_ms from llm_audit x where x.created_at >= sqlc.arg(from_ts) and x.created_at < sqlc.arg(head_to)
    union all
    select y.created_at, y.latency_ms from llm_audit y where y.created_at >= sqlc.arg(hi) and y.created_at < sqlc.arg(to_ts)
  ) a
 group by 1, 2;

-- name: ObsRollupWatermark :one
-- Không dòng = chưa chạy lần nào (mọi truy vấn đọc thô).
select watermark from llm_audit_rollup_state;

-- Worker: cộng gộp [from, to) (giờ nguyên) — xoá rồi ghi lại cả khoảng nên chạy lại bao nhiêu lần cũng ra cùng kết quả.
-- name: ObsRollupDeleteHours :exec
delete from llm_audit_hourly where hour >= sqlc.arg(from_ts) and hour < sqlc.arg(to_ts);

-- name: ObsRollupDeleteLat :exec
delete from llm_audit_hourly_lat where hour >= sqlc.arg(from_ts) and hour < sqlc.arg(to_ts);

-- name: ObsRollupInsertHours :exec
insert into llm_audit_hourly (hour, task, requests, errors, fallbacks, cost, tokens_in, tokens_out, pii_masked)
select date_trunc('hour', created_at), task, count(*), count(*) filter (where status not in ('ok', 'degraded', 'cancelled')), count(*) filter (where fallback_index > 0),
       coalesce(sum(cost_est), 0)::numeric(14,4), coalesce(sum(tokens_in), 0), coalesce(sum(tokens_out), 0), coalesce(sum(pii_masked_count), 0)
  from llm_audit where created_at >= sqlc.arg(from_ts) and created_at < sqlc.arg(to_ts)
 group by 1, 2;

-- name: ObsRollupInsertLat :exec
insert into llm_audit_hourly_lat (hour, bucket, n)
select date_trunc('hour', created_at), obs_lat_bucket(latency_ms), count(*)
  from llm_audit where created_at >= sqlc.arg(from_ts) and created_at < sqlc.arg(to_ts)
 group by 1, 2;

-- name: ObsRollupSetWatermark :exec
insert into llm_audit_rollup_state (id, watermark, updated_at) values (true, sqlc.arg(watermark), now())
on conflict (id) do update set watermark = excluded.watermark, updated_at = now();

-- name: ObsRollupLockState :exec
select pg_advisory_xact_lock(hashtext('llm_audit_rollup'));

-- name: ObsRollupPurge :exec
delete from llm_audit_hourly where hour < sqlc.arg(before_ts);

-- name: ObsRollupPurgeLat :exec
delete from llm_audit_hourly_lat where hour < sqlc.arg(before_ts);

-- name: ObsGradingStats :one
-- Hàng chờ chấm bài (US-P7-07 AC10): đếm theo run_state (toàn bộ), thời lượng p50 / p95 giây của bài AI chấm xong chưa ai sửa (updated_at − created_at, gồm cả chờ hàng) tạo từ from_ts.
-- FAILED = đã hết lượt thử (cùng tập với dead-letter `ep:grading.dead`).
select count(*) filter (where run_state = 'QUEUED')::bigint as queued,
       count(*) filter (where run_state = 'RUNNING')::bigint as running,
       count(*) filter (where run_state = 'DONE')::bigint as done,
       count(*) filter (where run_state = 'FAILED')::bigint as failed,
       coalesce(percentile_cont(0.5) within group (order by extract(epoch from updated_at - created_at)) filter (where run_state = 'DONE' and reviewed_at is null and created_at >= sqlc.arg(from_ts)), 0)::float8 as p50_s,
       coalesce(percentile_cont(0.95) within group (order by extract(epoch from updated_at - created_at)) filter (where run_state = 'DONE' and reviewed_at is null and created_at >= sqlc.arg(from_ts)), 0)::float8 as p95_s
from grading_results;
