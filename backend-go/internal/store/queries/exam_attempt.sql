-- Lượt làm của sinh viên (US-PE-05, SRS FEAT-weekly-exam 5.8–5.9, 4.3). Mọi truy vấn bắt đầu bằng course_id (luật 13).

-- name: AttemptInsert :one
-- Hai yêu cầu bắt đầu song song cho đúng MỘT dòng nhờ UNIQUE(exam_id, student_id); 0 dòng trả về = đã có lượt.
insert into exam_attempts (course_id, exam_id, student_id, started_at, deadline_at, writer_tab, writer_seen_at)
values (sqlc.arg(course_id), sqlc.arg(exam_id), sqlc.arg(student_id), sqlc.arg(started_at), sqlc.arg(deadline_at), sqlc.narg(writer_tab), sqlc.arg(started_at))
on conflict (exam_id, student_id) do nothing
returning *;

-- name: AttemptByStudent :one
select * from exam_attempts where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) and student_id = sqlc.arg(student_id);

-- name: AttemptLock :one
-- Lượt của CHÍNH sinh viên (người khác → không có dòng → 404), khoá hàng để kiểm trạng thái / người ghi.
select * from exam_attempts
where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) and id = sqlc.arg(id) and student_id = sqlc.arg(student_id)
for update;

-- name: AttemptOwn :one
-- Đọc (không khoá) lượt của CHÍNH sinh viên; người khác → không có dòng → 404.
select * from exam_attempts
where course_id = sqlc.arg(course_id) and exam_id = sqlc.arg(exam_id) and id = sqlc.arg(id) and student_id = sqlc.arg(student_id);

-- name: AttemptLockByID :one
select * from exam_attempts where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) for update;

-- name: AttemptSetWriter :exec
update exam_attempts set writer_tab = sqlc.narg(writer_tab), writer_seen_at = sqlc.arg(seen_at)
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: AnswersUpsert :exec
-- Một lệnh cho cả lô (SRS 4.3.3): `rows` = [{"item_id":…,"answer":{…}}].
insert into exam_answers (attempt_id, item_id, course_id, answer, saved_at)
select sqlc.arg(attempt_id), x.item_id, sqlc.arg(course_id), x.answer, sqlc.arg(saved_at)
from jsonb_to_recordset(sqlc.arg(rows)::jsonb) as x(item_id uuid, answer jsonb)
on conflict (attempt_id, item_id) do update set answer = excluded.answer, saved_at = excluded.saved_at;

-- name: AnswersDelete :exec
-- Mảng rỗng = xoá lựa chọn (câu coi như chưa trả lời).
delete from exam_answers where course_id = sqlc.arg(course_id) and attempt_id = sqlc.arg(attempt_id) and item_id = any(sqlc.arg(item_ids)::text[]::uuid[]);

-- name: AnswersList :many
select item_id, answer from exam_answers where course_id = sqlc.arg(course_id) and attempt_id = sqlc.arg(attempt_id);

-- name: AttemptFinish :one
-- `where status = 'IN_PROGRESS'`: nộp lần hai (hoặc tick chạy song song) không tìm thấy dòng → đã nộp, idempotent.
update exam_attempts
set status = sqlc.arg(status), submitted_at = sqlc.arg(submitted_at), submit_reason = sqlc.arg(submit_reason),
    auto_score = sqlc.narg(auto_score), graded_at = sqlc.narg(graded_at), breakdown = sqlc.narg(breakdown), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'IN_PROGRESS'
returning *;

-- name: AttemptsDue :many
-- Lượt IN_PROGRESS đã quá deadline + grace (lô 200). `by_close`: hạn do giờ đóng của lớp (CLOSED) hay do thời lượng (TIMEOUT).
select a.id, a.course_id, a.exam_id, a.student_id, a.deadline_at, (a.deadline_at >= e.closes_at)::boolean as by_close
from exam_attempts a
join exams e on e.course_id = a.course_id and e.id = a.exam_id
where a.status = 'IN_PROGRESS' and a.deadline_at + make_interval(secs => sqlc.arg(grace_seconds)::int) < sqlc.arg(now)::timestamptz
order by a.deadline_at, a.id
limit 200;

-- name: ExamGradeItems :many
-- Mục kèm đáp án đúng và `override` để chấm (CHỈ dùng ở phía máy chủ khi chấm — không bao giờ vào DTO sinh viên).
select i.id as item_id, i.question_id, i.points, i.override, q.type, q.answer_key
from exam_items i
join question_bank q on q.course_id = i.course_id and q.id = i.question_id
where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id)
order by i.position;

-- name: ExamEventInsert :exec
insert into exam_events (course_id, exam_id, attempt_id, student_id, type, meta)
values (sqlc.arg(course_id), sqlc.arg(exam_id), sqlc.arg(attempt_id), sqlc.arg(student_id), sqlc.arg(type), sqlc.arg(meta));

-- name: AttemptCountsByAnswer :one
select count(*)::int as answered from exam_answers where course_id = sqlc.arg(course_id) and attempt_id = sqlc.arg(attempt_id);
