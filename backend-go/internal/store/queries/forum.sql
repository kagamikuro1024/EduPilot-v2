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
