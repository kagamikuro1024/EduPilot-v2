-- Luyện đề, thi thử (US-P9-05…09, SRS FEAT-practice 4.3, 5). Mọi truy vấn bắt đầu bằng course_id (luật 13).
-- "Đủ điều kiện" (FR-21, đề xuất tầng truy vấn): APPROVED, chưa lưu trữ, không thuộc bài thi CHƯA công bố (DRAFT / SCHEDULED / OPEN / CLOSED chưa PUBLISHED).

-- name: PracticePool :many
select q.id, q.topic, q.difficulty
from question_bank q
where q.course_id = sqlc.arg(course_id) and q.review_status = 'APPROVED' and q.archived_at is null
  and q.type::text = any(sqlc.arg(types)::text[])
  and (sqlc.arg(topic)::text = '' or q.topic = sqlc.arg(topic))
  and not exists (select 1 from exam_items i join exams e on e.course_id = i.course_id and e.id = i.exam_id
                  where i.course_id = q.course_id and i.question_id = q.id and e.status <> 'PUBLISHED');

-- name: PracticeTopics :many
select q.topic, count(*)::int as available
from question_bank q
where q.course_id = sqlc.arg(course_id) and q.review_status = 'APPROVED' and q.archived_at is null
  and q.type::text = any(sqlc.arg(types)::text[])
  and not exists (select 1 from exam_items i join exams e on e.course_id = i.course_id and e.id = i.exam_id
                  where i.course_id = q.course_id and i.question_id = q.id and e.status <> 'PUBLISHED')
group by q.topic
order by available desc, q.topic;

-- name: PracticeQuestionsByIDs :many
select id, type, topic, difficulty, stem, answer_key, explanation, citations, version
from question_bank where course_id = sqlc.arg(course_id) and id = any(sqlc.arg(ids)::text[]::uuid[]);

-- name: PracticeOptionsByQuestions :many
select id, question_id, position, body, pinned_last
from question_options where course_id = sqlc.arg(course_id) and question_id = any(sqlc.arg(ids)::text[]::uuid[])
order by question_id, position;

-- name: PracticeRecent :many
-- Lần làm GẦN NHẤT mỗi câu trong 30 ngày của một sinh viên (đã chấm: is_correct không NULL).
select distinct on (i.question_id) i.question_id, a.is_correct
from practice_answers a
join practice_items i on i.course_id = a.course_id and i.id = a.item_id
join practice_attempts t on t.course_id = a.course_id and t.id = a.attempt_id
where a.course_id = sqlc.arg(course_id) and t.student_id = sqlc.arg(student_id) and a.is_correct is not null and a.answered_at > now() - interval '30 days'
order by i.question_id, a.answered_at desc;

-- name: PracticeTopicStats :many
-- Mỗi chủ đề (từ bản chụp câu) trong 30 ngày: số câu đã chấm và số sai — chỉ của MỘT sinh viên.
select (i.question ->> 'topic')::text as topic, count(*)::int as answered, (count(*) filter (where not a.is_correct))::int as wrong, max(a.answered_at)::timestamptz as last_at
from practice_answers a
join practice_items i on i.course_id = a.course_id and i.id = a.item_id
join practice_attempts t on t.course_id = a.course_id and t.id = a.attempt_id
where a.course_id = sqlc.arg(course_id) and t.student_id = sqlc.arg(student_id) and a.is_correct is not null and a.answered_at > now() - interval '30 days'
group by 1
order by wrong desc, topic;

-- name: PracticeWrongCitations :many
-- `citations` của các câu SAI trong một chủ đề (30 ngày) của một sinh viên, để gợi ý tài liệu.
select (i.question -> 'citations')::jsonb as citations
from practice_answers a
join practice_items i on i.course_id = a.course_id and i.id = a.item_id
join practice_attempts t on t.course_id = a.course_id and t.id = a.attempt_id
where a.course_id = sqlc.arg(course_id) and t.student_id = sqlc.arg(student_id) and a.is_correct = false and a.answered_at > now() - interval '30 days'
  and (i.question ->> 'topic') = sqlc.arg(topic)::text;

-- name: PracticeSignals :many
-- Tín hiệu tổng hợp cho giảng viên (P10): theo chủ đề, KHÔNG định danh; chủ đề < k sinh viên bị loại ngay trong truy vấn (k-ẩn danh).
select (i.question ->> 'topic')::text as topic, count(*)::int as wrong_answers, count(distinct t.student_id)::int as students_affected,
       (array_agg(distinct i.question_id))[1:3]::uuid[] as sample_question_ids
from practice_answers a
join practice_items i on i.course_id = a.course_id and i.id = a.item_id
join practice_attempts t on t.course_id = a.course_id and t.id = a.attempt_id
where a.course_id = sqlc.arg(course_id) and a.is_correct = false and a.answered_at >= sqlc.arg(from_at) and a.answered_at < sqlc.arg(to_at)
group by 1
having count(distinct t.student_id) >= sqlc.arg(min_students)::int
order by wrong_answers desc, topic;

-- name: PracticeAttemptInsert :one
insert into practice_attempts (course_id, student_id, mode, topic, deadline_at, item_count, matrix, notes, tab_id)
values (sqlc.arg(course_id), sqlc.arg(student_id), sqlc.arg(mode), sqlc.narg(topic), sqlc.narg(deadline_at), sqlc.arg(item_count), sqlc.narg(matrix), sqlc.arg(notes), sqlc.narg(tab_id))
returning *;

