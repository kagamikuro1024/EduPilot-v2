-- Dự phòng bền cho Redis (SRS 5.5): khoá theo (user_id, endpoint, key).

-- name: GetIdempotencyKey :one
select * from idempotency_keys
where user_id = sqlc.arg(user_id) and endpoint = sqlc.arg(endpoint) and key = sqlc.arg(key);

-- name: InsertIdempotencyKey :execrows
insert into idempotency_keys (user_id, endpoint, key, request_hash, status_code, response)
values (
    sqlc.arg(user_id),
    sqlc.arg(endpoint),
    sqlc.arg(key),
    sqlc.arg(request_hash),
    sqlc.arg(status_code),
    sqlc.arg(response)
)
on conflict (user_id, endpoint, key) do nothing;
