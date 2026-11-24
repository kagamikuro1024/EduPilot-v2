-- Thống kê điểm thi theo lớp / tuần (US-P10-06): CHỈ bài PUBLISHED, lượt GRADED, điểm chính thức = adjusted_score nếu có, không thì auto_score.

-- name: ExamAnalyticsExams :many
select e.id, e.title, coalesce(e.opens_at, e.created_at)::timestamptz as opens_at, e.max_score
from exams e
where e.course_id = sqlc.arg(course_id) and e.status = 'PUBLISHED'
order by coalesce(e.opens_at, e.created_at), e.id;

-- name: ExamAnalyticsScores :many
select a.exam_id, a.student_id, coalesce(a.adjusted_score, a.auto_score)::numeric as score
from exam_attempts a
join exams e on e.id = a.exam_id and e.course_id = a.course_id and e.status = 'PUBLISHED'
where a.course_id = sqlc.arg(course_id) and a.status = 'GRADED' and coalesce(a.adjusted_score, a.auto_score) is not null
  -- exam_ids: các uuid cách nhau dấu phẩy (null = mọi bài PUBLISHED); không dùng tham số mảng vì chế độ giao thức đơn giản (PgBouncer) không mã hoá được.
  and (sqlc.narg(exam_ids)::text is null or a.exam_id = any (string_to_array(sqlc.narg(exam_ids)::text, ',')::uuid[]));

-- name: ExamAnalyticsItems :many
select i.id as item_id, i.position, q.type, q.title, coalesce(q.topic, '')::text as topic
from exam_items i
join question_bank q on q.course_id = i.course_id and q.id = i.question_id
where i.course_id = sqlc.arg(course_id) and i.exam_id = sqlc.arg(exam_id)
order by i.position;

-- name: ExamAnalyticsBreakdowns :many
select a.breakdown
from exam_attempts a
join exams e on e.id = a.exam_id and e.course_id = a.course_id and e.status = 'PUBLISHED'
where a.course_id = sqlc.arg(course_id) and a.exam_id = sqlc.arg(exam_id) and a.status = 'GRADED';

-- name: ExamAnalyticsDeclining :many
-- Sinh viên có điểm GIẢM NGHIÊM NGẶT ở 3 bài PUBLISHED LIÊN TIẾP của lớp (bài vắng xen giữa làm đứt chuỗi); mỗi người một dòng = bộ ba mới nhất.
with ex as (
  select id, title, row_number() over (order by coalesce(opens_at, created_at), id) as rn
  from exams where course_id = sqlc.arg(course_id) and status = 'PUBLISHED'
), sc as (
  select a.student_id, ex.rn, coalesce(a.adjusted_score, a.auto_score) as score
  from exam_attempts a join ex on ex.id = a.exam_id
  where a.course_id = sqlc.arg(course_id) and a.status = 'GRADED' and coalesce(a.adjusted_score, a.auto_score) is not null
), t as (
  select student_id, rn, score,
         lag(score, 1) over w as s1, lag(score, 2) over w as s2, lag(rn, 1) over w as r1, lag(rn, 2) over w as r2
  from sc window w as (partition by student_id order by rn)
), hit as (
  select distinct on (student_id) student_id, rn, s2, s1, score
  from t where r1 = rn - 1 and r2 = rn - 2 and s1 > score and s2 > s1
  order by student_id, rn desc
)
select h.student_id, u.full_name, en.student_code_snapshot as student_code,
       h.s2::numeric as score1, h.s1::numeric as score2, h.score::numeric as score3,
       e1.id as exam1_id, e1.title as exam1_title, e2.id as exam2_id, e2.title as exam2_title, e3.id as exam3_id, e3.title as exam3_title
from hit h
join ex e1 on e1.rn = h.rn - 2 join ex e2 on e2.rn = h.rn - 1 join ex e3 on e3.rn = h.rn
join users u on u.id = h.student_id
join enrollments en on en.course_id = sqlc.arg(course_id) and en.user_id = h.student_id and en.role_in_course = 'STUDENT' and en.status = 'ACTIVE'
where (sqlc.narg(cursor_name)::text is null or (u.full_name, h.student_id) > (sqlc.narg(cursor_name)::text, sqlc.narg(cursor_id)::uuid))
order by u.full_name, h.student_id
limit sqlc.arg(max_rows);

-- name: ExamAnalyticsTeacherCourses :many
select c.id, c.class_code, c.name
from enrollments en join courses c on c.id = en.course_id
where en.user_id = sqlc.arg(user_id) and en.role_in_course = 'TEACHER' and en.status = 'ACTIVE'
order by c.class_code, c.id;
