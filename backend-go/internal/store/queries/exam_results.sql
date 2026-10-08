-- Chấm xong, công bố, kết quả, sửa điểm, chấm lại, phúc khảo (US-PE-08, SRS FEAT-weekly-exam 4.8, 5.8, 5.13). Mọi truy vấn bắt đầu bằng course_id (luật 13).

-- name: AttemptFinalSubmissions :many
-- Bản SUBMIT tính điểm (mới nhất chưa bị thay) của mỗi câu code trong lượt (SRS 4.4.5).
select distinct on (s.item_id) s.id, s.item_id, s.problem_id, s.status, s.verdict, s.compile_ok, s.compile_log, s.results, s.passed_weight, s.total_weight,
       s.language, s.source, s.created_at, s.judged_at, s.auto
from code_submissions s
where s.course_id = sqlc.arg(course_id) and s.attempt_id = sqlc.arg(attempt_id) and s.kind = 'SUBMIT' and s.status <> 'SUPERSEDED'
order by s.item_id, s.created_at desc, s.id desc;

-- name: AttemptSetGraded :one
-- `where status = 'GRADING'`: hai nơi cùng hoàn tất một lượt (tick + sự kiện) chỉ một nơi thắng.
update exam_attempts
set status = 'GRADED', auto_score = sqlc.arg(auto_score), breakdown = sqlc.arg(breakdown), graded_at = sqlc.arg(at), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'GRADING'
returning id, student_id, exam_id, version;

-- name: AttemptsToFinishGrading :many
-- Lượt GRADING không còn bản nộp cuối nào đang QUEUED / RUNNING (lô 200): đủ điều kiện thử hoàn tất chấm.
select a.id, a.course_id from exam_attempts a
where a.status = 'GRADING'
  and not exists (select 1 from code_submissions s where s.course_id = a.course_id and s.attempt_id = a.id and s.kind = 'SUBMIT' and s.status in ('QUEUED', 'RUNNING'))
order by a.submitted_at nulls last, a.id
limit 200;

-- name: AttemptGetByID :one
select * from exam_attempts where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: ExamPublishDue :many
-- SRS 4.8.2: bài CLOSED, không hoãn, không đang chấm lại, mọi lượt đã GRADED → PUBLISHED đúng MỘT lần (WHERE status = 'CLOSED').
update exams x set status = 'PUBLISHED', published_at = sqlc.arg(now), version = version + 1, updated_at = now()
where x.status = 'CLOSED' and not x.publish_hold and not x.regrading
  and (sqlc.narg(exam_id)::uuid is null or x.id = sqlc.narg(exam_id)::uuid)
  and not exists (select 1 from exam_attempts a where a.course_id = x.course_id and a.exam_id = x.id and a.status <> 'GRADED')
returning x.id, x.course_id, x.title;

-- name: ExamSetHold :one
update exams set publish_hold = sqlc.arg(hold), version = version + 1, updated_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and version = sqlc.arg(version)
returning *;

-- name: ExamGradingProgress :one
-- "Đang chấm {x}/{y}": y = lượt đã nộp; x = đã GRADED.
select count(*) filter (where status = 'GRADED')::int as graded, count(*) filter (where status in ('GRADING', 'GRADED'))::int as submitted,
       count(*) filter (where status = 'IN_PROGRESS')::int as in_progress, count(*) filter (where status = 'GRADING')::int as grading
from exam_attempts where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id);

-- name: ExamEnrolledCount :one
select count(*)::int from enrollments where course_id = sqlc.arg(course_id) and role_in_course = 'STUDENT' and status = 'ACTIVE';

