-- US-P7-03 — bài nộp (InAppSource) và đọc phía sinh viên. Mọi truy vấn bắt đầu từ course_id (luật 13).

-- name: SubmissionLockStudent :exec
-- Khoá cấp `version` theo (bài, sinh viên): mọi nguồn (IN_APP, UPLOAD, IMAP, TEAMS) đi qua cùng một khoá (SRS 4.3.2, mẫu TicketLockStudentCap).
select pg_advisory_xact_lock(hashtextextended('submission:' || sqlc.arg(assignment_id)::text || ':' || sqlc.arg(student_id)::text, 0));

-- name: SubmissionLatestMatched :one
-- Bản MATCHED mới nhất của (bài, sinh viên) kèm trạng thái kết quả hiện hành (vòng chấm cao nhất) — null nếu chưa có kết quả.
select s.id, s.version, s.file_name, s.received_at, s.submitted_at, coalesce(r.status, '')::text as result_status, coalesce(r.run_state, '')::text as result_run_state
from submissions s
left join lateral (select gr.status, gr.run_state from grading_results gr where gr.course_id = s.course_id and gr.submission_id = s.id order by gr.round desc limit 1) r on true
where s.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and s.student_id = sqlc.arg(student_id)::uuid and s.match_status = 'MATCHED'
order by s.version desc
limit 1;

-- name: SubmissionInsertInApp :one
insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at, received_at)
values (sqlc.arg(course_id), sqlc.arg(assignment_id), sqlc.arg(student_id), 'IN_APP',
        (select coalesce(max(v.version), 0) + 1 from submissions v where v.course_id = sqlc.arg(course_id) and v.assignment_id = sqlc.arg(assignment_id)::uuid and v.student_id = sqlc.arg(student_id)::uuid),
        sqlc.arg(blob_key), sqlc.arg(file_name), sqlc.arg(file_size), sqlc.arg(file_mime), sqlc.arg(sha256), 'MATCHED', 'IN_APP', sqlc.arg(at)::timestamptz, sqlc.arg(at)::timestamptz)
returning id, version, received_at, submitted_at;

-- name: SubmissionMineBatch :many
-- Bản MATCHED mới nhất của CHÍNH sinh viên cho các bài của một trang danh sách (một truy vấn, không N+1); bài chưa nộp không có dòng.
select distinct on (s.assignment_id) s.assignment_id, s.id as submission_id, s.version, s.file_name, s.received_at, s.submitted_at, coalesce(r.status, '')::text as result_status
from submissions s
left join lateral (select gr.status from grading_results gr where gr.course_id = s.course_id and gr.submission_id = s.id order by gr.round desc limit 1) r on true
where s.course_id = sqlc.arg(course_id) and s.student_id = sqlc.arg(student_id)::uuid and s.match_status = 'MATCHED' and s.assignment_id::text = any(sqlc.arg(ids)::text[])
order by s.assignment_id, s.version desc;

-- name: SubmissionBlobKeysUsed :many
-- Trong danh sách khoá, những khoá đã có bài nộp trỏ tới (job dọn tệp mồ côi giữ lại).
select s.blob_key from submissions s where s.blob_key = any(sqlc.arg(keys)::text[]);

-- name: TodayStudentAssignments :many
-- "Hôm nay" của sinh viên (ASSIGNMENT_DUE, bậc 41): bài đã công bố, CHƯA nộp (không có bài MATCHED của mình), hạn trong `horizon` hoặc đã quá hạn nhưng còn cửa muộn — MỘT truy vấn cho mọi lớp.
select a.id, a.course_id, a.title, a.due_at, a.allow_late, a.late_max_days
from assignments a
where a.course_id = any(sqlc.arg(course_ids)::text[]::uuid[])
  and a.published_at is not null
  and 'IN_APP' = any(a.channels)
  and a.due_at <= sqlc.arg(horizon)::timestamptz
  and (a.due_at > sqlc.arg(now)::timestamptz or (a.allow_late and a.due_at + make_interval(days => a.late_max_days::int) > sqlc.arg(now)::timestamptz))
  and not exists (select 1 from submissions s where s.course_id = a.course_id and s.assignment_id = a.id and s.student_id = sqlc.arg(user_id)::uuid and s.match_status = 'MATCHED')
order by a.due_at, a.id;

-- name: SubmissionRoster :many
-- Sinh viên ACTIVE của lớp cùng các tín hiệu để khớp: email đã xác minh, MSSV theo `student_code_snapshot` (chỉ dòng ROSTER / ADMIN là tín hiệu; dòng CODE là MSSV tự khai — TLR-1). KHÔNG đọc `users.student_code`.
select e.user_id, lower(u.email)::text as email, (u.email_verified_at is not null)::bool as email_verified, coalesce(e.student_code_snapshot, '')::text as code, e.joined_via::text as joined_via
from enrollments e join users u on u.id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE' and u.status = 'ACTIVE';

