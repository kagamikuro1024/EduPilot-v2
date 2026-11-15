-- Cấu hình decision (SRS FEAT-decision-model 5): ảnh chụp đọc MỘT lần mỗi lần làm mới, không đọc trên đường nóng (AC5 US-DM-02).

-- name: DecisionGatesAll :many
select gate, enabled, threshold, timeout_ms, version from decision_gates order by gate;

-- name: DecisionCourseGatesEnabled :many
-- Mọi cặp (lớp, cổng) đang bật; không có hàng = tắt.
select course_id, gate from course_decision_gates where enabled;

-- name: DecisionGatesWithCounts :many
-- Ba cổng kèm số lớp đang bật / tổng số lớp (màn cấu hình, bảng quan sát).
select g.gate, g.enabled, g.threshold, g.timeout_ms, g.version,
       (select count(*) from course_decision_gates c where c.gate = g.gate and c.enabled)::int as courses_enabled,
       (select count(*) from courses)::int as courses_total
  from decision_gates g
 order by g.gate;

-- name: DecisionGateUpdate :one
-- Sửa có kiểm phiên bản: không khớp → không hàng (người gọi phân biệt 404 / 409).
update decision_gates
   set enabled    = coalesce(sqlc.narg(enabled), enabled),
       threshold  = coalesce(sqlc.narg(threshold), threshold),
       timeout_ms = coalesce(sqlc.narg(timeout_ms), timeout_ms),
       version    = version + 1,
       updated_by = sqlc.narg(actor),
       updated_at = now()
 where gate = sqlc.arg(gate) and version = sqlc.arg(version)
returning gate, enabled, threshold, timeout_ms, version;

-- name: DecisionGateGet :one
select gate, enabled, threshold, timeout_ms, version from decision_gates where gate = $1;

-- name: DecisionCoursesPage :many
-- Lớp + công tắc của cổng; keyset (created_at, id), tìm theo tên / mã lớp. `q` đã thoát ký tự đặc biệt ở Go.
select c.id, c.name, c.class_code, c.created_at,
       coalesce(d.enabled, false) as enabled,
       coalesce(d.version, 0)::int as version
  from courses c
  left join course_decision_gates d on d.course_id = c.id and d.gate = sqlc.arg(gate)
 where (sqlc.arg(q)::text = '' or c.name ilike '%' || sqlc.arg(q)::text || '%' or c.class_code ilike '%' || sqlc.arg(q)::text || '%')
   and (sqlc.narg(cur_t)::timestamptz is null or (c.created_at, c.id) > (sqlc.narg(cur_t)::timestamptz, sqlc.narg(cur_id)::uuid))
 order by c.created_at, c.id
 limit sqlc.arg(row_limit);

-- name: CourseGateGet :one
select course_id, gate, enabled, version from course_decision_gates where course_id = $1 and gate = $2;

-- name: CourseGateInsert :one
-- Hàng chưa có = phiên bản 0: chèn phiên bản 1. Đua nhau chèn → hàng thứ hai không vào (người gọi trả 409).
insert into course_decision_gates (course_id, gate, enabled, version, updated_by)
values ($1, $2, $3, 1, sqlc.narg(actor))
on conflict (course_id, gate) do nothing
returning course_id, gate, enabled, version;

-- name: CourseGateUpdate :one
update course_decision_gates
   set enabled = $3, version = version + 1, updated_by = sqlc.narg(actor), updated_at = now()
 where course_id = $1 and gate = $2 and version = $4
returning course_id, gate, enabled, version;

-- name: CourseGatesOf :many
select gate, enabled, version from course_decision_gates where course_id = $1;

-- name: CourseExists :one
select exists (select 1 from courses where id = $1);

-- name: DecisionObservability :many
-- Chỉ số theo cổng trong cửa sổ; chỉ đếm, không nội dung. Phân vị liên tục trên độ trễ của CHÍNH lời gọi decision.
select cascade_gate::text as gate,
       count(*)::int as calls,
       (count(*) filter (where cascade_path = 'FAST'))::int as fast,
       (count(*) filter (where cascade_path = 'FALLBACK'))::int as fallback,
       (count(*) filter (where cascade_path = 'FAIL_CLOSED'))::int as fail_closed,
       (count(*) filter (where status not in ('ok', 'degraded')))::int as errors,
       coalesce(percentile_cont(0.5) within group (order by latency_ms), 0)::float8 as p50_ms,
       coalesce(percentile_cont(0.95) within group (order by latency_ms), 0)::float8 as p95_ms,
       coalesce(sum(cost_est), 0)::numeric as cost
  from llm_audit
 where task = 'DECIDE' and cascade_gate is not null and created_at >= $1
 group by cascade_gate;

-- name: DecisionBudgetBlockedSince :one
select count(*)::int from llm_audit where task = 'DECIDE' and status = 'budget_blocked' and created_at >= $1;

-- name: DecisionProviderGet :one
-- Provider `decision` đang bật (tối đa một, chỉ mục unique) kèm mô hình đang bật; không có → ErrNoRows.
select p.id, p.name, (p.api_key_enc is not null)::boolean as has_key, m.model
  from llm_providers p
  join llm_models m on m.provider_id = p.id and m.kind = 'decision' and m.enabled
 where p.type = 'decision' and p.enabled
 limit 1;

-- name: DecisionAlertRecent :one
-- Đã có cảnh báo cùng loại + cổng trong khoảng khử trùng chưa (khoá khử trùng bắt đầu bằng tiền tố).
select exists (select 1 from notifications where type = $1 and dedupe_key like sqlc.arg(prefix)::text || '%' and created_at >= $2);

-- name: DecisionActiveAdmins :many
select id from users where role = 'ADMIN' and status = 'ACTIVE' order by id;
