-- Token một lần (SRS FEAT-account-security 5.3). CHỈ gói internal/auth được gọi các truy vấn này
-- (TestOnlyAuthPackageTouchesTokenTables). Bản rõ không bao giờ vào DB: chỉ nhận sha256 hex.

-- name: InsertAuthToken :one
insert into auth_tokens (user_id, kind, token_hash, expires_at, created_by)
values (sqlc.arg(user_id), sqlc.arg(kind), sqlc.arg(token_hash), sqlc.arg(expires_at), sqlc.narg(created_by))
returning *;

-- name: RevokeUnusedAuthTokens :execrows
update auth_tokens
set revoked_at = now()
where user_id = sqlc.arg(user_id)
  and kind = sqlc.arg(kind)
  and used_at is null
  and revoked_at is null;

-- name: GetAuthTokenByHash :one
select * from auth_tokens where token_hash = sqlc.arg(token_hash);

-- Phiên đăng nhập (SRS FEAT-account-security 5.2). Thời điểm do ứng dụng truyền vào (đồng hồ giả được trong test).

-- name: InsertAuthSession :one
insert into auth_sessions (user_id, refresh_hash, user_agent, device_label, ip, created_at, last_used_at, expires_at, absolute_expires_at)
values (
    sqlc.arg(user_id), sqlc.arg(refresh_hash), sqlc.narg(user_agent), sqlc.narg(device_label), sqlc.narg(ip),
    sqlc.arg(now), sqlc.arg(now), sqlc.arg(expires_at), sqlc.arg(absolute_expires_at)
)
returning *;

-- name: LockAuthSessionByRefresh :one
-- Khoá hàng của phiên có refresh hiện tại HOẶC thế hệ trước; hai yêu cầu cùng phiên chạy tuần tự.
select * from auth_sessions
where refresh_hash = sqlc.arg(hash) or prev_refresh_hash = sqlc.arg(hash)
for update;

-- name: RotateAuthSession :exec
update auth_sessions
set prev_refresh_hash = refresh_hash,
    refresh_hash = sqlc.arg(new_hash),
    rotated_at = sqlc.arg(now)::timestamptz,
    last_used_at = sqlc.arg(now)::timestamptz,
    expires_at = sqlc.arg(expires_at)
where id = sqlc.arg(id);

-- name: RevokeAuthSession :execrows
update auth_sessions
set revoked_at = sqlc.arg(now)::timestamptz, revoked_reason = sqlc.arg(reason)::text
where id = sqlc.arg(id) and revoked_at is null;

-- name: GetAuthSessionRevokedReason :one
select revoked_reason from auth_sessions where id = sqlc.arg(id);

-- name: InsertLoginAttempt :exec
insert into login_attempts (email_hash, user_id, ip, user_agent, outcome, created_at)
values (sqlc.arg(email_hash), sqlc.narg(user_id), sqlc.narg(ip), sqlc.narg(user_agent), sqlc.arg(outcome), sqlc.arg(now));
