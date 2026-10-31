-- name: AssignmentInsert :one
insert into assignments (course_id, title, description, type, max_score, due_at, allow_late, late_penalty_per_day, late_max_days, appeal_days, channels, submission_code,
                         reference_answer, rubric, teams_class_id, teams_assignment_id, created_by)
values (sqlc.arg(course_id), sqlc.arg(title), sqlc.arg(description), sqlc.arg(type), sqlc.arg(max_score), sqlc.arg(due_at), sqlc.arg(allow_late), sqlc.arg(late_penalty_per_day),
        sqlc.arg(late_max_days), sqlc.arg(appeal_days), sqlc.arg(channels), sqlc.arg(submission_code), sqlc.narg(reference_answer), sqlc.arg(rubric),
        sqlc.narg(teams_class_id), sqlc.narg(teams_assignment_id), sqlc.arg(created_by))
returning *;

-- name: AssignmentGet :one
select * from assignments where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: AssignmentGetForUpdate :one
select * from assignments where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) for update;

-- name: AssignmentList :many
-- Danh sách theo (created_at giảm dần, id); sinh viên chỉ thấy bài đã công bố. Con trỏ keyset trên (created_at, id).
select * from assignments a
where a.course_id = sqlc.arg(course_id)
  and (not sqlc.arg(published_only)::bool or a.published_at is not null)
  and (sqlc.narg(cursor_at)::timestamptz is null or (a.created_at, a.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by a.created_at desc, a.id desc
limit sqlc.arg(page_limit);

-- name: AssignmentCounts :many
-- Đã nộp (sinh viên có bài MATCHED) / Tổng sinh viên ACTIVE / Đã chấm (kết quả hiện hành DONE) / Đã công bố, theo lô cho các bài của trang.
select a.id as assignment_id,
       (select count(distinct s.student_id) from submissions s where s.course_id = a.course_id and s.assignment_id = a.id and s.match_status = 'MATCHED')::int as submitted,
       (select count(*) from enrollments e where e.course_id = a.course_id and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE')::int as total,
       (select count(*) from submissions s join lateral (select r.status, r.run_state from grading_results r where r.course_id = s.course_id and r.submission_id = s.id order by r.round desc limit 1) r on true
         where s.course_id = a.course_id and s.assignment_id = a.id and s.match_status = 'MATCHED' and r.run_state = 'DONE'
           and s.version = (select max(s2.version) from submissions s2 where s2.course_id = s.course_id and s2.assignment_id = s.assignment_id and s2.student_id = s.student_id and s2.match_status = 'MATCHED'))::int as graded,
       (select count(*) from submissions s join lateral (select r.status from grading_results r where r.course_id = s.course_id and r.submission_id = s.id order by r.round desc limit 1) r on true
         where s.course_id = a.course_id and s.assignment_id = a.id and s.match_status = 'MATCHED' and r.status = 'PUBLISHED'
           and s.version = (select max(s2.version) from submissions s2 where s2.course_id = s.course_id and s2.assignment_id = s.assignment_id and s2.student_id = s.student_id and s2.match_status = 'MATCHED'))::int as published
from assignments a
where a.course_id = sqlc.arg(course_id) and a.id::text = any(sqlc.arg(ids)::text[]);

-- name: AssignmentUpdate :one
update assignments set title = sqlc.arg(title), description = sqlc.arg(description), max_score = sqlc.arg(max_score), due_at = sqlc.arg(due_at), allow_late = sqlc.arg(allow_late),
       late_penalty_per_day = sqlc.arg(late_penalty_per_day), late_max_days = sqlc.arg(late_max_days), appeal_days = sqlc.arg(appeal_days), channels = sqlc.arg(channels),
       reference_answer = sqlc.narg(reference_answer), teams_class_id = sqlc.narg(teams_class_id), teams_assignment_id = sqlc.narg(teams_assignment_id), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and version = sqlc.arg(version)
returning *;

-- name: AssignmentSetRubric :one
update assignments set rubric = sqlc.arg(rubric), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and version = sqlc.arg(version)
returning *;

-- name: AssignmentPublish :one
update assignments set published_at = now(), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and published_at is null
returning *;

-- name: AssignmentDelete :execrows
delete from assignments where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: AssignmentLockFacts :one
-- Khoá sửa: có kết quả chấm nào / có kết quả PUBLISHED nào / có bài nộp nào / có item sổ điểm trỏ tới.
select exists (select 1 from grading_results r join submissions s on s.course_id = r.course_id and s.id = r.submission_id where s.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(id)::uuid)::bool as has_results,
       exists (select 1 from grading_results r join submissions s on s.course_id = r.course_id and s.id = r.submission_id where s.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(id)::uuid and r.status = 'PUBLISHED')::bool as has_published,
       exists (select 1 from submissions s where s.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(id)::uuid)::bool as has_submissions,
       exists (select 1 from grade_items i where i.course_id = sqlc.arg(course_id) and i.assignment_id = sqlc.arg(id)::uuid)::bool as linked;

-- name: AssignmentGradeLink :one
-- "Tính vào điểm: {thành phần} · {key}": công thức CONFIRMED trước, không có thì DRAFT.
select c.name as component_name, c.key as component_key
from grade_items i
join grade_schemes sch on sch.course_id = i.course_id and sch.id = i.scheme_id
join grade_components c on c.course_id = i.course_id and c.id = i.component_id
where i.course_id = sqlc.arg(course_id) and i.assignment_id = sqlc.arg(id)::uuid and sch.status in ('CONFIRMED', 'DRAFT')
order by (sch.status = 'CONFIRMED') desc
limit 1;

-- name: AssignmentActiveStudents :many
select user_id from enrollments where course_id = sqlc.arg(course_id) and role_in_course = 'STUDENT' and status = 'ACTIVE' order by user_id;

-- name: AssignmentRecomputeCandidates :many
-- Kết quả DRAFT / REVIEWED của bài (đổi hạn → tính lại khoản trừ muộn; khoản đã sửa tay giữ nguyên).
select r.id, r.status, r.run_state, r.total, r.ai_total_before_late, r.criteria, r.late_days, r.late_penalty, r.late_penalty_manual, s.submitted_at
from grading_results r join submissions s on s.course_id = r.course_id and s.id = r.submission_id
where s.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and r.status in ('DRAFT', 'REVIEWED') and r.run_state = 'DONE'
for update of r;

-- name: AssignmentApplyLate :exec
update grading_results set late_days = sqlc.arg(late_days), late_penalty = sqlc.arg(late_penalty), total = sqlc.arg(total), status = sqlc.arg(status),
       reviewed_by = case when sqlc.arg(status)::text = 'DRAFT' then null else reviewed_by end, reviewed_at = case when sqlc.arg(status)::text = 'DRAFT' then null else reviewed_at end,
       version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);
