-- Bài thi (US-PE-04, SRS FEAT-weekly-exam 5.6–5.7, 4.2). Mọi truy vấn bắt đầu bằng course_id (luật 13).
-- `effective_status` tính ở đây bằng `now` truyền vào (đồng hồ của ứng dụng, giả được trong test): SCHEDULED + now ≥ opens_at → OPEN; còn lại theo closes_at.

-- name: ExamInsert :one
insert into exams (course_id, title, instructions, opens_at, closes_at, duration_minutes, shuffle_questions, shuffle_options, max_score, rounding_step,
                   multi_scoring, reveal_answers, appeal_days, created_by)
values (sqlc.arg(course_id), sqlc.arg(title), sqlc.narg(instructions), sqlc.narg(opens_at), sqlc.narg(closes_at), sqlc.narg(duration_minutes),
        sqlc.arg(shuffle_questions), sqlc.arg(shuffle_options), sqlc.arg(max_score), sqlc.arg(rounding_step), sqlc.arg(multi_scoring),
        sqlc.arg(reveal_answers), sqlc.arg(appeal_days), sqlc.arg(created_by))
returning *;

-- name: ExamGet :one
select * from exams where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: ExamLock :one
select * from exams where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) for update;

-- name: ExamShare :one
-- Bắt đầu làm bài: khoá chia sẻ hàng bài để `unschedule` / `extend` (FOR UPDATE) chờ — không có lượt nào chen vào giữa lúc kiểm trạng thái và INSERT (SRS 4.3.1).
select * from exams where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) for share;

-- name: ExamUpdate :one
-- Ghi mọi trường người dùng sửa được; `expected_version` là khoá lạc quan (0 dòng = sai version). Service đã quyết trường nào được đổi theo trạng thái.
update exams
set title = sqlc.arg(title), instructions = sqlc.narg(instructions), opens_at = sqlc.narg(opens_at), closes_at = sqlc.narg(closes_at),
    duration_minutes = sqlc.narg(duration_minutes), shuffle_questions = sqlc.arg(shuffle_questions), shuffle_options = sqlc.arg(shuffle_options),
    max_score = sqlc.arg(max_score), rounding_step = sqlc.arg(rounding_step), multi_scoring = sqlc.arg(multi_scoring),
    reveal_answers = sqlc.arg(reveal_answers), appeal_days = sqlc.arg(appeal_days), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and version = sqlc.arg(expected_version)
returning *;

-- name: ExamSetKind :exec
update exams set kind = sqlc.arg(kind), version = version + 1 where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: ExamSetStatus :one
-- Chuyển trạng thái một bài (schedule / unschedule): `from_status` làm điều kiện để hai người gọi cùng lúc không chuyển hai lần.
update exams set status = sqlc.arg(to_status), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = sqlc.arg(from_status)
returning *;

-- name: ExamExtend :one
update exams set closes_at = sqlc.arg(closes_at), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id)
returning *;

-- name: ExamAttemptsExtend :execrows
-- Gia hạn: lượt đang làm bị chặn bởi mốc đóng cũ (deadline ≥ mốc cũ) được tính lại min(started_at + thời lượng, mốc mới).
update exam_attempts a
set deadline_at = least(a.started_at + make_interval(mins => e.duration_minutes::int), sqlc.arg(new_closes_at)::timestamptz)
from exams e
where e.course_id = a.course_id and e.id = a.exam_id
  and a.course_id = sqlc.arg(course_id) and a.exam_id = sqlc.arg(exam_id)
  and a.status = 'IN_PROGRESS' and a.deadline_at >= sqlc.arg(old_closes_at)::timestamptz;

-- name: ExamAttemptCounts :one
select count(*)::int as started, (count(*) filter (where status = 'GRADED'))::int as graded
from exam_attempts where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id);

-- name: ExamDeleteDraft :execrows
delete from exams where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'DRAFT';

-- name: ExamClone :one
-- Bản sao DRAFT: cùng cài đặt, không giờ; tiêu đề thêm " (bản sao)" (cắt tiêu đề để vẫn ≤ 120 ký tự).
insert into exams (course_id, title, instructions, kind, shuffle_questions, shuffle_options, max_score, rounding_step, multi_scoring, reveal_answers, appeal_days, created_by)
select x.course_id, left(x.title, 110) || ' (bản sao)', x.instructions, x.kind, x.shuffle_questions, x.shuffle_options, x.max_score, x.rounding_step, x.multi_scoring,
       x.reveal_answers, x.appeal_days, sqlc.arg(created_by)
from exams x where x.course_id = sqlc.arg(course_id) and x.id = sqlc.arg(id)
returning *;

