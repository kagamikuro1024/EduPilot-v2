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
