-- Việc dài (SRS 5.4). Phân trang theo con trỏ khoá (created_at, id) — không bao giờ nhảy trang theo số.

-- name: InsertJob :one
insert into jobs (kind, owner_id)
values (sqlc.arg(kind), sqlc.arg(owner_id))
returning *;

-- name: GetJob :one
select * from jobs where id = sqlc.arg(id);

-- name: GetJobForOwner :one
select * from jobs where id = sqlc.arg(id) and owner_id = sqlc.arg(owner_id);

-- name: UpdateJobProgress :one
-- Tiến độ chỉ tăng (greatest); việc đang chạy thì chuyển QUEUED -> RUNNING.
update jobs
set progress = greatest(progress, sqlc.arg(progress)::smallint),
    status = 'RUNNING'
where id = sqlc.arg(id) and status in ('QUEUED', 'RUNNING')
returning *;

-- name: MarkJobSucceeded :one
update jobs
set status = 'SUCCEEDED', progress = 100, result = sqlc.narg(result), finished_at = now()
where id = sqlc.arg(id) and finished_at is null
returning *;

-- name: MarkJobFailed :one
update jobs
set status = 'FAILED', error = sqlc.narg(error), finished_at = now()
where id = sqlc.arg(id) and finished_at is null
returning *;

-- name: ListJobsByOwner :many
select * from jobs
where owner_id = sqlc.arg(owner_id)
  and (
    sqlc.narg(cursor_created_at)::timestamptz is null
    or (created_at, id) < (sqlc.narg(cursor_created_at)::timestamptz, sqlc.narg(cursor_id)::uuid)
  )
order by created_at desc, id desc
limit sqlc.arg(row_limit);
