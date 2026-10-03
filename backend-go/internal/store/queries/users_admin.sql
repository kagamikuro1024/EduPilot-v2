-- Quản trị người dùng (FEAT-account-security US-P2-06). Danh sách: một truy vấn, khoá phân trang (created_at DESC, id DESC).

-- name: InsertInvitedUser :one
-- Lời mời giảng viên / trợ giảng: chưa có mật khẩu. Trùng email ⇒ không dòng (handler trả 409 CONFLICT).
insert into users (email, full_name, role, status)
values (sqlc.arg(email), sqlc.arg(full_name), sqlc.arg(role), 'INVITED')
on conflict (email) do nothing
returning *;

-- name: ListUsersAdmin :many
select id, email, full_name, role, status, last_login_at, version, created_at
from users
where (sqlc.narg(role)::user_role is null or role = sqlc.narg(role)::user_role)
  and (sqlc.narg(status)::user_status is null or status = sqlc.narg(status)::user_status)
  and (sqlc.narg(name_like)::text is null
       or vn_fold(full_name) like sqlc.narg(name_like)::text escape '\'
       or email like sqlc.narg(email_like)::text escape '\')
  and (sqlc.narg(cur_at)::timestamptz is null or (created_at, id) < (sqlc.narg(cur_at)::timestamptz, sqlc.narg(cur_id)::uuid))
order by created_at desc, id desc
limit sqlc.arg(lim);

-- name: LockUserForUpdate :one
select * from users where id = sqlc.arg(id) for update;

-- name: UpdateUserAdmin :one
-- Khoá lạc quan theo version. Không dòng ⇒ sai version (người gọi đã khoá hàng nên người dùng chắc chắn còn).
update users
set full_name = coalesce(sqlc.narg(full_name), full_name),
    role = coalesce(sqlc.narg(role)::user_role, role),
    status = coalesce(sqlc.narg(status)::user_status, status),
    version = version + 1
where id = sqlc.arg(id) and version = sqlc.arg(version)
returning *;

-- name: CountOtherActiveAdmins :one
select count(*) from users where role = 'ADMIN' and status = 'ACTIVE' and id <> sqlc.arg(id);

-- name: ActivateInvitedUser :one
-- Nhận lời mời: đặt mật khẩu, ACTIVE, email coi như đã xác minh (thư mời tới đúng hộp thư của họ).
update users
set password_hash = sqlc.arg(password_hash), status = 'ACTIVE', email_verified_at = coalesce(email_verified_at, sqlc.arg(now)::timestamptz), version = version + 1
where id = sqlc.arg(id) and status = 'INVITED'
returning *;

-- name: InsertAdminUser :one
-- `gateway admin create`: ADMIN ACTIVE đã xác minh; idempotent theo email.
insert into users (email, full_name, role, status, password_hash, email_verified_at)
values (sqlc.arg(email), sqlc.arg(full_name), 'ADMIN', 'ACTIVE', sqlc.arg(password_hash), sqlc.arg(now)::timestamptz)
on conflict (email) do nothing
returning *;
