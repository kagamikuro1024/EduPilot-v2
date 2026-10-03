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
