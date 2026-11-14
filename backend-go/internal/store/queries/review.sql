-- US-P7-08 — hàng chờ, chi tiết, sửa, duyệt kết quả chấm.

-- name: AssignmentQueue :many
-- Hàng chờ chấm / danh sách bài nộp MATCHED: chỉ PHIÊN BẢN MỚI NHẤT của từng sinh viên, kèm kết quả HIỆN HÀNH (round cao nhất). Khoá con trỏ (rank, received_at, id) với rank = 0 cho `needs_review`.
select s.id, s.assignment_id, a.title as assignment_title, s.student_id, coalesce(u.full_name, '')::text as student_name,
       coalesce(e.student_code_snapshot, '')::text as student_code, coalesce(e.status::text, '') as enrollment_status,
       s.source, s.version, s.received_at, s.submitted_at, a.due_at,
       r.id as result_id, r.status as result_status, r.run_state, r.total, r.flags, r.needs_review as result_needs_review, r.error_code, r.version as result_version,
       (case when coalesce(r.needs_review, false) then 0 else 1 end)::int as rank
from submissions s
join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
left join users u on u.id = s.student_id
left join enrollments e on e.course_id = s.course_id and e.user_id = s.student_id
left join grading_results r on r.course_id = s.course_id and r.submission_id = s.id
     and r.round = (select max(x.round) from grading_results x where x.course_id = s.course_id and x.submission_id = s.id)
where s.course_id = sqlc.arg(course_id) and s.match_status = 'MATCHED'
  and not exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version)
  and (sqlc.narg(assignment_id)::uuid is null or s.assignment_id = sqlc.narg(assignment_id)::uuid)
  and (sqlc.narg(status)::text is null
       or (sqlc.narg(status)::text = 'NONE' and r.id is null)
       or (sqlc.narg(status)::text = 'RUNNING' and r.run_state in ('QUEUED', 'RUNNING'))
       or (sqlc.narg(status)::text = 'DRAFT' and r.status = 'DRAFT' and r.run_state = 'DONE')
       or (sqlc.narg(status)::text = 'REVIEWED' and r.status = 'REVIEWED')
       or (sqlc.narg(status)::text = 'PUBLISHED' and r.status = 'PUBLISHED')
       or (sqlc.narg(status)::text = 'FAILED' and r.run_state = 'FAILED' and r.status = 'DRAFT'))
  and (sqlc.narg(needs_review)::bool is null or coalesce(r.needs_review, false) = sqlc.narg(needs_review)::bool)
  and (sqlc.narg(flag)::text is null or sqlc.narg(flag)::text = any(r.flags))
  and (sqlc.narg(q)::text is null or u.full_name ilike '%' || sqlc.narg(q)::text || '%' or e.student_code_snapshot ilike '%' || sqlc.narg(q)::text || '%')
  and (sqlc.narg(cursor_rank)::int is null
       or ((case when coalesce(r.needs_review, false) then 0 else 1 end), s.received_at, s.id) > (sqlc.narg(cursor_rank)::int, sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by rank, s.received_at, s.id
limit sqlc.arg(page_limit);

-- name: ReviewDetail :one
-- Chi tiết một kết quả (kèm bài nộp, bài tập, bản nộp có mới hơn không). `for update` ở biến thể ReviewLock.
select r.id, r.course_id, r.submission_id, r.round, r.run_state, r.status, r.error_code, r.criteria, r.overall_feedback, r.total, r.ai_scores, r.ai_total_before_late,
       r.late_days, r.late_penalty, r.late_penalty_manual, r.flags, r.needs_review, r.pass_detail, r.confidence, r.reviewed_by, r.reviewed_at, r.published_at, r.version,
       s.assignment_id, s.student_id, s.version as sub_version, s.file_name, s.source, s.extracted_text, s.received_at, s.submitted_at, s.match_status,
       a.title, a.rubric, a.max_score, a.reference_answer, a.due_at, a.allow_late, a.late_penalty_per_day, a.late_max_days,
       (exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version))::bool as superseded,
       (coalesce(s.extracted_text, '') <> '')::bool as has_text
from grading_results r
join submissions s on s.course_id = r.course_id and s.id = r.submission_id
join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
where r.course_id = sqlc.arg(course_id) and r.id = sqlc.arg(id);

-- name: ReviewLock :one
select r.id, r.course_id, r.submission_id, r.round, r.run_state, r.status, r.error_code, r.criteria, r.overall_feedback, r.total, r.ai_scores, r.ai_total_before_late,
       r.late_days, r.late_penalty, r.late_penalty_manual, r.flags, r.needs_review, r.pass_detail, r.confidence, r.reviewed_by, r.reviewed_at, r.published_at, r.version,
       s.assignment_id, s.student_id, s.version as sub_version, s.file_name, s.source, s.extracted_text, s.received_at, s.submitted_at, s.match_status,
       a.title, a.rubric, a.max_score, a.reference_answer, a.due_at, a.allow_late, a.late_penalty_per_day, a.late_max_days,
       (exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version))::bool as superseded,
       (coalesce(s.extracted_text, '') <> '')::bool as has_text
