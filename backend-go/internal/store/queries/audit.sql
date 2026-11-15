-- `audit_log` chỉ-thêm: chỉ có INSERT (trigger chặn UPDATE/DELETE/TRUNCATE).

-- name: InsertAuditLog :one
insert into audit_log (course_id, actor_id, entity, entity_id, action, before, after, trace_id)
values (
    sqlc.narg(course_id),
    sqlc.narg(actor_id),
    sqlc.arg(entity),
    sqlc.arg(entity_id),
    sqlc.arg(action),
    sqlc.narg(before),
    sqlc.narg(after),
    sqlc.narg(trace_id)
)
returning *;

-- name: AuditList :many
-- Trang nhật ký cho ADMIN (US-P10-14): lọc AND, (created_at, id) giảm dần, con trỏ khoá (created_at, id). Chỉ tên khoá cấp 1 của before ∪ after — KHÔNG trả giá trị.
select a.id, a.created_at, a.actor_id, u.full_name as actor_name, u.role as actor_role, a.entity, a.entity_id, a.action, a.course_id, a.trace_id,
       coalesce(array(
         select k from (
           select jsonb_object_keys(case when jsonb_typeof(a.before) = 'object' then a.before else '{}'::jsonb end) as k
           union
           select jsonb_object_keys(case when jsonb_typeof(a.after) = 'object' then a.after else '{}'::jsonb end)
         ) keys order by k limit 20
       ), '{}')::text[] as changed_fields
from audit_log a
left join users u on u.id = a.actor_id
where a.created_at >= sqlc.arg(from_ts) and a.created_at < sqlc.arg(to_ts)
  and (sqlc.narg(entity)::text is null or a.entity = sqlc.narg(entity))
  and (sqlc.narg(actor_id)::uuid is null or a.actor_id = sqlc.narg(actor_id))
  and (sqlc.narg(cursor_at)::timestamptz is null or (a.created_at, a.id) < (sqlc.narg(cursor_at), sqlc.narg(cursor_id)::uuid))
order by a.created_at desc, a.id desc
limit sqlc.arg(max_rows);
