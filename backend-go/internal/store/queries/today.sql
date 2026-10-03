-- "Hôm nay" (US-P2-11, SRS 4.7). Mỗi truy vấn là TỔNG HỢP cho cả tập lớp của người xem (không N+1).

-- name: TodayViewer :many
-- Người xem và ghi danh ACTIVE / PENDING của chính họ trong lớp còn mở: MỘT truy vấn dựng Viewer (SRS 4.7 "ngân sách truy vấn").
-- Người chưa có lớp ⇒ đúng một dòng với course_id NULL.
select u.email, (u.email_verified_at is not null)::boolean as verified,
       e.course_id, c.class_code, e.role_in_course, e.status, e.status_changed_at
from users u
left join enrollments e on e.user_id = u.id and e.status in ('ACTIVE', 'PENDING')
left join courses c on c.id = e.course_id and c.status = 'ACTIVE'
where u.id = sqlc.arg(user_id);

-- name: TodayStaffPending :many
-- (epoch = không có.) Mỗi lớp: yêu cầu vào lớp chờ duyệt (không tính chờ xác minh email của roster), cũ nhất, và số hàng email chưa khớp MSSV.
select e.course_id,
       (count(*) filter (where e.warning is null))::int as pending,
       coalesce(min(e.status_changed_at) filter (where e.warning is null), 'epoch'::timestamptz)::timestamptz as oldest,
       (count(*) filter (where e.warning = 'EMAIL_MISMATCH'))::int as mismatch,
       coalesce(min(e.status_changed_at) filter (where e.warning = 'EMAIL_MISMATCH'), 'epoch'::timestamptz)::timestamptz as mismatch_oldest
from enrollments e
where e.course_id = any(sqlc.arg(course_ids)::text[]::uuid[]) and e.role_in_course = 'STUDENT' and e.status = 'PENDING'
group by e.course_id;

-- name: TodaySetup :many
-- Bốn bước "Thiết lập lớp mới" tự tick theo dữ liệu (SRS 4.7); dismissed = mốc giảng viên bỏ qua.
select c.id as course_id,
       (exists (select 1 from enrollments e where e.course_id = c.id and e.role_in_course = 'STUDENT' and e.status in ('ACTIVE', 'PENDING')))::boolean as share_code,
       (exists (select 1 from documents d where d.course_id = c.id and d.type = 'COURSE_POLICY' and d.status <> 'FAILED'))::boolean as policy,
       (exists (select 1 from class_sessions s where s.course_id = c.id))::boolean as sessions,
       ((exists (select 1 from documents d where d.course_id = c.id and d.type <> 'COURSE_POLICY' and d.status <> 'FAILED')
        or exists (select 1 from document_courses dc join documents d on d.id = dc.document_id where dc.course_id = c.id and d.status <> 'FAILED')))::boolean as documents,
       coalesce(c.settings #>> '{setup,dismissed_at}', '')::text as dismissed
from courses c
where c.id = any(sqlc.arg(course_ids)::text[]::uuid[]);

-- name: TodaySessions :many
-- Buổi học của các lớp trong khoảng [from, to) (chưa kết thúc trước from), cũ → mới.
select s.course_id, c.class_code, c.name as course_name, s.session_no, s.starts_at, s.ends_at, s.room
from class_sessions s
join courses c on c.id = s.course_id
where s.course_id = any(sqlc.arg(course_ids)::text[]::uuid[]) and s.ends_at > sqlc.arg(from_at)::timestamptz and s.starts_at < sqlc.arg(to_at)::timestamptz
order by s.starts_at, s.id
limit 60;

-- name: TodayAdminProviders :many
-- Nhà cung cấp AI đang bật: lỗi nếu lần kiểm gần nhất thất bại (mạch mở do Scheduler báo thêm).
select id, name, coalesce(last_test_ok, true)::boolean as ok from llm_providers where enabled order by name;

-- name: TodayAdminCoursesNoTeacher :many
select c.id, c.class_code
from courses c
where c.status = 'ACTIVE'
  and not exists (select 1 from enrollments e where e.course_id = c.id and e.role_in_course = 'TEACHER' and e.status = 'ACTIVE')
order by c.class_code, c.id
limit 50;

-- name: TodayAdminExpiredInvites :one
-- Giảng viên / TA được mời, chưa dùng, mọi lời mời còn sống đã hết hạn.
select count(*)::int
from users u
where u.status = 'INVITED' and u.role in ('TEACHER', 'TA')
  and exists (select 1 from auth_tokens t where t.user_id = u.id and t.kind = 'INVITE' and t.used_at is null and t.revoked_at is null and t.expires_at < sqlc.arg(now)::timestamptz)
  and not exists (select 1 from auth_tokens t where t.user_id = u.id and t.kind = 'INVITE' and t.used_at is null and t.revoked_at is null and t.expires_at >= sqlc.arg(now)::timestamptz);

-- name: TodayCourseStaff :many
-- Người bị ảnh hưởng khi lớp đổi: giảng viên + TA đang hoạt động (xoá cache "Hôm nay").
select user_id from enrollments where course_id = sqlc.arg(course_id) and role_in_course in ('TEACHER', 'TA') and status = 'ACTIVE';

-- name: TodayAdminIDs :many
select id from users where role = 'ADMIN' and status = 'ACTIVE';

-- name: TodayUserCourses :many
select course_id from enrollments where user_id = sqlc.arg(user_id) and status in ('ACTIVE', 'PENDING');

-- name: DismissCourseSetup :execrows
-- Ghi mốc bỏ qua một lần; gọi lại không đổi mốc (idempotent). Số dòng = 1 khi VỪA ghi.
update courses
set settings = jsonb_set(settings, '{setup}', coalesce(settings -> 'setup', '{}'::jsonb) || jsonb_build_object('dismissed_at', sqlc.arg(at)::text), true)
where id = sqlc.arg(id) and (settings #>> '{setup,dismissed_at}') is null;

-- name: TodayCourseMembers :many
-- Mọi người đang ở / chờ trong lớp: bị ảnh hưởng khi lớp đổi hoặc lưu trữ.
select user_id from enrollments where course_id = sqlc.arg(course_id) and status in ('ACTIVE', 'PENDING');
