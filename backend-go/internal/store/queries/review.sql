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
update grading_results set status = 'REVIEWED', reviewed_by = sqlc.arg(reviewer), reviewed_at = now(), run_state = 'DONE', error_code = null, version = version + 1, updated_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and version = sqlc.arg(version) and status = 'DRAFT';
