-- Cấu hình cổng LLM (FEAT-llm-gateway 5.2). Khoá API chỉ đi vào/ra dưới dạng bytea đã mã hoá.

-- name: ListLLMProviders :many
select * from llm_providers order by created_at, id;

-- name: GetLLMProvider :one
select * from llm_providers where id = $1;

-- name: CountLLMProviders :one
select count(*) from llm_providers;

-- name: InsertLLMProvider :one
insert into llm_providers (id, type, name, base_url, api_key_enc, enabled, rpm_limit, tpm_limit, last_test_ok, last_test_at, last_test_error)
values (
    sqlc.arg(id), sqlc.arg(type), sqlc.arg(name), sqlc.narg(base_url), sqlc.narg(api_key_enc), sqlc.arg(enabled),
    sqlc.narg(rpm_limit), sqlc.narg(tpm_limit), sqlc.narg(last_test_ok), sqlc.narg(last_test_at), sqlc.narg(last_test_error)
)
returning *;

-- Sửa có khoá lạc quan: 0 dòng = sai version hoặc không tồn tại. `api_key_enc` NULL = giữ khoá cũ.
-- name: UpdateLLMProvider :one
update llm_providers
   set type = sqlc.arg(type),
       name = sqlc.arg(name),
       base_url = sqlc.narg(base_url),
       api_key_enc = coalesce(sqlc.narg(api_key_enc), api_key_enc),
       enabled = sqlc.arg(enabled),
       rpm_limit = sqlc.narg(rpm_limit),
       tpm_limit = sqlc.narg(tpm_limit),
       last_test_ok = case when sqlc.arg(reset_test)::boolean then null else last_test_ok end,
       last_test_at = case when sqlc.arg(reset_test)::boolean then null else last_test_at end,
       last_test_error = case when sqlc.arg(reset_test)::boolean then null else last_test_error end,
       version = version + 1
 where id = sqlc.arg(id) and version = sqlc.arg(version)
returning *;

-- Kết quả Test không đổi version (không phải sửa cấu hình).
-- name: UpdateLLMProviderTest :one
update llm_providers
   set last_test_ok = sqlc.arg(last_test_ok), last_test_at = sqlc.arg(last_test_at), last_test_error = sqlc.narg(last_test_error)
 where id = sqlc.arg(id)
returning *;

-- name: DeleteLLMProvider :execrows
delete from llm_providers where id = $1;

-- name: ListLLMModels :many
select * from llm_models order by provider_id, created_at, id;

-- name: ListLLMModelsByProvider :many
select * from llm_models where provider_id = $1 order by created_at, id;

-- name: ListLLMModelsByIDs :many
select * from llm_models where id = any(sqlc.arg(ids)::uuid[]);

-- name: CountLLMModelsByProvider :one
select count(*) from llm_models where provider_id = $1;

-- name: UpsertLLMModel :one
insert into llm_models (provider_id, model, kind, dims, price_in, price_out, enabled)
values (sqlc.arg(provider_id), sqlc.arg(model), sqlc.arg(kind), sqlc.narg(dims), sqlc.arg(price_in), sqlc.arg(price_out), sqlc.arg(enabled))
on conflict (provider_id, model) do update
   set kind = excluded.kind, dims = excluded.dims, price_in = excluded.price_in, price_out = excluded.price_out, enabled = excluded.enabled
returning *;

-- name: DeleteLLMModelsNotIn :many
delete from llm_models where provider_id = sqlc.arg(provider_id) and not (model = any(sqlc.arg(keep)::text[]))
returning id, model;

-- Tác vụ đang dùng nhà cung cấp / mô hình (chặn xoá).
-- name: LLMTasksUsingProvider :many
select distinct r.task
  from llm_task_routes r join llm_models m on m.id = r.model_id
 where m.provider_id = $1
 order by r.task;

-- name: LLMTasksUsingModels :many
select distinct task from llm_task_routes where model_id = any(sqlc.arg(ids)::uuid[]) order by task;

-- name: ListLLMRoutes :many
select r.id, r.task, r.model_id, r.fallback_order, r.params, r.version,
       m.provider_id, m.model, m.kind, m.dims, m.price_in, m.price_out, m.enabled as model_enabled, p.name as provider_name, p.enabled as provider_enabled
  from llm_task_routes r
  join llm_models m on m.id = r.model_id
  join llm_providers p on p.id = m.provider_id
 order by r.task, r.fallback_order;

-- name: ListLLMRoutesByTask :many
select * from llm_task_routes where task = $1 order by fallback_order;

