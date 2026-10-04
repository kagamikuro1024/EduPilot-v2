-- Hàng đợi thư (SRS FEAT-account-security 5.5, 5.6). Không có đường HTTP đọc bảng này.

-- name: InsertMailOutbox :one
insert into mail_outbox (to_addr, template, payload, dedupe_key)
values (sqlc.arg(to_addr), sqlc.arg(template), sqlc.arg(payload), sqlc.narg(dedupe_key))
on conflict (dedupe_key) where dedupe_key is not null do nothing
returning *;

-- name: LockMailOutbox :one
select * from mail_outbox where id = sqlc.arg(id) for update;

-- name: MarkMailSent :exec
update mail_outbox
set status = 'SENT', sent_at = now(), last_error = null
where id = sqlc.arg(id) and status = 'QUEUED';

-- name: RecordMailFailure :exec
update mail_outbox
set attempts = least(attempts + 1, 4),
    last_error = left(sqlc.arg(last_error)::text, 1000)
where id = sqlc.arg(id) and status = 'QUEUED';

-- name: MarkMailDead :exec
update mail_outbox
set status = 'DEAD',
    attempts = least(attempts + 1, 4),
    last_error = left(sqlc.arg(last_error)::text, 1000)
where id = sqlc.arg(id) and status = 'QUEUED';