-- name: ExamResultRows :many
-- Mọi sinh viên ACTIVE của lớp + lượt của họ (nếu có). Ô tìm theo tên / MSSV lọc ở service (MSSV không được dùng làm điều kiện truy vấn — SRS 4.2.5). Sắp xếp và phân trang con trỏ làm ở service (lớp ≤ 1.000 sinh viên: một truy vấn, số truy vấn không đổi theo cỡ lớp).
select u.id as student_id, u.full_name, coalesce(u.student_code, '') as student_code,
       a.id as attempt_id, a.status as attempt_status, a.auto_score, a.adjusted_score, a.submitted_at, a.submit_reason,
       coalesce(case when sqlc.arg(with_flags)::boolean then (select count(*) from similarity_reports r where r.course_id = e.course_id and r.exam_id = e.id and r.flagged
              and (r.attempt_a = a.id or r.attempt_b = a.id)
              and r.run_id = (select l.run_id from similarity_reports l where l.course_id = e.course_id and l.exam_id = e.id order by l.created_at desc, l.id desc limit 1)) end, 0)::int as sim_flags,
       coalesce(case when sqlc.arg(with_flags)::boolean then (select count(*) from exam_events ev where ev.course_id = e.course_id and ev.attempt_id = a.id and ev.type = 'TAB_HIDDEN') end, 0)::int as tab_hidden,
       coalesce(case when sqlc.arg(with_flags)::boolean then (select count(*) from exam_events ev where ev.course_id = e.course_id and ev.attempt_id = a.id and ev.type = 'PASTE') end, 0)::int as paste
from exams e
join enrollments en on en.course_id = e.course_id and en.role_in_course = 'STUDENT' and en.status = 'ACTIVE'
join users u on u.id = en.user_id
left join exam_attempts a on a.course_id = e.course_id and a.exam_id = e.id and a.student_id = u.id
where e.course_id = sqlc.arg(course_id) and e.id = sqlc.arg(exam_id)
  and (sqlc.arg(status)::text = '' or (sqlc.arg(status)::text = 'ABSENT' and a.id is null) or (sqlc.arg(status)::text <> 'ABSENT' and a.status::text = sqlc.arg(status)::text))
order by u.id;

-- name: ExamResultProgress :one
select (select count(*) from enrollments en where en.course_id = sqlc.arg(course_id) and en.role_in_course = 'STUDENT' and en.status = 'ACTIVE'
          and not exists (select 1 from exam_attempts a where a.course_id = en.course_id and a.exam_id = sqlc.arg(exam_id) and a.student_id = en.user_id))::int as without_attempt,
       count(*) filter (where a.status = 'IN_PROGRESS')::int as in_progress, count(*) filter (where a.status = 'GRADING')::int as grading, count(*) filter (where a.status = 'GRADED')::int as graded
from exam_attempts a where a.course_id = sqlc.arg(course_id) and a.exam_id = sqlc.arg(exam_id);

-- name: ExamScoredAttempts :many
-- Lượt GRADED của bài (điểm chính thức + breakdown) cho thống kê / CSV.
select a.id, a.student_id, a.auto_score, a.adjusted_score, a.breakdown from exam_attempts a
where a.course_id = sqlc.arg(course_id) and a.exam_id = sqlc.arg(exam_id) and a.status = 'GRADED';

-- name: ExamItemsMeta :many
select i.id as item_id, i.question_id, i.position, i.points, i.override, q.type, q.title, q.stem, q.explanation, q.answer_key
from exam_items i join question_bank q on q.course_id = i.course_id and q.id = i.question_id
where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id)
order by i.position;

-- name: AttemptAdjust :one
update exam_attempts
set adjusted_score = sqlc.narg(score), adjusted_reason = sqlc.narg(reason), adjusted_by = sqlc.narg(by), adjusted_at = sqlc.narg(at), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'GRADED' and version = sqlc.arg(version)
returning *;

-- name: ItemSetOverride :one
update exam_items set override = sqlc.narg(override) where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) and id = sqlc.arg(id) returning id;

-- name: ExamSetRegrading :exec
update exams set regrading = sqlc.arg(regrading), updated_at = now() where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: AttemptsGradedOfExam :many
select id, student_id, version from exam_attempts where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) and status = 'GRADED';

-- name: AttemptUpdateGraded :one
-- Tính lại điểm một lượt GRADED (chấm lại / override). Lượt KHÔNG quay về GRADING. Điều kiện `version` chống ghi chồng.
update exam_attempts set auto_score = sqlc.arg(auto_score), breakdown = sqlc.arg(breakdown), graded_at = sqlc.arg(at), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'GRADED' and version = sqlc.arg(version)
returning id, student_id, version;

