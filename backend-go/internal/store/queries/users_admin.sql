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

-- Hồ sơ và tuỳ chọn của chính mình (US-P2-07). `student_code` chỉ được GHI ở đây; không điều kiện nối lớp nào dùng nó (SRS 4.2.5).

-- name: UpdateProfile :one
-- Khoá lạc quan theo version. Không dòng ⇒ sai version (handler đọc bản hiện hành để trả 409).
update users
set full_name = coalesce(sqlc.narg(full_name), full_name),
    student_code = case when sqlc.arg(set_student_code)::bool then sqlc.narg(student_code) else student_code end,
    version = version + 1
where id = sqlc.arg(id) and version = sqlc.arg(version)
returning *;

-- name: EnsureUserSettings :one
-- Tạo lười khi đọc lần đầu; luôn trả dòng hiện có.
with ins as (
    insert into user_settings (user_id) values (sqlc.arg(user_id)) on conflict (user_id) do nothing returning *
)
select * from ins
union all
select * from user_settings where user_id = sqlc.arg(user_id) and not exists (select 1 from ins);

-- name: UpdateUserSettings :one
update user_settings
set notify_ticket_by_mail = coalesce(sqlc.narg(notify_ticket_by_mail), notify_ticket_by_mail),
    notify_answer_by_mail = coalesce(sqlc.narg(notify_answer_by_mail), notify_answer_by_mail),
    remind_deadline_by_mail = coalesce(sqlc.narg(remind_deadline_by_mail), remind_deadline_by_mail),
    -- US-P8-03: gộp từng khoá của preferences.reminders (khoá vắng = giữ nguyên)
    preferences = case when sqlc.narg(reminders)::jsonb is null then preferences
                  else jsonb_set(preferences, '{reminders}', coalesce(preferences->'reminders', '{}'::jsonb) || sqlc.narg(reminders)::jsonb) end,
    version = version + 1
where user_id = sqlc.arg(user_id) and version = sqlc.arg(version)
returning *;
