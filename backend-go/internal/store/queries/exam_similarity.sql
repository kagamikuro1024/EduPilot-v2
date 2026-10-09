-- So độ giống mã nguồn giữa các bản nộp cuối (US-PE-07 AC7–AC9, SRS FEAT-weekly-exam 4.7, 5.14). Mã không rời hệ thống.

-- name: SimilarityFinalSubmissions :many
-- Bản SUBMIT cuối (mới nhất chưa bị thay, kể cả bản tự động) của mỗi (lượt, câu).
select distinct on (s.attempt_id, s.item_id) s.id, s.attempt_id, s.item_id, s.problem_id, s.student_id, s.language, s.source
from code_submissions s
where s.course_id = sqlc.arg(course_id) and s.exam_id = sqlc.arg(exam_id) and s.kind = 'SUBMIT' and s.status <> 'SUPERSEDED'
order by s.attempt_id, s.item_id, s.created_at desc, s.id desc;

-- name: SimilarityStarters :many
select question_id, starter_code from code_problems where course_id = sqlc.arg(course_id) and question_id = any(sqlc.arg(ids)::text[]::uuid[]);

-- name: SimilarityInsert :exec
-- Một lệnh cho cả lô (≤ 200 cặp). `score` gửi dạng ‰ (số nguyên) rồi chia 1000: không đi qua số thực.
insert into similarity_reports (course_id, exam_id, problem_id, run_id, submission_a, submission_b, attempt_a, attempt_b, score, shared_fingerprints, flagged)
select sqlc.arg(course_id), sqlc.arg(exam_id), x.problem_id, sqlc.arg(run_id), x.sa, x.sb, x.aa, x.ab, (x.score::numeric / 1000)::numeric(4,3), x.shared, x.flagged
from jsonb_to_recordset(sqlc.arg(rows)::jsonb) as x(problem_id uuid, sa uuid, sb uuid, aa uuid, ab uuid, score int, shared int, flagged boolean);

-- name: SimilarityLatestRun :one
select run_id from similarity_reports where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) order by created_at desc, id desc limit 1;

-- name: SimilarityList :many
-- Điểm cao trước (cursor theo (score, id)); mặc định bản chạy mới nhất; `flagged_only` lọc cặp gắn cờ.
select r.id, r.problem_id, qb.title as problem_title, r.run_id, r.attempt_a, r.attempt_b, ua.full_name as name_a, ub.full_name as name_b,
       r.score, r.shared_fingerprints, r.flagged, r.review_state, r.note, r.reviewed_at, r.created_at
from similarity_reports r
join question_bank qb on qb.course_id = r.course_id and qb.id = r.problem_id
join exam_attempts aa on aa.course_id = r.course_id and aa.id = r.attempt_a
join exam_attempts ab on ab.course_id = r.course_id and ab.id = r.attempt_b
join users ua on ua.id = aa.student_id
join users ub on ub.id = ab.student_id
where r.course_id = sqlc.arg(course_id) and r.exam_id = sqlc.arg(exam_id) and r.run_id = sqlc.arg(run_id)
  and (not sqlc.arg(flagged_only)::boolean or r.flagged)
  and (sqlc.narg(cursor_score)::numeric is null or (r.score, r.id) < (sqlc.narg(cursor_score)::numeric, sqlc.narg(cursor_id)::uuid))
order by r.score desc, r.id desc
limit sqlc.arg(row_limit);

-- name: SimilarityGet :one
select r.id, r.problem_id, qb.title as problem_title, r.run_id, r.attempt_a, r.attempt_b, r.submission_a, r.submission_b, ua.full_name as name_a, ub.full_name as name_b,
       r.score, r.shared_fingerprints, r.flagged, r.review_state, r.note, r.reviewed_at, r.created_at
from similarity_reports r
join question_bank qb on qb.course_id = r.course_id and qb.id = r.problem_id
join exam_attempts aa on aa.course_id = r.course_id and aa.id = r.attempt_a
join exam_attempts ab on ab.course_id = r.course_id and ab.id = r.attempt_b
join users ua on ua.id = aa.student_id
join users ub on ub.id = ab.student_id
where r.course_id = sqlc.arg(course_id) and r.exam_id = sqlc.arg(exam_id) and r.id = sqlc.arg(id);

-- name: SimilaritySource :one
select language, source from code_submissions where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: SimilarityReview :one
update similarity_reports set review_state = sqlc.arg(state), note = sqlc.narg(note), reviewed_by = sqlc.arg(reviewer), reviewed_at = sqlc.arg(at)
where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) and id = sqlc.arg(id)
returning id;

-- name: SimilarityDueExams :many
-- Bài code đã đóng, mọi lượt đã nộp, chưa từng xếp việc so độ giống (dấu vết ở audit_log): bộ lập lịch xếp việc `exam.similarity`.
select e.course_id, e.id, e.created_by from exams e
where e.status in ('CLOSED', 'PUBLISHED') and e.kind in ('CODE', 'MIXED')
  and not exists (select 1 from exam_attempts a where a.course_id = e.course_id and a.exam_id = e.id and a.status = 'IN_PROGRESS')
  and not exists (select 1 from audit_log l where l.course_id = e.course_id and l.entity = 'exam' and l.entity_id = e.id::text and l.action in ('exam.similarity.queued', 'exam.similarity.run'))
order by e.closes_at
limit 20;