-- name: RegradeResetSubmissions :many
-- Đặt lại bản nộp CUỐI trong phạm vi về QUEUED (SRS 4.8.4). `scope`: all | item | attempt | errors. Idempotent theo (bản nộp, tests_version): bản đã chấm bằng bộ test HIỆN HÀNH
-- (`tests_version` của câu) thì không chấm đôi; bản lỗi (ERROR / IE) luôn được chấm lại. `errors` chỉ chấm các bản lỗi.
update code_submissions c
set status = 'QUEUED', fail_count = 0, next_attempt_at = sqlc.arg(now), enqueued_at = null, lease_until = null, verdict = null, results = null, tests_version = null, compile_ok = null,
    compile_log = null, passed_weight = null, total_weight = null, judged_at = null, attempts = 0
where c.id in (
    select distinct on (s.attempt_id, s.item_id) s.id
    from code_submissions s
    where s.course_id = sqlc.arg(course_id) and s.exam_id = sqlc.arg(exam_id) and s.kind = 'SUBMIT' and s.status <> 'SUPERSEDED'
      and (sqlc.arg(scope)::text in ('all', 'errors') or (sqlc.arg(scope)::text = 'item' and s.item_id = sqlc.narg(item_id)::uuid) or (sqlc.arg(scope)::text = 'attempt' and s.attempt_id = sqlc.narg(attempt_id)::uuid))
    order by s.attempt_id, s.item_id, s.created_at desc, s.id desc
  ) and c.status in ('DONE', 'ERROR')
  and (c.status = 'ERROR' or c.verdict = 'IE'
       or (sqlc.arg(scope)::text <> 'errors' and c.tests_version is distinct from (select cp.tests_version from code_problems cp where cp.course_id = c.course_id and cp.question_id = c.problem_id)))
returning c.id, c.attempt_id;

-- name: ExamRegradePending :one
-- Còn bản SUBMIT nào của bài đang QUEUED / RUNNING (chấm lại chưa xong)?
select count(*)::int from code_submissions where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) and kind = 'SUBMIT' and status in ('QUEUED', 'RUNNING');

-- name: ExamRegradingExams :many
-- Chỉ bài đã yên ≥ 15 s kể từ lúc đặt cờ: việc `exam.regrade` (xếp ở request) có thời gian đặt lại bản nộp trước khi tick coi "không còn gì chờ" là xong.
select id, course_id from exams where regrading and status in ('CLOSED', 'PUBLISHED') and updated_at < now() - interval '15 seconds' limit 100;

-- name: AttemptsNeedingRecompute :many
-- Lượt GRADED của bài đang chấm lại mà một bản nộp cuối có `judged_at > graded_at` (SRS 4.8.4).
select distinct a.id, a.version from exam_attempts a
join code_submissions s on s.course_id = a.course_id and s.attempt_id = a.id and s.kind = 'SUBMIT' and s.status = 'DONE'
where a.course_id = sqlc.arg(course_id) and a.exam_id = sqlc.arg(exam_id) and a.status = 'GRADED' and s.judged_at is not null and (a.graded_at is null or s.judged_at > a.graded_at);

-- name: AppealInsert :one
insert into exam_appeals (course_id, exam_id, attempt_id, student_id, reason) values (sqlc.arg(course_id), sqlc.arg(exam_id), sqlc.arg(attempt_id), sqlc.arg(student_id), sqlc.arg(reason))
on conflict do nothing returning *;

-- name: AppealByAttempt :one
select * from exam_appeals where course_id = sqlc.arg(course_id) and attempt_id = sqlc.arg(attempt_id);

