-- Bài code trong lượt làm (US-PE-06, SRS FEAT-weekly-exam 4.4, 5.10–5.11). Mọi truy vấn bắt đầu bằng course_id (luật 13).

-- name: CodeItemOfExam :one
-- Mục CODE của bài thi: `languages` để kiểm ngôn ngữ; không phải mục code → 0 dòng.
select i.id as item_id, i.question_id as problem_id, cp.languages
from exam_items i
join question_bank q on q.course_id = i.course_id and q.id = i.question_id and q.type = 'CODE'
join code_problems cp on cp.course_id = q.course_id and cp.question_id = q.id
where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id) and i.id = sqlc.arg(item_id);

-- name: CodeDraftGet :one
select language, source, rev, updated_at from code_drafts
where course_id = sqlc.arg(course_id) and attempt_id = sqlc.arg(attempt_id) and item_id = sqlc.arg(item_id) and language = sqlc.arg(language);

-- name: CodeDraftUpsert :one
-- Gọi sau khi đã kiểm `base_rev` (lượt đã bị khoá `for update`, nên không có hai bản ghi chen nhau). `rev` tăng 1 mỗi lần ghi.
insert into code_drafts (attempt_id, item_id, course_id, language, source, rev, updated_at)
values (sqlc.arg(attempt_id), sqlc.arg(item_id), sqlc.arg(course_id), sqlc.arg(language), sqlc.arg(source), 1, sqlc.arg(at))
on conflict (attempt_id, item_id, language) do update set source = excluded.source, rev = code_drafts.rev + 1, updated_at = excluded.updated_at
returning rev, updated_at;

-- name: CodeDraftsOfAttempt :many
-- Bản nháp của CHÍNH lượt này (trả lại khi mở / tải lại trang làm bài).
select item_id, language, source, rev, updated_at from code_drafts
where course_id = sqlc.arg(course_id) and attempt_id = sqlc.arg(attempt_id)
order by item_id, updated_at desc;

-- name: CodeDraftLatest :one
-- Bản nháp có `updated_at` lớn nhất của câu (một bản duy nhất — SRS 4.4.5, góp ý #2 (b)).
select language, source from code_drafts
where course_id = sqlc.arg(course_id) and attempt_id = sqlc.arg(attempt_id) and item_id = sqlc.arg(item_id)
order by updated_at desc, language
limit 1;

-- name: CodeSubmitCount :one
select count(*)::int from code_submissions
where course_id = sqlc.arg(course_id) and attempt_id = sqlc.arg(attempt_id) and item_id = sqlc.arg(item_id) and kind = 'SUBMIT';

-- name: CodeSubmissionInsert :one
insert into code_submissions (course_id, exam_id, attempt_id, item_id, problem_id, student_id, kind, language, source, source_sha256, auto)
values (sqlc.arg(course_id), sqlc.arg(exam_id), sqlc.arg(attempt_id), sqlc.arg(item_id), sqlc.arg(problem_id), sqlc.arg(student_id), sqlc.arg(kind), sqlc.arg(language), sqlc.arg(source),
        sqlc.arg(source_sha256), sqlc.arg(auto))
returning id, created_at;

-- name: CodeSubmissionGetOwn :one
-- Một bản nộp / chạy thử của CHÍNH sinh viên trong lượt này; người khác → 0 dòng → 404. `is_final`: bản SUBMIT mới nhất chưa bị thay.
select c.id, c.item_id, c.problem_id, c.kind, c.language, c.source, c.status, c.verdict, c.compile_ok, c.compile_log, c.results, c.auto, c.created_at,
       (c.kind = 'SUBMIT' and c.status <> 'SUPERSEDED' and not exists (
          select 1 from code_submissions n
          where n.course_id = c.course_id and n.attempt_id = c.attempt_id and n.item_id = c.item_id and n.kind = 'SUBMIT' and n.status <> 'SUPERSEDED'
            and (n.created_at, n.id) > (c.created_at, c.id)))::boolean as is_final
from code_submissions c
where c.course_id = sqlc.arg(course_id) and c.exam_id = sqlc.arg(exam_id) and c.attempt_id = sqlc.arg(attempt_id) and c.student_id = sqlc.arg(student_id) and c.id = sqlc.arg(id)
  and c.kind = sqlc.arg(kind);

-- name: CodeSubmissionsListOwn :many
-- Lịch sử nộp của (lượt, câu), mới nhất trước, cursor (created_at, id). Không lấy `source` (xem lại mã qua GET một bản).
select c.id, c.item_id, c.problem_id, c.kind, c.language, c.status, c.verdict, c.compile_ok, c.compile_log, c.results, c.auto, c.created_at,
       (c.status <> 'SUPERSEDED' and not exists (
          select 1 from code_submissions n
          where n.course_id = c.course_id and n.attempt_id = c.attempt_id and n.item_id = c.item_id and n.kind = 'SUBMIT' and n.status <> 'SUPERSEDED'
            and (n.created_at, n.id) > (c.created_at, c.id)))::boolean as is_final
from code_submissions c
where c.course_id = sqlc.arg(course_id) and c.attempt_id = sqlc.arg(attempt_id) and c.student_id = sqlc.arg(student_id) and c.item_id = sqlc.arg(item_id) and c.kind = 'SUBMIT'
  and (sqlc.narg(cursor_at)::timestamptz is null or (c.created_at, c.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by c.created_at desc, c.id desc
limit sqlc.arg(row_limit);

-- name: CodeSamplesOfProblem :many
-- Test MẪU đã duyệt (công khai theo định nghĩa); test lớn lưu ở kho (input NULL) không có ở đây.
select id, name, input, expected from code_testcases
where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id) and is_sample and approved and input is not null and expected is not null
order by position;

-- name: CodeSubmissionStudent :one
-- Chủ của bản nộp: worker dùng để phát SSE tới đúng người (không đưa danh tính vào payload outbox của máy chấm).
select student_id, kind, status, item_id, attempt_id from code_submissions where id = sqlc.arg(id);
