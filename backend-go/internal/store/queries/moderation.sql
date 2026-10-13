-- Kiểm duyệt Threads (US-P4-06, SRS FEAT-escalation-notify 4.10). Mọi truy vấn ở đây là đường Staff / chủ bài / ghi nội bộ (allowlist TLR-22: `-- hidden-ok`).

-- name: ReportLock :exec
select pg_advisory_xact_lock(hashtextextended('report:' || sqlc.arg(target_id)::text, 0));

-- name: ReportInsert :execrows
insert into post_reports (course_id, thread_id, post_id, reporter_id, reason, note)
values (sqlc.arg(course_id), sqlc.narg(thread_id), sqlc.narg(post_id), sqlc.arg(reporter_id), sqlc.arg(reason), sqlc.narg(note))
on conflict (reporter_id, target_id) do nothing;

-- name: ReportCountOpenDistinct :one
select count(distinct reporter_id)::int from post_reports where target_id = sqlc.arg(target_id) and status = 'OPEN';

-- name: ReportRecentByUser :one
select count(*)::int from post_reports where reporter_id = sqlc.arg(reporter_id) and created_at > sqlc.arg(since)::timestamptz;

-- name: ReportDeleteOwn :execrows
-- Rút báo cáo ("Hoàn tác"): chỉ của chính mình, còn OPEN, trong cửa sổ `since`.
delete from post_reports
where reporter_id = sqlc.arg(reporter_id) and target_id = sqlc.arg(target_id) and course_id = sqlc.arg(course_id) and status = 'OPEN' and created_at > sqlc.arg(since)::timestamptz;

-- name: ReportHandleOpen :execrows
update post_reports set status = sqlc.arg(status)::post_report_status, handled_by = sqlc.arg(handled_by), handled_at = now()
where target_id = sqlc.arg(target_id) and course_id = sqlc.arg(course_id) and status = 'OPEN';

-- name: ModerationPendingCount :one
-- hidden-ok: đếm cho Staff. "N bài bị báo cáo chờ xem": bài có báo cáo OPEN hoặc tự ẩn chờ duyệt (chưa xoá).
select count(*)::int from (
  select r.target_id from post_reports r where r.course_id = sqlc.arg(course_id) and r.status = 'OPEN' group by r.target_id
  union
  select t.id from forum_threads t where t.course_id = sqlc.arg(course_id) and t.hidden_at is not null and t.hidden_by is null and t.deleted_at is null
  union
  select p.id from forum_posts p where p.course_id = sqlc.arg(course_id) and p.hidden_at is not null and p.hidden_by is null and p.deleted_at is null
) x;

-- name: ThreadLock :one
-- hidden-ok: khoá hàng cho đường Staff / chủ bài.
select id, course_id, author_id, title, body, tags, week_no, version, hidden_at, hidden_reason, hidden_by, pinned_at, deleted_at, ai_state
from forum_threads where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) for update;

-- name: PostLockMod :one
-- hidden-ok: khoá hàng cho đường Staff / chủ bài.
select p.id, p.course_id, p.thread_id, p.author_id, p.kind, p.body, p.version, p.hidden_at, p.hidden_by, p.deleted_at, p.verification_state, t.author_id as thread_author_id, t.deleted_at as thread_deleted_at
from forum_posts p join forum_threads t on t.course_id = p.course_id and t.id = p.thread_id
where p.course_id = sqlc.arg(course_id) and p.id = sqlc.arg(id) for update of p;

-- name: ThreadHide :exec
update forum_threads set hidden_at = now(), hidden_reason = sqlc.arg(reason), hidden_by = sqlc.narg(by), version = version + 1
where id = sqlc.arg(id) and course_id = sqlc.arg(course_id) and hidden_at is null;

-- name: ThreadUnhide :exec
update forum_threads set hidden_at = null, hidden_reason = null, hidden_by = null, version = version + 1
where id = sqlc.arg(id) and course_id = sqlc.arg(course_id) and hidden_at is not null;

-- name: PostHide :exec
update forum_posts set hidden_at = now(), hidden_reason = sqlc.arg(reason), hidden_by = sqlc.narg(by), version = version + 1
where id = sqlc.arg(id) and course_id = sqlc.arg(course_id) and hidden_at is null;

-- name: PostUnhide :exec
update forum_posts set hidden_at = null, hidden_reason = null, hidden_by = null, version = version + 1
where id = sqlc.arg(id) and course_id = sqlc.arg(course_id) and hidden_at is not null;

-- name: ThreadPinCount :one
-- hidden-ok: đếm hạn mức ghim, thread ẩn vẫn chiếm chỗ
select count(*)::int from forum_threads where course_id = sqlc.arg(course_id) and pinned_at is not null and deleted_at is null;

-- name: ThreadSetPinned :exec
update forum_threads set pinned_at = case when sqlc.arg(pinned)::bool then now() else null end
where id = sqlc.arg(id) and course_id = sqlc.arg(course_id);

-- name: ThreadEdit :exec
update forum_threads set title = sqlc.arg(title), body = sqlc.arg(body), tags = sqlc.arg(tags), week_no = sqlc.narg(week_no), embedding = sqlc.narg(embedding), similar_of = null,
       edited_at = now(), version = version + 1
where id = sqlc.arg(id) and course_id = sqlc.arg(course_id);

