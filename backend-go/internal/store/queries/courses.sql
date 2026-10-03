-- Lớp học: tra quyền và đọc (FEAT-course-foundation US-P2-07). Mọi truy vấn của bảng thuộc lớp bắt đầu bằng course_id (luật 13).

-- name: GetMembership :one
-- CourseAccessGuard: MỘT truy vấn theo chỉ mục (course_id, user_id), không cache.
select role_in_course, status from enrollments where course_id = sqlc.arg(course_id) and user_id = sqlc.arg(user_id);

-- name: ListMyCourses :many
-- Sinh viên: lớp ACTIVE và PENDING của chính mình; Giảng viên / TA: lớp được phân công (ACTIVE). Một truy vấn, không N+1.
select c.id, c.class_code, c.subject_code, c.name, c.semester, c.status as course_status,
       e.role_in_course, e.status as enrollment_status, e.created_at as enrolled_at, e.id as enrollment_id
from enrollments e
join courses c on c.id = e.course_id
where e.user_id = sqlc.arg(user_id)
  and (e.status = 'ACTIVE' or (e.status = 'PENDING' and e.role_in_course = 'STUDENT'))
  and (sqlc.narg(cur_at)::timestamptz is null or (e.created_at, e.id) < (sqlc.narg(cur_at)::timestamptz, sqlc.narg(cur_id)::uuid))
order by e.created_at desc, e.id desc
limit sqlc.arg(lim);

-- name: GetCourseBasic :one
select id, class_code, subject_code, name, semester, status from courses where id = sqlc.arg(id);

-- name: ListCourseTeacherNames :many
select u.full_name
from enrollments e
join users u on u.id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'TEACHER' and e.status = 'ACTIVE'
order by u.full_name;

-- name: CountCourseStudents :one
select count(*) filter (where status = 'ACTIVE')::int as active, count(*) filter (where status = 'PENDING')::int as pending
from enrollments
where course_id = sqlc.arg(course_id) and role_in_course = 'STUDENT';

-- name: ChunksForCourse :many
-- Đường DUY NHẤT đọc content_chunks ở P2: luôn lọc theo lớp (course_ids, GIN). Không có hàm đọc chunk "không lọc lớp".
select id, document_id, text, audience
from content_chunks
where course_ids @> array[sqlc.arg(course_id)::uuid]
order by id
limit sqlc.arg(lim);
