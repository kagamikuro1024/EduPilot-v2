-- Ngân hàng câu hỏi (US-PE-03, SRS 5.2–5.5). Mọi truy vấn đều bắt đầu bằng course_id (luật 13).

-- name: QuestionCourseStatus :one
select status from courses where id = sqlc.arg(course_id);

-- name: QuestionInsert :one
insert into question_bank (course_id, type, title, topic, difficulty, stem, answer_key, explanation, origin, review_status, created_by, ai_job_id)
values (sqlc.arg(course_id), sqlc.arg(type), sqlc.arg(title), sqlc.arg(topic), sqlc.arg(difficulty), sqlc.arg(stem), sqlc.narg(answer_key), sqlc.narg(explanation),
        sqlc.arg(origin), sqlc.arg(review_status), sqlc.arg(created_by), sqlc.narg(ai_job_id))
returning *;

-- name: QuestionOptionInsert :one
insert into question_options (id, course_id, question_id, position, body, pinned_last)
values (sqlc.arg(id), sqlc.arg(course_id), sqlc.arg(question_id), sqlc.arg(position), sqlc.arg(body), sqlc.arg(pinned_last))
returning id;

-- name: QuestionGet :one
select * from question_bank where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: QuestionLock :one
select * from question_bank where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) for update;

-- name: QuestionOptions :many
select id, position, body, pinned_last from question_options
where course_id = sqlc.arg(course_id) and question_id = sqlc.arg(question_id) order by position;

-- name: QuestionOptionsDelete :exec
delete from question_options where course_id = sqlc.arg(course_id) and question_id = sqlc.arg(question_id);

-- name: QuestionList :many
-- Danh sách có lọc, phân trang con trỏ (created_at desc, id desc). `used_in_exams` đếm bằng truy vấn con (không N+1).
select q.id, q.type, q.title, q.topic, q.difficulty, q.review_status, q.origin, q.version, q.updated_at, q.created_at, q.archived_at,
       (select count(distinct i.exam_id) from exam_items i where i.course_id = q.course_id and i.question_id = q.id)::int as used_in_exams
from question_bank q
where q.course_id = sqlc.arg(course_id)
  and (sqlc.arg(archived)::boolean or q.archived_at is null)
  and (sqlc.narg(review_status)::question_review_status is null or q.review_status = sqlc.narg(review_status))
  and (sqlc.narg(topic)::text is null or q.topic = sqlc.narg(topic))
  and (sqlc.narg(difficulty)::question_difficulty is null or q.difficulty = sqlc.narg(difficulty))
  and (sqlc.narg(type)::question_type is null or q.type = sqlc.narg(type))
  and (sqlc.narg(origin)::question_origin is null or q.origin = sqlc.narg(origin))
  and (sqlc.narg(q)::text is null or q.title ilike '%' || sqlc.narg(q) || '%')
  and (sqlc.narg(cursor_at)::timestamptz is null or (q.created_at, q.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by q.created_at desc, q.id desc
limit sqlc.arg(max_rows);

-- name: QuestionUpdateContent :one
-- Sửa nội dung: tăng version, về DRAFT (gỡ người duyệt) nếu `reset_review`; `expected_version` là khoá lạc quan (0 dòng = sai version).
update question_bank
set title = sqlc.arg(title), topic = sqlc.arg(topic), difficulty = sqlc.arg(difficulty), stem = sqlc.arg(stem),
    answer_key = sqlc.narg(answer_key), explanation = sqlc.narg(explanation), version = version + 1, updated_at = now(),
    review_status = case when sqlc.arg(reset_review)::boolean and review_status in ('APPROVED', 'REJECTED') then 'DRAFT'::question_review_status else review_status end,
    reviewed_by = case when sqlc.arg(reset_review)::boolean and review_status in ('APPROVED', 'REJECTED') then null else reviewed_by end,
    reviewed_at = case when sqlc.arg(reset_review)::boolean and review_status in ('APPROVED', 'REJECTED') then null else reviewed_at end
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and version = sqlc.arg(expected_version)
returning *;

-- name: QuestionBump :one
-- Đổi cấu hình bài code / test: tăng version, về DRAFT.
update question_bank
set version = version + 1, updated_at = now(),
    review_status = case when review_status in ('APPROVED', 'REJECTED') then 'DRAFT'::question_review_status else review_status end,
    reviewed_by = case when review_status in ('APPROVED', 'REJECTED') then null else reviewed_by end,
    reviewed_at = case when review_status in ('APPROVED', 'REJECTED') then null else reviewed_at end
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id)
returning *;

-- name: QuestionArchive :one
update question_bank set archived_at = coalesce(archived_at, now()), version = version + 1, updated_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id)
returning *;

-- name: QuestionReview :one
update question_bank
set review_status = sqlc.arg(review_status), reviewed_by = sqlc.narg(reviewed_by), reviewed_at = sqlc.narg(reviewed_at), version = version + 1, updated_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and version = sqlc.arg(expected_version)
returning *;

-- name: QuestionUsage :many
-- Bài thi đang dùng câu này, kèm trạng thái (để khoá sửa / tính "đang dùng").
select distinct e.id, e.title, e.status from exam_items i
join exams e on e.course_id = i.course_id and e.id = i.exam_id
where i.course_id = sqlc.arg(course_id) and i.question_id = sqlc.arg(question_id)
order by e.title, e.id;

-- name: CodeProblemInsert :one
insert into code_problems (question_id, course_id) values (sqlc.arg(question_id), sqlc.arg(course_id)) returning *;

