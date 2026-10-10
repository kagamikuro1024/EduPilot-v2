-- Threads (US-P3-01; service ở US-P3-06).

-- name: InsertForumThread :one
insert into forum_threads (course_id, author_id, title, body, tags, week_no)
values (sqlc.arg(course_id), sqlc.arg(author_id), sqlc.arg(title), sqlc.arg(body), sqlc.arg(tags), sqlc.narg(week_no))
returning *;

-- name: GetForumThread :one
select * from forum_threads where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and deleted_at is null;

-- name: ListForumThreads :many
select * from forum_threads
where course_id = sqlc.arg(course_id) and deleted_at is null
  and (sqlc.narg(cursor_at)::timestamptz is null or (last_activity_at, id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by last_activity_at desc, id desc
limit sqlc.arg(page_limit);

-- name: InsertForumPost :one
insert into forum_posts (course_id, thread_id, author_id, kind, body, verification_state, citations, confidence)
values (sqlc.arg(course_id), sqlc.arg(thread_id), sqlc.narg(author_id), sqlc.arg(kind), sqlc.arg(body), sqlc.arg(verification_state),
        sqlc.arg(citations), sqlc.narg(confidence))
returning *;

-- name: ListForumPosts :many
select * from forum_posts
where thread_id = sqlc.arg(thread_id) and deleted_at is null
order by created_at, id
limit sqlc.arg(page_limit);

-- name: BumpForumThread :exec
update forum_threads set reply_count = reply_count + 1, last_activity_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- ===== US-P3-06: service Threads =====

-- name: ThreadList :many
-- Một hàng gọn mỗi thread. Sinh viên không thấy bài AI REJECTED / ẩn (thread vẫn hiện); Staff thấy hết. `state`: pending | verified | none.
select t.id, t.title, left(t.body, 160)::text as preview, t.tags, t.week_no, t.state, t.ai_state, t.reply_count, t.last_activity_at, t.created_at, t.author_id,
       u.full_name as author_name, u.role as author_role,
       coalesce((select p.verification_state::text from forum_posts p
         where p.thread_id = t.id and p.course_id = t.course_id and p.kind = 'AI' and p.deleted_at is null
           and (sqlc.arg(is_staff)::bool or (p.hidden_at is null and p.verification_state <> 'REJECTED')) limit 1), '')::text as ai_verification
from forum_threads t
join users u on u.id = t.author_id
where t.course_id = sqlc.arg(course_id) and t.deleted_at is null
  and (sqlc.narg(week_no)::int is null or t.week_no = sqlc.narg(week_no)::int)
  and (sqlc.narg(tag)::text is null or sqlc.narg(tag)::text = any(t.tags))
  and (sqlc.narg(q)::text is null or vn_fold(t.title) like '%' || vn_fold(sqlc.narg(q)::text) || '%' escape '\')
  and (sqlc.narg(state)::text is null
       or (sqlc.narg(state)::text = 'pending' and exists (select 1 from forum_posts p where p.thread_id = t.id and p.kind = 'AI' and p.verification_state = 'PENDING' and p.deleted_at is null and p.hidden_at is null))
       or (sqlc.narg(state)::text = 'verified' and exists (select 1 from forum_posts p where p.thread_id = t.id and p.kind = 'AI' and p.verification_state in ('VERIFIED', 'CORRECTED') and p.deleted_at is null and p.hidden_at is null))
       or (sqlc.narg(state)::text = 'none' and not exists (select 1 from forum_posts p where p.thread_id = t.id and p.kind = 'AI' and p.verification_state <> 'REJECTED' and p.deleted_at is null and p.hidden_at is null)))
  and (sqlc.narg(cursor_at)::timestamptz is null or (t.last_activity_at, t.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by t.last_activity_at desc, t.id desc
limit sqlc.arg(page_limit);

-- name: ThreadDetail :one
select t.id, t.course_id, t.author_id, t.title, t.body, t.tags, t.week_no, t.state, t.ai_state, t.ai_skip_reason, t.similar_of, t.reply_count, t.last_activity_at, t.created_at, t.version,
       u.full_name as author_name, u.role as author_role
from forum_threads t join users u on u.id = t.author_id
where t.course_id = sqlc.arg(course_id) and t.id = sqlc.arg(id) and t.deleted_at is null;

-- name: ThreadPosts :many
-- Sinh viên: bỏ bài REJECTED / ẩn / xoá. Staff: thấy cả bài REJECTED (hiện dòng thu gọn "Đã loại").
select p.id, p.thread_id, p.author_id, p.kind, p.body, p.verification_state, p.citations, p.confidence, p.ai_body, p.hidden_at, p.version, p.verified_at, p.created_at,
       u.full_name as author_name, u.role as author_role
from forum_posts p left join users u on u.id = p.author_id
where p.course_id = sqlc.arg(course_id) and p.thread_id = sqlc.arg(thread_id) and p.deleted_at is null
  and (sqlc.arg(is_staff)::bool or (p.hidden_at is null and p.verification_state <> 'REJECTED'))
  and (sqlc.narg(cursor_at)::timestamptz is null or (p.created_at, p.id) > (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by p.created_at, p.id
limit sqlc.arg(page_limit);

-- name: ThreadInsert :one
insert into forum_threads (course_id, author_id, title, body, tags, week_no, embedding)
values (sqlc.arg(course_id), sqlc.arg(author_id), sqlc.arg(title), sqlc.arg(body), sqlc.arg(tags), sqlc.narg(week_no), sqlc.narg(embedding))
returning id, created_at;

-- name: PostGet :one
select p.id, p.course_id, p.thread_id, p.author_id, p.kind, p.body, p.verification_state, p.ai_body, p.version, t.author_id as thread_author_id
from forum_posts p join forum_threads t on t.course_id = p.course_id and t.id = p.thread_id
where p.course_id = sqlc.arg(course_id) and p.id = sqlc.arg(id) and p.deleted_at is null and t.deleted_at is null
for update of p;

-- name: PostDecide :one
-- Chuyển trạng thái bài AI (đã khoá hàng): verify / correct / reject. `ai_body` giữ bản AI gốc ở lần sửa đầu.
update forum_posts
set verification_state = sqlc.arg(state)::post_verification,
    body = coalesce(sqlc.narg(new_body)::text, body),
    ai_body = case when sqlc.narg(new_body)::text is not null then coalesce(ai_body, body) else ai_body end,
    verified_by = sqlc.arg(actor), verified_at = now(), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and kind = 'AI'
returning id, thread_id, author_id, kind, body, verification_state, citations, confidence, ai_body, hidden_at, version, verified_at, created_at;

-- name: ThreadGetAuthor :one
select author_id, course_id from forum_threads where id = sqlc.arg(id) and deleted_at is null;

-- name: ThreadSimilar :many
select t.id, t.title, left(t.body, 160)::text as preview, (1 - (t.embedding <=> src.embedding))::float8 as cosine
from forum_threads t, forum_threads src
where src.course_id = sqlc.arg(course_id) and src.id = sqlc.arg(id) and t.course_id = src.course_id and t.id <> src.id
  and t.deleted_at is null and t.embedding is not null and src.embedding is not null
  and not exists (select 1 from forum_posts p where p.thread_id = t.id and p.kind = 'AI' and (p.verification_state = 'REJECTED' or p.hidden_at is not null))
  and 1 - (t.embedding <=> src.embedding) >= sqlc.arg(min_cosine)::float8
order by t.embedding <=> src.embedding
limit sqlc.arg(lim);

-- name: ThreadAnswerLoad :one
-- Việc AI trả lời: nạp thread (không lọc lớp: id đến từ hàng đợi nội bộ), kèm vectơ đã có.
select id, course_id, author_id, title, body, ai_state, embedding from forum_threads where id = sqlc.arg(id) and deleted_at is null;

-- name: ThreadSetEmbedding :exec
update forum_threads set embedding = sqlc.arg(embedding) where id = sqlc.arg(id) and embedding is null;

-- name: ThreadSetSimilarOf :exec
update forum_threads t set similar_of = (
  select o.id from forum_threads o
  where o.course_id = t.course_id and o.id <> t.id and o.deleted_at is null and o.embedding is not null and t.embedding is not null
    and 1 - (o.embedding <=> t.embedding) >= sqlc.arg(min_cosine)::float8
  order by o.embedding <=> t.embedding limit 1)
where t.id = sqlc.arg(id) and t.similar_of is null;

-- name: ThreadMarkAnswered :execrows
update forum_threads set ai_state = 'ANSWERED', reply_count = reply_count + 1, last_activity_at = now()
where id = sqlc.arg(id) and ai_state = 'PENDING';

-- name: ThreadMarkSkipped :execrows
update forum_threads set ai_state = 'SKIPPED', ai_skip_reason = sqlc.arg(reason)
where id = sqlc.arg(id) and ai_state = 'PENDING';