from grading_results r
join submissions s on s.course_id = r.course_id and s.id = r.submission_id
join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
where r.course_id = sqlc.arg(course_id) and r.id = sqlc.arg(id)
for update of r;

-- name: ReviewApplyEdit :execrows
-- Ghi bản sửa của nhân viên: `ai_scores` / `ai_total_before_late` không đụng; REVIEWED quay về DRAFT; FAILED nhập đủ điểm → DONE.
update grading_results set criteria = sqlc.arg(criteria), overall_feedback = sqlc.narg(overall_feedback), total = sqlc.arg(total),
    late_days = sqlc.arg(late_days), late_penalty = sqlc.arg(late_penalty), late_penalty_manual = sqlc.arg(late_penalty_manual),
    status = 'DRAFT', run_state = sqlc.arg(run_state), error_code = case when sqlc.arg(run_state)::text = 'DONE' then null else error_code end,
    reviewed_by = null, reviewed_at = null, pass_detail = sqlc.narg(pass_detail), version = version + 1, updated_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and version = sqlc.arg(version) and status in ('DRAFT', 'REVIEWED');

-- name: ReviewMark :execrows
update grading_results set status = 'REVIEWED', reviewed_by = sqlc.arg(reviewer), reviewed_at = now(), run_state = 'DONE', error_code = null, version = version + 1, updated_at = now(),
    first_reviewed_at = coalesce(first_reviewed_at, now()), first_reviewed_by = coalesce(first_reviewed_by, sqlc.arg(reviewer)) -- P-106-5: ghi một lần, không bị ReviewApplyEdit xoá
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and version = sqlc.arg(version) and status = 'DRAFT';

-- name: PublishReviewedIDs :many
-- scope ALL_REVIEWED: kết quả REVIEWED HIỆN HÀNH (round cao nhất) của bài, bản nộp mới nhất; tối đa limit.
select r.id from grading_results r
join submissions s on s.course_id = r.course_id and s.id = r.submission_id
where r.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and r.status = 'REVIEWED'
  and r.round = (select max(x.round) from grading_results x where x.course_id = r.course_id and x.submission_id = r.submission_id)
  and not exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version)
order by r.id
limit sqlc.arg(page_limit);

-- name: PublishLockMany :many
-- Khoá các kết quả sẽ công bố (thứ tự id: tránh deadlock giữa hai giảng viên) kèm dữ kiện để tính lại độ lệch / khoản trừ muộn lần cuối.
select r.id, r.status, r.run_state, r.version, r.criteria, r.ai_scores, r.ai_total_before_late, r.total, r.late_days, r.late_penalty, r.late_penalty_manual, r.guidance_version, r.first_reviewed_at,
       s.student_id, s.assignment_id, s.submitted_at,
       a.title, a.due_at, a.allow_late, a.late_penalty_per_day, a.late_max_days, a.max_score, a.guidance_version as assignment_guidance_version,
       (exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version))::bool as superseded
from grading_results r
join submissions s on s.course_id = r.course_id and s.id = r.submission_id
join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
where r.course_id = sqlc.arg(course_id) and r.id = any(sqlc.arg(ids)::text[]::uuid[])
order by r.id
for update of r;

-- name: PublishMark :execrows
-- NƠI DUY NHẤT đặt status = 'PUBLISHED' (TestPublishOnlyFromHandler quét mã).
update grading_results set status = 'PUBLISHED', published_by = sqlc.arg(publisher), published_at = now(), deviation = sqlc.arg(deviation),
    teacher_changed_criteria = sqlc.arg(changed), late_days = sqlc.arg(late_days), late_penalty = sqlc.arg(late_penalty), version = version + 1, updated_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'REVIEWED' and version = sqlc.arg(version);