-- name: CodeProblemInsertFull :one
insert into code_problems (question_id, course_id, languages, time_limit_ms, memory_limit_mb, output_limit_kb, checker, float_eps, starter_code, reference_language, reference_source)
values (sqlc.arg(question_id), sqlc.arg(course_id), sqlc.arg(languages), sqlc.arg(time_limit_ms), sqlc.arg(memory_limit_mb), sqlc.arg(output_limit_kb), sqlc.arg(checker),
        sqlc.narg(float_eps), sqlc.arg(starter_code), sqlc.narg(reference_language), sqlc.narg(reference_source))
returning *;

-- name: CodeProblemGet :one
select * from code_problems where course_id = sqlc.arg(course_id) and question_id = sqlc.arg(question_id);

-- name: CodeProblemLock :one
select * from code_problems where course_id = sqlc.arg(course_id) and question_id = sqlc.arg(question_id) for update;

-- name: CodeProblemUpdate :one
-- `bump`: giới hạn / checker / ngôn ngữ / lời giải mẫu đổi → tests_version + 1 và gỡ cờ đã kiểm; chỉ `starter_code` đổi thì giữ nguyên (góp ý #2 (a)).
update code_problems
set languages = sqlc.arg(languages), time_limit_ms = sqlc.arg(time_limit_ms), memory_limit_mb = sqlc.arg(memory_limit_mb), output_limit_kb = sqlc.arg(output_limit_kb),
    checker = sqlc.arg(checker), float_eps = sqlc.narg(float_eps), starter_code = sqlc.arg(starter_code),
    reference_language = sqlc.narg(reference_language), reference_source = sqlc.narg(reference_source),
    tests_version = tests_version + case when sqlc.arg(bump)::boolean then 1 else 0 end,
    reference_verified_version = case when sqlc.arg(bump)::boolean or sqlc.arg(clear_verified)::boolean then null else reference_verified_version end,
    reference_verified_at = case when sqlc.arg(bump)::boolean or sqlc.arg(clear_verified)::boolean then null else reference_verified_at end,
    updated_at = now()
where course_id = sqlc.arg(course_id) and question_id = sqlc.arg(question_id)
returning *;

-- name: CodeProblemBumpTests :one
-- Mỗi thay đổi tập test: tests_version + 1, gỡ cờ lời giải mẫu đã kiểm.
update code_problems
set tests_version = tests_version + 1, reference_verified_version = null, reference_verified_at = null, updated_at = now()
where course_id = sqlc.arg(course_id) and question_id = sqlc.arg(question_id)
returning *;

-- name: CodeProblemMarkVerified :execrows
-- Chỉ ghi cờ khi tests_version chưa đổi từ lúc job bắt đầu (job chạy trên bản test cũ không được "xác minh" bản mới).
update code_problems set reference_verified_version = tests_version, reference_verified_at = now(), updated_at = now()
where course_id = sqlc.arg(course_id) and question_id = sqlc.arg(question_id) and tests_version = sqlc.arg(tests_version);

-- name: TestcaseList :many
select id, position, name, is_sample, weight, input, input_blob_key, expected, expected_blob_key, input_bytes, expected_bytes, source, approved, created_at, updated_at
from code_testcases
where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id)
  and (sqlc.narg(cursor_pos)::int is null or position > sqlc.narg(cursor_pos))
order by position
limit sqlc.arg(max_rows);

-- name: TestcaseGet :one
select * from code_testcases where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id) and id = sqlc.arg(id);

-- name: TestcaseStats :one
select count(*) filter (where approved)::int as total,
       count(*)::int as total_all,
       count(*) filter (where approved and is_sample)::int as samples,
       count(*) filter (where approved and not is_sample)::int as hidden,
       coalesce(sum(weight) filter (where approved), 0)::int as total_weight,
       coalesce(max(position), 0)::int as max_position
from code_testcases where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id);

-- name: TestcaseInsert :one
insert into code_testcases (course_id, problem_id, position, name, is_sample, weight, input, input_blob_key, expected, expected_blob_key, input_bytes, expected_bytes, source, approved)
values (sqlc.arg(course_id), sqlc.arg(problem_id), sqlc.arg(position), sqlc.arg(name), sqlc.arg(is_sample), sqlc.arg(weight), sqlc.narg(input), sqlc.narg(input_blob_key),
        sqlc.narg(expected), sqlc.narg(expected_blob_key), sqlc.arg(input_bytes), sqlc.arg(expected_bytes), sqlc.arg(source), sqlc.arg(approved))
returning *;

-- name: TestcaseUpdate :one
update code_testcases
set name = sqlc.arg(name), is_sample = sqlc.arg(is_sample), weight = sqlc.arg(weight), position = sqlc.arg(position),
    input = sqlc.narg(input), input_blob_key = sqlc.narg(input_blob_key), expected = sqlc.narg(expected), expected_blob_key = sqlc.narg(expected_blob_key),
    input_bytes = sqlc.arg(input_bytes), expected_bytes = sqlc.arg(expected_bytes), updated_at = now()
where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id) and id = sqlc.arg(id)
returning *;

-- name: TestcaseDelete :one
delete from code_testcases where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id) and id = sqlc.arg(id)
returning input_blob_key, expected_blob_key;

-- name: TestcasesDeleteAll :many
delete from code_testcases where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id) returning input_blob_key, expected_blob_key;

-- name: TestcasesApprove :execrows
update code_testcases set approved = true, updated_at = now()
where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id) and id = any(sqlc.arg(ids)::text[]::uuid[]) and not approved;

-- name: TestcaseNames :many
select name from code_testcases where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id);

-- name: TestcaseSetPosition :exec
update code_testcases set position = sqlc.arg(position), updated_at = now()
where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id) and id = sqlc.arg(id);

-- name: TestcaseIDsByPosition :many
select id from code_testcases where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id) order by position, id;