-- name: SubmissionAssignmentRefs :many
-- Bài đã công bố của lớp có kênh `channel` bật: mã nộp thư để bộ khớp tìm trong tiêu đề / tên tệp.
select a.id, a.submission_code
from assignments a
where a.course_id = sqlc.arg(course_id) and a.published_at is not null and sqlc.arg(channel)::text = any(a.channels);

-- name: SubmissionIngest :one
-- Ghi một bài nộp từ nguồn ngoài. Trùng (course, source, external_ref, external_ts) → không ghi (sqlc trả no rows = `deduped`). `version` chỉ cấp khi MATCHED (dưới khoá `SubmissionLockStudent`).
insert into submissions (course_id, assignment_id, student_id, candidate_student_id, source, version, blob_key, file_name, file_size, file_mime, sha256,
                         external_ref, external_ts, from_email, subject, sender_verified, match_status, match_method, unmatched_reason, submitted_at, received_at)
values (sqlc.arg(course_id), sqlc.narg(assignment_id), sqlc.narg(student_id), sqlc.narg(candidate_student_id), sqlc.arg(source),
        case when sqlc.narg(student_id)::uuid is not null then (select coalesce(max(v.version), 0) + 1 from submissions v where v.course_id = sqlc.arg(course_id) and v.assignment_id = sqlc.narg(assignment_id)::uuid and v.student_id = sqlc.narg(student_id)::uuid) end,
        sqlc.arg(blob_key), sqlc.arg(file_name), sqlc.arg(file_size), sqlc.arg(file_mime), sqlc.arg(sha256),
        sqlc.narg(external_ref), sqlc.narg(external_ts), sqlc.narg(from_email), sqlc.narg(subject), sqlc.arg(sender_verified), sqlc.arg(match_status), sqlc.narg(match_method), sqlc.narg(unmatched_reason),
        sqlc.narg(submitted_at), sqlc.arg(received_at))
on conflict (course_id, source, external_ref, external_ts) where external_ref is not null do nothing
returning id, version;

-- name: SubmissionUnmatchedList :many
-- Tab `Chưa khớp`: bài UNMATCHED chưa `Bỏ qua`, mới nhất trước; keyset (received_at, id). `version` = updated_at theo micro-giây (khoá lạc quan của dòng; bảng không có cột version cho bài chưa khớp).
select s.id, s.file_name, s.source, s.received_at, s.unmatched_reason, s.assignment_id, s.candidate_student_id,
       coalesce(u.full_name, '')::text as candidate_name, (extract(epoch from s.updated_at) * 1000000)::bigint as row_version