-- name: DeleteLLMRoutesByTask :exec
delete from llm_task_routes where task = $1;

-- name: InsertLLMRoute :one
insert into llm_task_routes (task, model_id, fallback_order, params, version)
values (sqlc.arg(task), sqlc.arg(model_id), sqlc.arg(fallback_order), sqlc.arg(params), sqlc.arg(version))
returning *;

-- Khoá tư vấn theo tác vụ: hai Admin tạo tuyến lần đầu cùng lúc không đua nhau.
-- name: LockLLMTask :exec
select pg_advisory_xact_lock(hashtextextended('llm_route:' || sqlc.arg(task)::text, 0));

-- name: GetLLMBudget :one
select * from llm_budgets
 where scope = sqlc.arg(scope) and course_id is not distinct from sqlc.narg(course_id);

-- name: InsertLLMBudget :one
insert into llm_budgets (scope, course_id, daily_limit, monthly_limit)
values (sqlc.arg(scope), sqlc.narg(course_id), sqlc.narg(daily_limit), sqlc.narg(monthly_limit))
returning *;

-- name: UpdateLLMBudget :one
update llm_budgets
   set daily_limit = sqlc.narg(daily_limit), monthly_limit = sqlc.narg(monthly_limit), version = version + 1
 where id = sqlc.arg(id) and version = sqlc.arg(version)
returning *;

-- Mức dùng (GET usage): gom theo tác vụ hoặc theo ngày (Asia/Ho_Chi_Minh); `course_id` NULL = toàn hệ thống.
-- name: LLMUsageByTask :many
select task as key, count(*)::bigint as calls, coalesce(sum(tokens_in), 0)::bigint as tokens_in, coalesce(sum(tokens_out), 0)::bigint as tokens_out,
       coalesce(sum(cost_est), 0)::numeric(14,4) as cost_est,
       coalesce(round(percentile_cont(0.5) within group (order by latency_ms)), 0)::bigint as latency_p50_ms,
       coalesce(round(percentile_cont(0.95) within group (order by latency_ms)), 0)::bigint as latency_p95_ms,
       count(*) filter (where status in ('error', 'timeout', 'rate_limited', 'circuit_open', 'not_configured'))::bigint as errors,
       count(*) filter (where degraded)::bigint as degraded
  from llm_audit
 where created_at >= sqlc.arg(from_ts) and created_at < sqlc.arg(to_ts)
   and (sqlc.narg(course_id)::uuid is null or course_id = sqlc.narg(course_id))
 group by task
 order by task;

-- name: LLMUsageByDay :many
select to_char(created_at at time zone 'Asia/Ho_Chi_Minh', 'YYYY-MM-DD') as key, count(*)::bigint as calls,
       coalesce(sum(tokens_in), 0)::bigint as tokens_in, coalesce(sum(tokens_out), 0)::bigint as tokens_out,
       coalesce(sum(cost_est), 0)::numeric(14,4) as cost_est,
       coalesce(round(percentile_cont(0.5) within group (order by latency_ms)), 0)::bigint as latency_p50_ms,
       coalesce(round(percentile_cont(0.95) within group (order by latency_ms)), 0)::bigint as latency_p95_ms,
       count(*) filter (where status in ('error', 'timeout', 'rate_limited', 'circuit_open', 'not_configured'))::bigint as errors,
       count(*) filter (where degraded)::bigint as degraded
  from llm_audit
 where created_at >= sqlc.arg(from_ts) and created_at < sqlc.arg(to_ts)
   and (sqlc.narg(course_id)::uuid is null or course_id = sqlc.narg(course_id))
 group by 1
 order by 1;

-- Ghi llm_audit theo lô bằng COPY — cổng llm ghi bất đồng bộ, một lần cho cả lô.
-- name: InsertLLMAudit :copyfrom
insert into llm_audit (task, lane, provider, model, tokens_in, tokens_out, latency_ms, queue_wait_ms, attempts, fallback_index,
                       cost_est, status, error_kind, degraded, pii_masked_count, user_id, course_id, trace_id, created_at)
values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19);

-- Ngân sách: tổng chi phí thực tế từ llm_audit để đối soát bộ đếm Redis.
-- name: LLMCostSum :one
select coalesce(sum(cost_est), 0)::numeric(14,4) as total
  from llm_audit
 where created_at >= sqlc.arg(from_ts) and created_at < sqlc.arg(to_ts)
   and (sqlc.narg(course_id)::uuid is null or course_id = sqlc.narg(course_id));
