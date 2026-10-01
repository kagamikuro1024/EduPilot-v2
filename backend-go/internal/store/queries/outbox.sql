-- Outbox (SRS 3.2, 5.3): relay lấy dòng tới hạn bằng FOR UPDATE SKIP LOCKED, consumer đánh dấu
-- dispatched / failed (backoff) / dead. `attempts` tối đa 4 (ràng buộc outbox_attempts_chk).

-- name: InsertOutbox :one
insert into outbox (topic, payload)
values (sqlc.arg(topic), sqlc.arg(payload))
returning *;

-- name: GetOutbox :one
select * from outbox where id = sqlc.arg(id);

-- name: SelectDueOutbox :many
select * from outbox
where dispatched_at is null
  and dead_at is null
  and enqueued_at is null
  and next_attempt_at <= now()
order by next_attempt_at, id
limit sqlc.arg(row_limit)
for update skip locked;

-- name: MarkOutboxEnqueued :execrows
update outbox
set enqueued_at = now()
where id = sqlc.arg(id) and enqueued_at is null and dispatched_at is null and dead_at is null;

-- name: MarkOutboxDispatched :execrows
update outbox
set dispatched_at = now(), enqueued_at = null
where id = sqlc.arg(id) and dispatched_at is null and dead_at is null;

-- name: MarkOutboxFailed :one
update outbox
set attempts = attempts + 1,
    last_error = left(sqlc.arg(last_error)::text, 1000),
    enqueued_at = null,
    next_attempt_at = now() + (sqlc.arg(backoff_ms)::bigint * interval '1 millisecond')
where id = sqlc.arg(id) and dispatched_at is null and dead_at is null
returning attempts;

-- name: MarkOutboxDead :execrows
update outbox
set dead_at = now(),
    last_error = left(sqlc.arg(last_error)::text, 1000),
    enqueued_at = null
where id = sqlc.arg(id) and dispatched_at is null and dead_at is null;

-- name: RequeueStaleOutbox :many
update outbox
set enqueued_at = null
where enqueued_at is not null
  and dispatched_at is null
  and dead_at is null
  and enqueued_at < now() - (sqlc.arg(stale_after_ms)::bigint * interval '1 millisecond')
returning id;