-- name: AssignmentStats :one
-- Số liệu bài tập (nhân viên): kết quả HIỆN HÀNH của bản nộp mới nhất; độ lệch / tỷ lệ chỉ trên kết quả đã công bố; phúc khảo từ grade_appeals.
with cur as (
  select r.status, r.run_state, r.deviation, r.teacher_changed_criteria
  from submissions s
  join grading_results r on r.course_id = s.course_id and r.submission_id = s.id
  where s.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and s.match_status = 'MATCHED'
    and r.round = (select max(x.round) from grading_results x where x.course_id = r.course_id and x.submission_id = r.submission_id)
    and not exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version)),
subs as (
  select count(distinct s.student_id)::int as n from submissions s
  where s.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and s.match_status = 'MATCHED'),
ap as (
  select count(*)::int as appealed, count(*) filter (where status = 'ADJUSTED')::int as adjusted, count(*) filter (where status <> 'OPEN')::int as decided
  from grade_appeals where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id)::uuid)
select (select n from subs) as submitted,
       (select count(*) from cur where run_state = 'DONE')::int as graded,
       (select count(*) from cur where status in ('REVIEWED', 'PUBLISHED'))::int as reviewed,
       (select count(*) from cur where status = 'PUBLISHED')::int as published,
       coalesce((select avg(deviation) from cur where status = 'PUBLISHED'), 0)::numeric(5,2) as mean_abs_deviation,
       coalesce((select (count(*) filter (where cardinality(teacher_changed_criteria) > 0))::numeric / nullif(count(*), 0) from cur where status = 'PUBLISHED'), 0)::numeric(5,4) as teacher_changed_rate,
       (select appealed from ap) as appealed, (select adjusted from ap) as adjusted, (select decided from ap) as decided;

-- name: MinePublishedResult :one
-- Kết quả ĐÃ CÔNG BỐ của chính sinh viên (bản nộp mới nhất, round cao nhất). Không đọc cờ / độ tin cậy / điểm AI / lượt chấm / người duyệt; chỉ `status = 'PUBLISHED'`.
select r.id, r.total, r.criteria, r.overall_feedback, r.late_days, r.late_penalty, r.published_at
from submissions s
join grading_results r on r.course_id = s.course_id and r.submission_id = s.id
where s.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and s.student_id = sqlc.arg(student_id)::uuid and s.match_status = 'MATCHED'
  and r.status = 'PUBLISHED'
  and s.version = (select max(v.version) from submissions v where v.course_id = s.course_id and v.assignment_id = s.assignment_id and v.student_id = s.student_id and v.match_status = 'MATCHED')
order by r.round desc
limit 1;

-- name: MineAppealOf :one
select status, response from grade_appeals where course_id = sqlc.arg(course_id) and result_id = sqlc.arg(result_id);

-- name: SubmissionVersions :many
-- Chi tiết một bài nộp cho nhân viên (SRS 6 hàng 15): mọi PHIÊN BẢN MATCHED của cùng (bài tập, sinh viên), mới nhất trước, kèm kết quả chấm HIỆN HÀNH của từng bản. Bài chưa khớp / chưa có sinh viên: chỉ chính nó.
select s.id, s.assignment_id, a.title as assignment_title, s.student_id, coalesce(u.full_name, '')::text as student_name,
       coalesce(e.student_code_snapshot, '')::text as student_code, coalesce(e.status::text, '')::text as enrollment_status,
       s.source, s.version, s.file_name, s.file_size, s.match_status, s.unmatched_reason, s.received_at, s.submitted_at, a.due_at,
       (s.id = cur.id)::bool as requested,
       r.id as result_id, r.status as result_status, r.run_state, r.total, r.flags, r.needs_review as result_needs_review, r.error_code, r.version as result_version
from submissions cur
join submissions s on s.course_id = cur.course_id and (s.id = cur.id or (cur.student_id is not null and s.assignment_id = cur.assignment_id and s.student_id = cur.student_id and s.match_status = 'MATCHED'))
left join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
left join users u on u.id = s.student_id
left join enrollments e on e.course_id = s.course_id and e.user_id = s.student_id
left join grading_results r on r.course_id = s.course_id and r.submission_id = s.id
     and r.round = (select max(x.round) from grading_results x where x.course_id = s.course_id and x.submission_id = s.id)
where cur.course_id = sqlc.arg(course_id) and cur.id = sqlc.arg(id)
order by s.version desc nulls last, s.received_at desc, s.id desc;
