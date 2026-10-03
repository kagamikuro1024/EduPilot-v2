-- Bảng `users` ở PG chỉ dùng cho test và các phase sau (không endpoint nghiệp vụ nào ghi).

-- name: InsertUser :one
insert into users (email, full_name, role, status, password_hash, student_code)
values (
    sqlc.arg(email),
    sqlc.arg(full_name),
    sqlc.arg(role),
    sqlc.arg(status),
    sqlc.narg(password_hash),
    sqlc.narg(student_code)
)
returning *;

-- name: GetUser :one
select * from users where id = sqlc.arg(id);

-- name: GetUserByEmail :one
select * from users where email = sqlc.arg(email);