-- name: ExamItemsClone :exec
insert into exam_items (course_id, exam_id, question_id, position, points)
select i.course_id, sqlc.arg(new_exam_id), i.question_id, i.position, i.points
from exam_items i where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id);

-- name: ExamList :many
-- Danh sách theo vai: `student_only` ẩn DRAFT; `student_id` nối lượt làm của chính người đó. `status` lọc theo effective_status.
-- Đếm mục / lượt bằng truy vấn con (không N+1). Phân trang con trỏ (created_at desc, id desc).
with e as (
  select x.*,
         (case when x.status in ('SCHEDULED', 'OPEN') and x.closes_at <= sqlc.arg(now)::timestamptz then 'CLOSED'
               when x.status = 'SCHEDULED' and x.opens_at <= sqlc.arg(now)::timestamptz then 'OPEN'
               else x.status::text end)::exam_status as eff
  from exams x
  where x.course_id = sqlc.arg(course_id)
    and (not sqlc.arg(student_only)::boolean or x.status <> 'DRAFT')
    and (sqlc.narg(exam_id)::uuid is null or x.id = sqlc.narg(exam_id)::uuid)
    and (sqlc.narg(cursor_at)::timestamptz is null or (x.created_at, x.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
)
select e.id, e.course_id, e.title, e.instructions, e.kind, e.status, e.eff as effective_status, e.opens_at, e.closes_at, e.duration_minutes, e.max_score,
       e.published_at, e.version, e.created_at,
       (select count(*) from exam_items i where i.course_id = e.course_id and i.exam_id = e.id)::int as items_count,
       (select count(*) from exam_attempts a where a.course_id = e.course_id and a.exam_id = e.id)::int as attempts_started,
       (select count(*) from exam_attempts a where a.course_id = e.course_id and a.exam_id = e.id and a.status = 'GRADED')::int as attempts_graded,
       ma.id as my_attempt_id, ma.status as my_attempt_status, ma.deadline_at as my_deadline_at, ma.submitted_at as my_submitted_at,
       ma.adjusted_score as my_adjusted_score, ma.auto_score as my_auto_score -- service chỉ lộ điểm khi PUBLISHED
from e
left join exam_attempts ma on ma.course_id = e.course_id and ma.exam_id = e.id and ma.student_id = sqlc.narg(student_id)
where (sqlc.narg(status)::exam_status is null or e.eff = sqlc.narg(status))
order by e.created_at desc, e.id desc
limit sqlc.arg(max_rows);

-- name: ExamItemList :many
select i.id, i.question_id, i.position, i.points, q.type, q.title, q.topic, q.difficulty, q.review_status, q.archived_at
from exam_items i
join question_bank q on q.course_id = i.course_id and q.id = i.question_id
where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id)
order by i.position;

-- name: ExamItemListShared :many
-- Như ExamItemList nhưng khoá chia sẻ các câu hỏi: một `REJECT` / lưu trữ đang chạy song song phải xong trước (hoặc chờ tới khi `schedule` commit) — SRS 4.2.4, AC10.
select i.id, i.question_id, i.position, i.points, q.type, q.title, q.topic, q.difficulty, q.review_status, q.archived_at
from exam_items i
join question_bank q on q.course_id = i.course_id and q.id = i.question_id
where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id)
order by i.position
for share of q;

-- name: ExamItemsDelete :exec
delete from exam_items where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id);

-- name: ExamItemInsert :exec
insert into exam_items (course_id, exam_id, question_id, position, points)
values (sqlc.arg(course_id), sqlc.arg(exam_id), sqlc.arg(question_id), sqlc.arg(position), sqlc.arg(points));

-- name: ExamQuestionsByIDs :many
-- Câu của CÙNG lớp theo danh sách id (id của lớp khác không trả về → QUESTION_NOT_IN_COURSE). `::text[]::uuid[]` vì PgBouncer.
select id, type, review_status, archived_at from question_bank
where course_id = sqlc.arg(course_id) and id = any(sqlc.arg(ids)::text[]::uuid[]);

-- name: ExamPreviewItems :many
select i.id as item_id, i.position, i.points, q.id as question_id, q.type, q.stem,
       cp.languages, cp.time_limit_ms, cp.memory_limit_mb, cp.starter_code
from exam_items i
join question_bank q on q.course_id = i.course_id and q.id = i.question_id
left join code_problems cp on cp.course_id = q.course_id and cp.question_id = q.id
where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id)
order by i.position;

-- name: ExamPreviewOptions :many
select o.id, o.question_id, o.position, o.body, o.pinned_last
from question_options o
join exam_items i on i.course_id = o.course_id and i.question_id = o.question_id
where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id)
order by o.question_id, o.position;

