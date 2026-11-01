-- US-P7-10 — phúc khảo bài tập (bảng riêng `grade_appeals`, không qua escalation_tickets; AI không tham gia).

-- name: GradeAppealContext :one
-- Kết quả ĐÃ CÔNG BỐ của sinh viên (bản nộp mới nhất, round cao nhất) + quy tắc phúc khảo của bài. Không có kết quả PUBLISHED → ErrNoRows.
select a.title, a.appeal_days, a.rubric, a.published_at as asg_published_at,
       s.id as submission_id, r.id as result_id, r.total, r.criteria, r.published_at,
       coalesce((select ap.id from grade_appeals ap where ap.course_id = a.course_id and ap.assignment_id = a.id and ap.student_id = sqlc.arg(student_id)::uuid), '00000000-0000-0000-0000-000000000000'::uuid)::uuid as appeal_id
from assignments a
join submissions s on s.course_id = a.course_id and s.assignment_id = a.id and s.student_id = sqlc.arg(student_id)::uuid and s.match_status = 'MATCHED'
join grading_results r on r.course_id = s.course_id and r.submission_id = s.id and r.status = 'PUBLISHED'
where a.course_id = sqlc.arg(course_id) and a.id = sqlc.arg(assignment_id) and a.published_at is not null
  and s.version = (select max(v.version) from submissions v where v.course_id = s.course_id and v.assignment_id = s.assignment_id and v.student_id = s.student_id and v.match_status = 'MATCHED')
order by r.round desc
limit 1;

-- name: GradeAppealInsert :one
insert into grade_appeals (course_id, assignment_id, student_id, submission_id, result_id, criteria_ids, reason, score_before)
values (sqlc.arg(course_id), sqlc.arg(assignment_id), sqlc.arg(student_id), sqlc.arg(submission_id), sqlc.arg(result_id), sqlc.arg(criteria_ids), sqlc.arg(reason), sqlc.arg(score_before))
returning id, created_at;

-- name: GradeAppealLock :one
-- Khoá phúc khảo + kết quả đã công bố gắn với nó + bài tập (rubric).
select ap.id, ap.assignment_id, ap.student_id, ap.submission_id, ap.result_id, ap.criteria_ids, ap.reason, ap.status, ap.score_before, ap.score_after, ap.response, ap.decided_by, ap.decided_at, ap.version, ap.created_at,
       r.total, r.criteria, r.late_penalty, r.late_penalty_manual, r.status as result_status, r.ai_total_before_late, r.version as result_version,
       a.title, a.rubric, a.max_score
from grade_appeals ap
join grading_results r on r.course_id = ap.course_id and r.id = ap.result_id
join assignments a on a.course_id = ap.course_id and a.id = ap.assignment_id
where ap.course_id = sqlc.arg(course_id) and ap.id = sqlc.arg(id)
for update of ap, r;

-- name: GradeAppealDecide :execrows
update grade_appeals set status = sqlc.arg(status), response = sqlc.arg(response), score_after = sqlc.narg(score_after), decided_by = sqlc.arg(decided_by), decided_at = now(), version = version + 1, updated_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'OPEN' and version = sqlc.arg(version);

-- name: GradeAppealAdjustResult :execrows
-- ADJUSTED: cập nhật kết quả đã công bố TẠI CHỖ (criteria, total, version + 1); published_at, ai_scores, trạng thái giữ nguyên.
update grading_results set criteria = sqlc.arg(criteria), total = sqlc.arg(total), version = version + 1, updated_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'PUBLISHED';

-- name: GradeAppealList :many
select ap.id, ap.assignment_id, a.title as assignment_title, ap.student_id, coalesce(u.full_name, '')::text as student_name, coalesce(e.student_code_snapshot, '')::text as student_code,
       ap.criteria_ids, ap.reason, ap.status, ap.created_at, ap.score_before, ap.score_after, ap.response, ap.decided_by, ap.decided_at, ap.version, ap.submission_id, ap.result_id,
       r.ai_total_before_late
from grade_appeals ap
join assignments a on a.course_id = ap.course_id and a.id = ap.assignment_id
join grading_results r on r.course_id = ap.course_id and r.id = ap.result_id
left join users u on u.id = ap.student_id
left join enrollments e on e.course_id = ap.course_id and e.user_id = ap.student_id
where ap.course_id = sqlc.arg(course_id)
  and (sqlc.narg(assignment_id)::uuid is null or ap.assignment_id = sqlc.narg(assignment_id)::uuid)
  and (sqlc.narg(status)::text is null or ap.status = sqlc.narg(status)::text)
  and (sqlc.narg(cursor_at)::timestamptz is null or (ap.created_at, ap.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by ap.created_at desc, ap.id desc
limit sqlc.arg(page_limit);

-- name: GradeAppealGet :one
select ap.id, ap.assignment_id, a.title as assignment_title, ap.student_id, coalesce(u.full_name, '')::text as student_name, coalesce(e.student_code_snapshot, '')::text as student_code,
       ap.criteria_ids, ap.reason, ap.status, ap.created_at, ap.score_before, ap.score_after, ap.response, ap.decided_by, ap.decided_at, ap.version, ap.submission_id, ap.result_id,
       r.ai_total_before_late, a.rubric
from grade_appeals ap
join assignments a on a.course_id = ap.course_id and a.id = ap.assignment_id
join grading_results r on r.course_id = ap.course_id and r.id = ap.result_id
left join users u on u.id = ap.student_id
left join enrollments e on e.course_id = ap.course_id and e.user_id = ap.student_id
where ap.course_id = sqlc.arg(course_id) and ap.id = sqlc.arg(id);

-- name: GradeAppealOpenCount :one
select count(*)::int from grade_appeals where course_id = sqlc.arg(course_id) and status = 'OPEN';

-- name: GradeAppealTeachers :many
-- TEACHER ACTIVE của lớp (chuông gộp "n yêu cầu xem lại mới"; TA không nhận).
select e.user_id from enrollments e where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'TEACHER' and e.status = 'ACTIVE';

-- name: GradeAppealIDByResult :one
select id from grade_appeals where course_id = sqlc.arg(course_id) and result_id = sqlc.arg(result_id);

-- name: SubmissionVersionCount :one
-- Số phiên bản đã KHỚP của sinh viên ở bài (hộp thoại gán tay: "Sinh viên này đã có n phiên bản").
select count(*)::int from submissions where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id) and student_id = sqlc.arg(student_id) and match_status = 'MATCHED';