-- name: AppealList :many
select p.id, p.attempt_id, p.status, p.reason, p.response, p.created_at, p.responded_at, p.score_before, p.score_after, p.version, u.full_name, coalesce(u.student_code, '') as student_code
from exam_appeals p join users u on u.id = p.student_id
where p.course_id = sqlc.arg(course_id) and p.exam_id = sqlc.arg(exam_id) and (sqlc.arg(status)::text = '' or p.status::text = sqlc.arg(status)::text)
  and (sqlc.narg(cursor_at)::timestamptz is null or (p.created_at, p.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by p.created_at desc, p.id desc
limit sqlc.arg(row_limit);

-- name: AppealLock :one
select * from exam_appeals where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) and id = sqlc.arg(id) for update;

-- name: AppealAnswer :one
update exam_appeals set status = sqlc.arg(status), response = sqlc.arg(response), responded_by = sqlc.arg(by), responded_at = sqlc.arg(at), score_before = sqlc.narg(score_before), score_after = sqlc.narg(score_after),
       version = version + 1, updated_at = now()
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'OPEN' and version = sqlc.arg(version)
returning *;

-- name: TodayExamResults :many
-- Việc của sinh viên sau công bố: EXAM_RESULT (7 ngày kể từ công bố) và EXAM_APPEAL_REPLY (có phản hồi chưa xem trong 7 ngày); một truy vấn cho mọi lớp.
select e.id as exam_id, e.course_id, e.title, e.published_at, a.id as attempt_id, p.status as appeal_status, p.responded_at
from exams e
join exam_attempts a on a.course_id = e.course_id and a.exam_id = e.id and a.student_id = sqlc.arg(user_id)
left join exam_appeals p on p.course_id = e.course_id and p.attempt_id = a.id
where e.course_id = any(sqlc.arg(course_ids)::text[]::uuid[]) and e.status = 'PUBLISHED' and (e.published_at > sqlc.arg(since)::timestamptz or p.responded_at > sqlc.arg(since)::timestamptz);

-- name: ExamNotifyUsers :execrows
-- Thông báo của bài thi (EXAM_PUBLISHED / EXAM_REGRADED / EXAM_APPEAL_*) cho danh sách người; khử trùng `<dedupe_prefix>:<user_id>`.
insert into notifications (user_id, course_id, type, title, body, link, dedupe_key)
select u, sqlc.arg(course_id), sqlc.arg(type), sqlc.arg(title), null, sqlc.arg(link), sqlc.arg(dedupe_prefix)::text || ':' || u::text
from unnest(sqlc.arg(user_ids)::text[]::uuid[]) as u
on conflict (user_id, dedupe_key) where dedupe_key is not null do nothing;

-- name: ExamAttemptStudents :many
select student_id from exam_attempts where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id);

-- name: ExamCourseStaff :many
select user_id from enrollments where course_id = sqlc.arg(course_id) and role_in_course in ('TEACHER', 'TA') and status = 'ACTIVE';

-- name: ExamResultSamples :many
-- Test MẪU đã duyệt của các câu code trong bài, kèm id để ghép với `breakdown` (sinh viên được thấy; test ẩn KHÔNG có ở đây).
select t.id, t.problem_id, t.name, t.input, t.expected
from code_testcases t
join exam_items i on i.course_id = t.course_id and i.question_id = t.problem_id
where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id) and t.is_sample and t.approved and t.input is not null and t.expected is not null
order by t.problem_id, t.position;

-- name: AttemptStaffHead :one
select a.*, u.full_name, coalesce(u.student_code, '') as student_code
from exam_attempts a join users u on u.id = a.student_id
where a.course_id = sqlc.arg(course_id) and a.exam_id = sqlc.arg(exam_id) and a.id = sqlc.arg(id);

-- name: AttemptSubmitHistory :many
-- Mọi bản nộp (SUBMIT) của lượt, cho Staff: verdict TỪNG test kể cả test ẩn nằm trong `results`.
select id, item_id, status, verdict, language, source, created_at, judged_at, compile_ok, compile_log, results, tests_version, passed_weight, total_weight
from code_submissions
where course_id = sqlc.arg(course_id) and attempt_id = sqlc.arg(attempt_id) and kind = 'SUBMIT'
order by item_id, created_at, id;