-- name: ThreadAIPostStates :many
-- hidden-ok: kiểm khoá sửa của chủ bài (trạng thái duyệt của bài AI, kể cả bài ẩn).
select id, verification_state from forum_posts where thread_id = sqlc.arg(thread_id) and course_id = sqlc.arg(course_id) and kind = 'AI' and deleted_at is null;

-- name: ThreadRejectPendingAI :many
-- Sửa thread khi bài AI còn PENDING: bài cũ REJECTED (tự động, không báo ai).
update forum_posts set verification_state = 'REJECTED', version = version + 1
where thread_id = sqlc.arg(thread_id) and course_id = sqlc.arg(course_id) and kind = 'AI' and verification_state = 'PENDING' and deleted_at is null
returning id;

-- name: ThreadResetAIState :exec
update forum_threads set ai_state = 'PENDING', ai_skip_reason = null where id = sqlc.arg(id) and course_id = sqlc.arg(course_id);

-- name: PostEdit :exec
update forum_posts set body = sqlc.arg(body), edited_at = now(), version = version + 1 where id = sqlc.arg(id) and course_id = sqlc.arg(course_id);

-- name: ThreadSoftDelete :exec
update forum_threads set deleted_at = now(), version = version + 1 where id = sqlc.arg(id) and course_id = sqlc.arg(course_id) and deleted_at is null;

-- name: PostSoftDelete :exec
update forum_posts set deleted_at = now(), version = version + 1 where id = sqlc.arg(id) and course_id = sqlc.arg(course_id) and deleted_at is null;

-- name: ModerationQueue :many
-- Hàng chờ của Staff. status OPEN: bài có báo cáo OPEN (kể cả bài vừa bị chủ bài xoá); HIDDEN: bài đang ẩn chưa xoá. Phân trang (qat, id): OPEN = báo cáo OPEN sớm nhất, HIDDEN = lúc ẩn.
-- hidden-ok: đường Staff.
select * from (
  select 'thread'::text as kind, t.id as id, t.id as thread_id, null::uuid as post_id, left(t.title, 140)::text as title_or_excerpt,
         t.hidden_at, (t.hidden_at is not null and t.hidden_by is null)::bool as hidden_by_auto, t.created_at,
         case when sqlc.arg(status)::text = 'OPEN' then (select min(r.created_at) from post_reports r where r.target_id = t.id and r.status = 'OPEN') else t.hidden_at end as qat
  from forum_threads t
  where t.course_id = sqlc.arg(course_id)
    and ((sqlc.arg(status)::text = 'OPEN' and exists (select 1 from post_reports r where r.target_id = t.id and r.status = 'OPEN'))
      or (sqlc.arg(status)::text = 'HIDDEN' and t.hidden_at is not null and t.deleted_at is null))
  union all
  select 'post'::text, p.id, p.thread_id, p.id, left(p.body, 140)::text,
         p.hidden_at, (p.hidden_at is not null and p.hidden_by is null)::bool, p.created_at,
         case when sqlc.arg(status)::text = 'OPEN' then (select min(r.created_at) from post_reports r where r.target_id = p.id and r.status = 'OPEN') else p.hidden_at end
  from forum_posts p
  where p.course_id = sqlc.arg(course_id)
    and ((sqlc.arg(status)::text = 'OPEN' and exists (select 1 from post_reports r where r.target_id = p.id and r.status = 'OPEN'))
      or (sqlc.arg(status)::text = 'HIDDEN' and p.hidden_at is not null and p.deleted_at is null))
) q
where (sqlc.narg(cursor_at)::timestamptz is null or (q.qat, q.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by q.qat desc, q.id desc
limit sqlc.arg(page_limit);

-- name: ModerationReportStats :many
-- Số báo cáo OPEN và theo lý do cho một nhóm bài (không trả người báo cáo).
select r.target_id, r.reason::text as reason, count(*)::int as n
from post_reports r where r.course_id = sqlc.arg(course_id) and r.status = 'OPEN' and r.target_id = any(sqlc.arg(ids)::text[]::uuid[])
group by r.target_id, r.reason;

-- name: ModerationStaffIDs :many
-- TA + TEACHER ACTIVE của lớp (người nhận thông báo gộp).
select user_id from enrollments where course_id = sqlc.arg(course_id) and status = 'ACTIVE' and role_in_course in ('TEACHER', 'TA');

-- name: ThreadKnowledgeSource :one
-- hidden-ok: nguồn tri thức của thread (bài AI duy nhất + câu hỏi); trạng thái ẩn / xoá trả kèm để người gọi quyết định nạp hay gỡ.
select p.id as post_id, p.verification_state::text as state, p.body as answer, (p.hidden_at is not null or p.deleted_at is not null or t.hidden_at is not null or t.deleted_at is not null)::bool as unavailable,
       t.title, t.body as question
from forum_posts p join forum_threads t on t.course_id = p.course_id and t.id = p.thread_id
where p.course_id = sqlc.arg(course_id) and p.thread_id = sqlc.arg(thread_id) and p.kind = 'AI';

-- name: ModerationAIPendingCount :one
-- "N câu trả lời AI chờ duyệt": bài AI PENDING chưa ẩn / chưa xoá của thread chưa ẩn / chưa xoá.
select count(*)::int from forum_posts p join forum_threads t on t.course_id = p.course_id and t.id = p.thread_id
where p.course_id = sqlc.arg(course_id) and p.kind = 'AI' and p.verification_state = 'PENDING' and p.hidden_at is null and p.deleted_at is null and t.hidden_at is null and t.deleted_at is null;

