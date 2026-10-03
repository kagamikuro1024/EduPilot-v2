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

-- Đăng ký / xác minh email (US-P2-03). MSSV chỉ được GHI (InsertPendingStudent), không bao giờ nằm trong điều kiện nối lớp (SRS 4.2.5).

-- name: InsertPendingStudent :one
-- Idempotent theo email: trùng (cả khi hai đăng ký đua nhau) ⇒ không có dòng nào trả về, không lỗi, giao dịch không bị huỷ.
insert into users (email, full_name, role, status, password_hash, student_code)
values (sqlc.arg(email), sqlc.arg(full_name), 'STUDENT', 'PENDING_VERIFICATION', sqlc.arg(password_hash), sqlc.narg(student_code))
on conflict (email) do nothing
returning *;

-- name: ConsumeAuthToken :one
-- Nguyên tử: hai yêu cầu song song cùng token ⇒ đúng một dòng trả về.
update auth_tokens
set used_at = sqlc.arg(now)::timestamptz
where token_hash = sqlc.arg(token_hash)
  and kind = sqlc.arg(kind)
  and used_at is null
  and revoked_at is null
  and expires_at > sqlc.arg(now)::timestamptz
returning user_id;

-- name: MarkEmailVerified :one
update users
set email_verified_at = coalesce(email_verified_at, sqlc.arg(now)::timestamptz),
    status = case when status = 'PENDING_VERIFICATION' then 'ACTIVE'::user_status else status end
where id = sqlc.arg(id)
returning *;

-- name: PromoteUnverifiedRosterEnrollments :execrows
-- Email đã xác minh trùng email trong danh sách lớp ⇒ đẩy các enrollment PENDING (cảnh báo EMAIL_UNVERIFIED) thành ACTIVE.
update enrollments
set status = 'ACTIVE', previous_status = 'PENDING', warning = null, status_changed_at = sqlc.arg(now)::timestamptz, version = version + 1
where user_id = sqlc.arg(user_id) and status = 'PENDING' and warning = 'EMAIL_UNVERIFIED';

-- name: LatestInviterName :one
-- Tên người đã phát lời mời gần nhất cho user (để gửi lại thư mời); không có ⇒ không dòng.
select u.full_name
from auth_tokens t
join users u on u.id = t.created_by
where t.user_id = sqlc.arg(user_id) and t.kind = 'INVITE' and t.created_by is not null
order by t.created_at desc
limit 1;

-- Đặt lại / đổi mật khẩu, quản lý thiết bị (US-P2-04).

-- name: ResetUserPassword :one
-- Đặt lại = đọc thư = kiểm soát hộp thư: xác minh email, mở khoá, bỏ bộ đếm sai.
update users
set password_hash = sqlc.arg(password_hash),
    failed_logins = 0,
    locked_until = null,
    email_verified_at = coalesce(email_verified_at, sqlc.arg(now)::timestamptz),
    status = case when status = 'PENDING_VERIFICATION' then 'ACTIVE'::user_status else status end
where id = sqlc.arg(id)
returning *;

-- name: ChangeUserPassword :exec
update users
set password_hash = sqlc.arg(password_hash), failed_logins = 0, locked_until = null
where id = sqlc.arg(id);

-- name: RevokeUserSessions :many
-- Thu hồi mọi phiên còn sống của người dùng, trừ phiên `except_id` (NULL = không trừ). Trả id để đặt khoá thu hồi ở Redis.
update auth_sessions
set revoked_at = sqlc.arg(now)::timestamptz, revoked_reason = sqlc.arg(reason)::text
where user_id = sqlc.arg(user_id)
  and revoked_at is null
  and (sqlc.narg(except_id)::uuid is null or id <> sqlc.narg(except_id)::uuid)
returning id;

-- name: RevokeOwnSession :one
-- Chỉ phiên của CHÍNH người dùng và còn sống; không khớp ⇒ không dòng (handler trả 404, không lộ tồn tại).
update auth_sessions
set revoked_at = sqlc.arg(now)::timestamptz, revoked_reason = 'REVOKED_BY_USER'
where id = sqlc.arg(id) and user_id = sqlc.arg(user_id) and revoked_at is null
returning id;

-- name: ListOwnSessions :many
-- Phiên còn hiệu lực của chính mình; phiên hiện tại đứng đầu, rồi theo last_used_at giảm dần. Không trả refresh hash / user agent.
select id, device_label, ip, created_at, last_used_at, coalesce(id = sqlc.narg(current_id)::uuid, false)::bool as is_current
from auth_sessions
where user_id = sqlc.arg(user_id)
  and revoked_at is null
  and expires_at > sqlc.arg(now)::timestamptz
  and absolute_expires_at > sqlc.arg(now)::timestamptz
order by coalesce(id = sqlc.narg(current_id)::uuid, false) desc, last_used_at desc
limit 50;