from submissions s
left join users u on u.id = s.candidate_student_id
where s.course_id = sqlc.arg(course_id) and s.match_status = 'UNMATCHED' and s.dismissed_at is null
  and (sqlc.narg(cursor_at)::timestamptz is null or (s.received_at, s.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by s.received_at desc, s.id desc
limit sqlc.arg(page_limit);

-- name: SubmissionGetForUpdate :one
select id, course_id, assignment_id, student_id, match_status, dismissed_at, blob_key, file_name, (extract(epoch from updated_at) * 1000000)::bigint as row_version
from submissions where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) for update;

-- name: SubmissionFileKey :one
select blob_key, file_name from submissions where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: SubmissionAssignmentHasPublished :one
-- Bài đã có kết quả chấm PUBLISHED (TA không gán / bỏ qua nữa — Q5).
select exists (select 1 from grading_results r join submissions s on s.course_id = r.course_id and s.id = r.submission_id
               where s.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and r.status = 'PUBLISHED')::bool;

-- name: SubmissionStudentActive :one
select exists (select 1 from enrollments e where e.course_id = sqlc.arg(course_id) and e.user_id = sqlc.arg(user_id)::uuid and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE')::bool;

-- name: SubmissionMatchManual :one
-- Gán tay: version = max + 1 của (bài, sinh viên) — gọi SAU khi giữ `SubmissionLockStudent`.
update submissions set assignment_id = sqlc.arg(assignment_id), student_id = sqlc.arg(student_id), match_method = 'MANUAL', match_status = 'MATCHED', unmatched_reason = null, candidate_student_id = null,
       version = (select coalesce(max(v.version), 0) + 1 from submissions v where v.course_id = sqlc.arg(course_id) and v.assignment_id = sqlc.arg(assignment_id)::uuid and v.student_id = sqlc.arg(student_id)::uuid),
       updated_at = now()
where submissions.course_id = sqlc.arg(course_id) and submissions.id = sqlc.arg(id) and submissions.match_status = 'UNMATCHED'
returning id, version;

-- name: SubmissionDismiss :execrows
update submissions set dismissed_at = now(), dismissed_by = sqlc.arg(actor), dismissed_reason = sqlc.arg(reason), updated_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and match_status = 'UNMATCHED' and dismissed_at is null;

-- name: SubmissionUndismiss :execrows
update submissions set dismissed_at = null, dismissed_by = null, dismissed_reason = null, updated_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and dismissed_at is not null;

-- name: TodayStaffAssignmentBatch :many
-- "Hôm nay" của nhân viên: UNMATCHED_SUBMISSION (60), GRADING_ERROR (28), GRADING_REVIEW (30) — MỘT truy vấn tổng hợp cho mọi lớp (SRS 4.12). Chỉ kết quả HIỆN HÀNH của bài nộp MATCHED mới nhất.
with ids as (select unnest(sqlc.arg(course_ids)::text[])::uuid as course_id),
unm as (
  select s.course_id, count(*)::int as n from submissions s
  where s.course_id in (select course_id from ids) and s.match_status = 'UNMATCHED' and s.dismissed_at is null
  group by s.course_id),
cur as (
  select r.course_id,
         count(*) filter (where r.run_state = 'FAILED')::int as err_n,
         count(*) filter (where r.run_state = 'DONE' and r.needs_review)::int as rev_n,
         coalesce(min(a.title) filter (where r.run_state = 'FAILED'), '')::text as err_title,
         coalesce(min(a.title) filter (where r.run_state = 'DONE' and r.needs_review), '')::text as rev_title
  from grading_results r
  join submissions s on s.course_id = r.course_id and s.id = r.submission_id
  join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
  where r.course_id in (select course_id from ids) and r.status = 'DRAFT' and r.run_state in ('FAILED', 'DONE') and s.match_status = 'MATCHED'
    and r.round = (select max(x.round) from grading_results x where x.course_id = r.course_id and x.submission_id = r.submission_id)
    and not exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version)
  group by r.course_id)
select ids.course_id, coalesce(unm.n, 0)::int as unmatched, coalesce(cur.err_n, 0)::int as err_n, coalesce(cur.rev_n, 0)::int as rev_n,
       coalesce(cur.err_title, '')::text as err_title, coalesce(cur.rev_title, '')::text as rev_title
from ids left join unm on unm.course_id = ids.course_id left join cur on cur.course_id = ids.course_id;

-- name: GradeInputAssignments :many
-- Điểm item ASSIGNMENT (đọc thẳng như EXAM, TLR-14 A): `grading_results.total` PUBLISHED của BẢN NỘP MỚI NHẤT của (bài, sinh viên), theo lô sinh viên; kèm có bài MATCHED hay không và thang của bài / item.
select i.id as item_id, st.student_id::uuid as student_id, i.max_score as item_max, a.max_score as asg_max,
       (select r.total from submissions s join grading_results r on r.course_id = s.course_id and r.submission_id = s.id and r.status = 'PUBLISHED'
         where s.course_id = i.course_id and s.assignment_id = i.assignment_id and s.student_id = st.student_id::uuid and s.match_status = 'MATCHED'
           and s.version = (select max(v.version) from submissions v where v.course_id = s.course_id and v.assignment_id = s.assignment_id and v.student_id = s.student_id and v.match_status = 'MATCHED')
         order by r.round desc limit 1) as score,
       exists (select 1 from submissions s where s.course_id = i.course_id and s.assignment_id = i.assignment_id and s.student_id = st.student_id::uuid and s.match_status = 'MATCHED')::bool as has_matched
from grade_items i
join assignments a on a.course_id = i.course_id and a.id = i.assignment_id
cross join unnest(sqlc.arg(student_ids)::text[]) as st(student_id)
where i.course_id = sqlc.arg(course_id) and i.scheme_id = sqlc.arg(scheme_id) and i.source = 'ASSIGNMENT';

-- name: GradeAssignmentFacts :many
-- Dữ kiện theo item ASSIGNMENT: còn bao nhiêu bài UNMATCHED chưa bỏ qua của bài đó, và cửa nộp đã đóng chưa (due_at, cộng late_max_days nếu cho nộp muộn).
select i.id as item_id,
       (select count(*) from submissions s where s.course_id = i.course_id and s.assignment_id = i.assignment_id and s.match_status = 'UNMATCHED' and s.dismissed_at is null)::int as unmatched,
       (a.due_at + case when a.allow_late then make_interval(days => a.late_max_days::int) else interval '0' end < now())::bool as closed
from grade_items i join assignments a on a.course_id = i.course_id and a.id = i.assignment_id
where i.course_id = sqlc.arg(course_id) and i.scheme_id = sqlc.arg(scheme_id) and i.source = 'ASSIGNMENT';