-- name: PracticeItemInsert :exec
insert into practice_items (course_id, attempt_id, question_id, position, option_order, answer_key, question, question_version)
values (sqlc.arg(course_id), sqlc.arg(attempt_id), sqlc.arg(question_id), sqlc.arg(position), sqlc.arg(option_order), sqlc.arg(answer_key), sqlc.arg(question), sqlc.arg(question_version));

-- name: PracticeAttemptGet :one
select * from practice_attempts where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: PracticeAttemptLock :one
select * from practice_attempts where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) for update;

-- name: PracticeAttemptOpen :one
-- Lượt DỞ của (sinh viên, lớp, chế độ, chủ đề); MOCK: topic rỗng.
select * from practice_attempts
where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id) and mode = sqlc.arg(mode) and coalesce(topic, '') = sqlc.arg(topic)::text and status = 'IN_PROGRESS';

-- name: PracticeItemsWithAnswers :many
select i.id, i.question_id, i.position, i.option_order, i.answer_key, i.question, i.question_version,
       a.answer, a.is_correct, a.locked, a.version as answer_version, a.answered_at
from practice_items i
left join practice_answers a on a.course_id = i.course_id and a.item_id = i.id
where i.course_id = sqlc.arg(course_id) and i.attempt_id = sqlc.arg(attempt_id)
order by i.position;

-- name: PracticeItemGet :one
select i.id, i.question_id, i.position, i.option_order, i.answer_key, i.question, i.question_version,
       a.answer, a.is_correct, a.locked, a.version as answer_version, a.answered_at
from practice_items i
left join practice_answers a on a.course_id = i.course_id and a.item_id = i.id
where i.course_id = sqlc.arg(course_id) and i.attempt_id = sqlc.arg(attempt_id) and i.id = sqlc.arg(item_id);

-- name: PracticeAnswerUpsert :one
-- Ghi đáp án: lần đầu chèn; lần sau chỉ khi đúng `expected_version` và chưa `locked` (0 dòng = xung đột / đã khoá — service đọc lại để phân biệt).
insert into practice_answers (course_id, attempt_id, item_id, answer, is_correct, answered_at, locked)
values (sqlc.arg(course_id), sqlc.arg(attempt_id), sqlc.arg(item_id), sqlc.arg(answer), sqlc.narg(is_correct), sqlc.arg(answered_at), sqlc.arg(locked))
on conflict (item_id) do update
set answer = excluded.answer, is_correct = excluded.is_correct, answered_at = excluded.answered_at, locked = excluded.locked, version = practice_answers.version + 1
where practice_answers.version = sqlc.arg(expected_version) and not practice_answers.locked
returning version;

-- name: PracticeAnswerSetResult :exec
-- Chấm khi nộp: đặt is_correct (câu bỏ trống: chèn dòng {} sai).
insert into practice_answers (course_id, attempt_id, item_id, answer, is_correct, answered_at, locked)
values (sqlc.arg(course_id), sqlc.arg(attempt_id), sqlc.arg(item_id), '{}', sqlc.arg(is_correct), now(), true)
on conflict (item_id) do update set is_correct = excluded.is_correct, locked = true;

-- name: PracticeAttemptSetIndex :one
update practice_attempts set current_index = sqlc.arg(current_index), tab_id = coalesce(sqlc.narg(tab_id), tab_id), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'IN_PROGRESS'
returning *;

-- name: PracticeAttemptTakeTab :exec
update practice_attempts set tab_id = sqlc.arg(tab_id) where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: PracticeAttemptClose :one
-- Đóng lượt (nộp hoặc hết giờ): chỉ khi còn IN_PROGRESS (idempotent: 0 dòng = đã đóng).
update practice_attempts
set status = sqlc.arg(status), submitted_at = now(), correct_count = sqlc.arg(correct_count), score = sqlc.arg(score), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'IN_PROGRESS'
returning *;

-- name: PracticeDue :many
-- Lượt MOCK quá hạn + dung sai (giây) chưa đóng; SKIP LOCKED để nhiều worker không đóng trùng.
select id, course_id from practice_attempts
where status = 'IN_PROGRESS' and mode = 'MOCK' and deadline_at + sqlc.arg(grace_seconds)::int * interval '1 second' < now()
order by deadline_at
limit sqlc.arg(max_rows)
for update skip locked;

-- name: PracticeHistory :many
select t.id, t.mode, t.topic, t.status, t.started_at, t.submitted_at, t.item_count, t.correct_count, t.score
from practice_attempts t
where t.course_id = sqlc.arg(course_id) and t.student_id = sqlc.arg(student_id)
  and (sqlc.narg(cursor_at)::timestamptz is null or (t.started_at, t.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by t.started_at desc, t.id desc
limit sqlc.arg(max_rows);

-- name: PracticeInProgress :many
select id, mode, topic, started_at, deadline_at, item_count, current_index
from practice_attempts
where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id) and status = 'IN_PROGRESS'
order by started_at desc;

-- name: PracticeLastTopicAt :one
-- Lần cuối sinh viên luyện (bất kỳ lượt) một chủ đề trong lớp; epoch nếu chưa. Dùng cho nguồn Hôm nay "ôn chủ đề" (chưa luyện trong 3 ngày).
select coalesce(max(started_at), 'epoch'::timestamptz)::timestamptz as last_at
from practice_attempts
where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id) and topic = sqlc.arg(topic);

-- name: PracticeDocTitles :many
-- Tài liệu gợi ý ôn: chỉ tài liệu sinh viên được thấy (READY, hiện cho sinh viên, không phải đề / đáp án / quy chế).
select id, title from documents
where course_id = sqlc.arg(course_id) and id = any(sqlc.arg(ids)::text[]::uuid[]) and status = 'READY' and visible_to_students and type::text in ('LECTURE', 'OTHER');
