-- Thông báo (SRS FEAT-escalation-notify 4.4). NƠI DUY NHẤT được `insert into notifications` (TestNotificationsOnlyViaNotify): mọi nơi khác đi qua gói `notify`.

-- name: NotifyInsert :one
-- Idempotent theo (user_id, dedupe_key): trùng thì KHÔNG trả hàng (pgx.ErrNoRows) — không phát lại SSE.
insert into notifications (user_id, course_id, type, title, body, link, dedupe_key)
values (sqlc.arg(user_id), sqlc.narg(course_id), sqlc.arg(type), sqlc.arg(title), sqlc.narg(body), sqlc.narg(link), sqlc.narg(dedupe_key))
on conflict (user_id, dedupe_key) where dedupe_key is not null do nothing
returning id, created_at;

-- name: NotifyInsertMany :many
-- MỘT câu cho cả danh sách người nhận (1.000 sinh viên = 1 INSERT); khoá khử trùng `<dedupe_prefix>:<user_id>`. Trả các dòng MỚI để phát SSE.
insert into notifications (user_id, course_id, type, title, body, link, dedupe_key)
select u, sqlc.narg(course_id), sqlc.arg(type), sqlc.arg(title), sqlc.narg(body), sqlc.narg(link), sqlc.arg(dedupe_prefix)::text || ':' || u::text
from unnest(sqlc.arg(user_ids)::text[]::uuid[]) as u
on conflict (user_id, dedupe_key) where dedupe_key is not null do nothing
returning id, user_id, created_at;

-- name: NotifyUnreadCounts :many
select user_id, count(*)::int as n from notifications
where user_id = any(sqlc.arg(user_ids)::text[]::uuid[]) and read_at is null
group by user_id;

-- name: NotifyAggLock :exec
-- Khoá giao dịch theo (lớp, loại): hai handler song song cho số cuối đúng (SRS 4.4.3, TLR-7).
select pg_advisory_xact_lock(hashtextextended('notify_agg:' || sqlc.arg(course_id)::text || ':' || sqlc.arg(type)::text, 0));

-- name: NotifyAggGet :one
select id, title, read_at from notifications where user_id = sqlc.arg(user_id) and dedupe_key = sqlc.arg(dedupe_key) for update;

-- name: NotifyAggInsert :one
insert into notifications (user_id, course_id, type, title, body, link, dedupe_key)
values (sqlc.arg(user_id), sqlc.arg(course_id), sqlc.arg(type), sqlc.arg(title), sqlc.narg(body), sqlc.narg(link), sqlc.arg(dedupe_key))
returning id, created_at;

-- name: NotifyAggRaise :one
-- Số TĂNG: dòng gộp nổi lên (created_at mới) và chưa đọc trở lại.
update notifications set title = sqlc.arg(title), created_at = now(), read_at = null where id = sqlc.arg(id) returning id, created_at;

-- name: NotifyAggRetitle :exec
-- Số giảm: đổi chữ, GIỮ read_at (người đã đọc không bị "chưa đọc" lại).
update notifications set title = sqlc.arg(title) where id = sqlc.arg(id);

-- name: NotifyAggMarkRead :exec
update notifications set read_at = coalesce(read_at, now()) where id = sqlc.arg(id);

-- name: NotifyRow :one
select id, type, title, body, link, course_id, created_at from notifications where id = sqlc.arg(id);

-- name: NotifyMarkReadByLink :execrows
-- Đánh dấu đã đọc mọi thông báo CHƯA đọc CỦA CHÍNH MÌNH có `link` trùng (chat gọi khi mở phiên có tin của giảng viên).
update notifications set read_at = now() where user_id = sqlc.arg(user_id) and link = sqlc.arg(link) and read_at is null;