-- name: ExamPreviewSamples :many
-- Test MẪU đã duyệt (sinh viên được thấy); test lưu ở kho đối tượng (> 64 KiB) không phải test mẫu thực tế nên bỏ.
select t.problem_id, t.name, t.input, t.expected
from code_testcases t
join exam_items i on i.course_id = t.course_id and i.question_id = t.problem_id
where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id)
  and t.is_sample and t.approved and t.input is not null and t.expected is not null
order by t.problem_id, t.position;

-- name: ExamCodeQuestions :many
-- Câu CODE của bài kèm tổng "đồng hồ" các test đã duyệt: mỗi test tốn tối đa 3 × time_limit_ms (clockLimit của máy chấm).
select q.id, q.title, cp.time_limit_ms,
       (select count(*) from code_testcases t where t.course_id = cp.course_id and t.problem_id = cp.question_id and t.approved)::int as approved_tests
from exam_items i
join question_bank q on q.course_id = i.course_id and q.id = i.question_id
join code_problems cp on cp.course_id = q.course_id and cp.question_id = q.id
where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id) and q.type = 'CODE'
order by i.position;

-- name: ExamTickOpen :many
-- (1) mở bài đến giờ (chưa quá closes_at); idempotent nhờ `where status = 'SCHEDULED'`.
update exams set status = 'OPEN', version = version + 1
where status = 'SCHEDULED' and opens_at <= sqlc.arg(now)::timestamptz and closes_at > sqlc.arg(now)::timestamptz
returning id, course_id;

-- name: ExamTickClose :many
-- (2) đóng bài đến hạn; SCHEDULED mà đã quá closes_at (máy chủ tắt dài) chuyển thẳng CLOSED.
update exams set status = 'CLOSED', version = version + 1
where status in ('SCHEDULED', 'OPEN') and closes_at <= sqlc.arg(now)::timestamptz
returning id, course_id;

-- name: ExamNotifyScheduled :execrows
-- Thông báo EXAM_SCHEDULED cho mọi sinh viên ACTIVE; khử trùng theo (bài, người).
insert into notifications (user_id, course_id, type, title, body, link, dedupe_key)
select e.user_id, e.course_id, 'EXAM_SCHEDULED', sqlc.arg(title), sqlc.arg(body), sqlc.arg(link), 'exam.scheduled:' || sqlc.arg(exam_id)::text || ':' || e.user_id::text
from enrollments e
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE'
on conflict (user_id, dedupe_key) where dedupe_key is not null do nothing;

-- name: ExamRecallScheduled :many
-- Thu hồi thông báo lịch của một bài (khi bỏ lịch): trả người đã nhận để báo "hoãn". Xoá cả dấu khử trùng nên lần lên lịch sau báo lại được.
delete from notifications
where course_id = sqlc.arg(course_id) and type = 'EXAM_SCHEDULED' and dedupe_key like 'exam.scheduled:' || sqlc.arg(exam_id)::text || ':%'
returning user_id;

-- name: ExamNotifyUnscheduled :execrows
insert into notifications (user_id, course_id, type, title, body, link, dedupe_key)
select u, sqlc.arg(course_id), 'EXAM_UNSCHEDULED', sqlc.arg(title), sqlc.arg(body), sqlc.arg(link), 'exam.unscheduled:' || sqlc.arg(outbox_id)::text || ':' || u::text
from unnest(sqlc.arg(user_ids)::text[]::uuid[]) as u
on conflict (user_id, dedupe_key) where dedupe_key is not null do nothing;

-- name: TodayStudentExams :many
-- "Hôm nay" của sinh viên: bài SCHEDULED / OPEN còn hạn, mở trong `horizon`, chưa có lượt làm hoặc lượt còn đang làm — MỘT truy vấn cho mọi lớp.
select e.id, e.course_id, e.title, e.opens_at, e.closes_at, e.duration_minutes, a.status as attempt_status, a.deadline_at as attempt_deadline
from exams e
left join exam_attempts a on a.course_id = e.course_id and a.exam_id = e.id and a.student_id = sqlc.arg(user_id)
where e.course_id = any(sqlc.arg(course_ids)::text[]::uuid[])
  and e.status in ('SCHEDULED', 'OPEN')
  and e.closes_at > sqlc.arg(now)::timestamptz
  and e.opens_at <= sqlc.arg(horizon)::timestamptz
  and (a.id is null or (a.status = 'IN_PROGRESS' and a.deadline_at > sqlc.arg(now)::timestamptz))
order by e.opens_at, e.id;
