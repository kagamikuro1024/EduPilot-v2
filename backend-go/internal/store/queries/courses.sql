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

-- ===== Admin mở lớp, gán giảng viên / TA, lưu trữ (US-P2-08) =====

-- name: InsertCourse :one
-- Trùng join_code ⇒ không dòng (người gọi sinh mã khác, tối đa 5 lần); trùng class_code ⇒ 23505 courses_class_code_key.
insert into courses (subject_code, class_code, name, semester, capacity, join_code, created_by)
values (sqlc.arg(subject_code), sqlc.arg(class_code), sqlc.arg(name), sqlc.arg(semester), sqlc.narg(capacity), sqlc.arg(join_code), sqlc.arg(created_by))
on conflict (join_code) do nothing
returning *;

-- name: LockCourse :one
select * from courses where id = sqlc.arg(id) for update;

-- name: GetCourse :one
-- Đọc đầy đủ một lớp (worker, không khoá).
select * from courses where id = sqlc.arg(id);

-- name: UpdateCourse :one
-- Khoá lạc quan theo version; không dòng ⇒ sai version (người gọi đã khoá dòng nên đọc lại bản hiện hành).
update courses
set subject_code = coalesce(sqlc.narg(subject_code), subject_code),
    class_code = coalesce(sqlc.narg(class_code), class_code),
    name = coalesce(sqlc.narg(name), name),
    semester = coalesce(sqlc.narg(semester), semester),
    capacity = case when sqlc.arg(set_capacity)::bool then sqlc.narg(capacity)::int else capacity end,
    version = version + 1
where id = sqlc.arg(id) and version = sqlc.arg(version)
returning *;

-- name: ArchiveCourse :one
-- Lưu trữ: tắt mã tham gia NGAY; thành viên giữ nguyên. Không dòng ⇒ đã lưu trữ (idempotent ở người gọi).
update courses
set status = 'ARCHIVED', archived_at = now(), join_enabled = false, version = version + 1
where id = sqlc.arg(id) and status = 'ACTIVE'
returning *;

-- name: ListAdminCourses :many
-- MỘT truy vấn tổng hợp cho cả trang (không N+1). Không có join_code, không có sinh viên (Admin không đọc nội dung lớp).
select c.id, c.class_code, c.subject_code, c.name, c.semester, c.status, c.capacity, c.version, c.created_at,
       t.user_id as teacher_id, tu.full_name as teacher_name,
       (select count(*) from enrollments e where e.course_id = c.id and e.role_in_course = 'TA' and e.status = 'ACTIVE')::int as assistants_count,
       (select count(*) from enrollments e where e.course_id = c.id and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE')::int as students_active,
       (select count(*) from enrollments e where e.course_id = c.id and e.role_in_course = 'STUDENT' and e.status = 'PENDING')::int as students_pending
from courses c
left join enrollments t on t.course_id = c.id and t.role_in_course = 'TEACHER' and t.status = 'ACTIVE'
left join users tu on tu.id = t.user_id
where (sqlc.narg(id)::uuid is null or c.id = sqlc.narg(id)::uuid)
  and (sqlc.narg(status)::course_status is null or c.status = sqlc.narg(status)::course_status)
  and (sqlc.narg(semester)::text is null or c.semester = sqlc.narg(semester)::text)
  and (sqlc.narg(name_like)::text is null
       or vn_fold(c.name) like sqlc.narg(name_like)::text escape '\'
       or lower(c.class_code) like sqlc.narg(code_like)::text escape '\')
  and (sqlc.narg(cur_at)::timestamptz is null or (c.created_at, c.id) < (sqlc.narg(cur_at)::timestamptz, sqlc.narg(cur_id)::uuid))
order by c.created_at desc, c.id desc
limit sqlc.arg(lim);

-- name: ListStaffEnrollments :many
-- Giảng viên và TA ACTIVE của lớp (để so khác biệt khi gán).
select e.user_id, e.role_in_course, u.full_name
from enrollments e join users u on u.id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.role_in_course in ('TEACHER', 'TA') and e.status = 'ACTIVE'
order by u.full_name, e.user_id;

-- name: ActivateStaffEnrollment :one
-- Gán giảng viên / TA. Người từng bị gỡ (REMOVED) ⇒ DÙNG LẠI dòng cũ (SRS 4.4). joined_via=ADMIN.
insert into enrollments (course_id, user_id, role_in_course, status, joined_via, status_changed_by)
values (sqlc.arg(course_id), sqlc.arg(user_id), sqlc.arg(role_in_course), 'ACTIVE', 'ADMIN', sqlc.narg(actor))
on conflict (course_id, user_id) do update
set role_in_course = excluded.role_in_course, status = 'ACTIVE', joined_via = 'ADMIN',
    previous_status = enrollments.status, status_changed_at = now(), status_changed_by = excluded.status_changed_by,
    removed_at = null, version = enrollments.version + 1
returning id;

-- name: RemoveStaffEnrollment :exec
-- Gỡ giảng viên / TA: mất quyền ở yêu cầu kế tiếp (guard không cache).
update enrollments
set previous_status = status, status = 'REMOVED', status_changed_at = now(), status_changed_by = sqlc.narg(actor), removed_at = now(), version = version + 1
where course_id = sqlc.arg(course_id) and user_id = sqlc.arg(user_id) and role_in_course in ('TEACHER', 'TA') and status <> 'REMOVED';

-- name: GetUsersForAssign :many
select id, full_name, role, status from users where id = any(sqlc.arg(ids)::text[]::uuid[]); -- ids là text[]: pgx ở chế độ simple protocol (PgBouncer) không mã hoá được []uuid.UUID

-- name: ListAssistantCandidates :many
select id, full_name, email
from users
where role = 'TA' and status in ('ACTIVE', 'INVITED')
  and (sqlc.narg(name_like)::text is null
       or vn_fold(full_name) like sqlc.narg(name_like)::text escape '\'
       or email like sqlc.narg(email_like)::text escape '\')
order by full_name, id
limit 20;

-- name: CountActiveStudents :one
select count(*)::int from enrollments where course_id = sqlc.arg(course_id) and role_in_course = 'STUDENT' and status = 'ACTIVE';

-- ===== Thông báo (US-P2-08) =====

-- name: InsertNotification :execrows
-- Idempotent theo (user_id, dedupe_key): giao lại cùng tin outbox không tạo dòng thứ hai.
insert into notifications (user_id, course_id, type, title, body, link, dedupe_key)
values (sqlc.arg(user_id), sqlc.narg(course_id), sqlc.arg(type), sqlc.arg(title), sqlc.narg(body), sqlc.narg(link), sqlc.narg(dedupe_key))
on conflict (user_id, dedupe_key) where dedupe_key is not null do nothing;

-- name: ListNotifications :many
select id, type, title, body, link, course_id, read_at, created_at
from notifications
where user_id = sqlc.arg(user_id)
  and (not sqlc.arg(unread_only)::bool or read_at is null)
  and (sqlc.narg(cur_at)::timestamptz is null or (created_at, id) < (sqlc.narg(cur_at)::timestamptz, sqlc.narg(cur_id)::uuid))
order by created_at desc, id desc
limit sqlc.arg(lim);

-- name: CountUnreadNotifications :one
select count(*)::int from notifications where user_id = sqlc.arg(user_id) and read_at is null;

-- name: MarkNotificationRead :execrows
-- Chỉ thông báo CỦA MÌNH (người khác ⇒ 0 dòng ⇒ 404). Đã đọc rồi vẫn tính là thành công (idempotent): giữ read_at đầu tiên.
update notifications set read_at = coalesce(read_at, now()) where id = sqlc.arg(id) and user_id = sqlc.arg(user_id);

-- ===== Vào lớp bằng mã, cài đặt tham gia, thành viên (US-P2-09) =====

-- name: GetCourseByJoinCode :one
-- Bước tra mã DUY NHẤT của preview và join (luôn đúng một truy vấn ở mọi nhánh thất bại).
select * from courses where join_code = sqlc.arg(join_code);

-- name: LockEnrollment :one
select * from enrollments where course_id = sqlc.arg(course_id) and user_id = sqlc.arg(user_id) for update;

-- name: EnrollmentConflictByStudentCode :one
-- (Danh sách trắng SRS FEAT-account-security 4.2.5.) MSSV tự khai của người vào bằng mã đã là ảnh chụp của người KHÁC đang ở / chờ vào lớp? (không bao giờ dùng để nối tài khoản)
select exists (
    select 1 from enrollments
    where course_id = sqlc.arg(course_id) and student_code_snapshot = sqlc.arg(code) and user_id <> sqlc.arg(user_id) and status in ('ACTIVE', 'PENDING')
) as held;

-- name: InsertCodeEnrollment :one
insert into enrollments (course_id, user_id, role_in_course, status, joined_via, student_code_snapshot, warning, status_changed_at)
values (sqlc.arg(course_id), sqlc.arg(user_id), 'STUDENT', sqlc.arg(status), 'CODE', sqlc.narg(snapshot), sqlc.narg(warning), sqlc.arg(at))
returning *;

-- name: RejoinEnrollment :one
-- Vào lại sau khi bị mời ra: CÙNG dòng, REMOVED → PENDING (mặc định an toàn, Q5); snapshot cũ giữ nguyên.
update enrollments
set status = 'PENDING', previous_status = 'REMOVED', warning = sqlc.narg(warning), status_changed_at = sqlc.arg(at), status_changed_by = null, removed_at = null, version = version + 1
where id = sqlc.arg(id) and status = 'REMOVED'
returning *;

-- name: SetEnrollmentStatus :one
-- Đổi trạng thái một ghi danh (duyệt / từ chối / mời ra / hoàn tác). `previous` = trạng thái ghi vào previous_status (null = không hoàn tác được nữa).
update enrollments
set status = sqlc.arg(status)::enrollment_status,
    previous_status = sqlc.narg(previous)::enrollment_status,
    status_changed_at = sqlc.arg(at),
    status_changed_by = sqlc.narg(actor),
    removed_at = case when sqlc.arg(status)::enrollment_status = 'REMOVED' then sqlc.arg(at)::timestamptz else null end,
    version = version + 1
where id = sqlc.arg(id)
returning *;

-- name: UpdateJoinSettings :one
update courses
set join_enabled = coalesce(sqlc.narg(enabled), join_enabled),
    join_require_approval = coalesce(sqlc.narg(require_approval), join_require_approval),
    join_expires_at = case when sqlc.arg(set_expires)::bool then sqlc.narg(expires_at)::timestamptz else join_expires_at end,
    allowed_email_domain = case when sqlc.arg(set_domain)::bool then sqlc.narg(domain)::text else allowed_email_domain end,
    capacity = case when sqlc.arg(set_capacity)::bool then sqlc.narg(capacity)::int else capacity end,
    version = version + 1
where id = sqlc.arg(id) and version = sqlc.arg(version)
returning *;

-- name: RegenerateJoinCode :one
-- Thay mã trong MỘT câu UPDATE (mã cũ chết ngay). Trùng mã mới ⇒ không dòng (người gọi thử mã khác, tối đa 5 lần).
update courses
set join_code = sqlc.arg(join_code), version = version + 1
where courses.id = sqlc.arg(id) and courses.status = 'ACTIVE' and not exists (select 1 from courses o where o.join_code = sqlc.arg(join_code))
returning *;

-- name: ListMembers :many
-- Một truy vấn: số đếm (toàn lớp, không theo bộ lọc) + một trang thành viên. Trang rỗng vẫn trả đúng một dòng mang số đếm.
with cnt as (
    select count(*) filter (where c.status = 'ACTIVE' and c.role_in_course = 'STUDENT')::int as active,
           count(*) filter (where c.status = 'PENDING' and c.role_in_course = 'STUDENT')::int as pending
    from enrollments c where c.course_id = sqlc.arg(course_id)
), page as (
    select e.user_id, u.full_name, u.email, e.student_code_snapshot, e.role_in_course, e.status, e.joined_via, e.warning, e.status_changed_at
    from enrollments e join users u on u.id = e.user_id
    where e.course_id = sqlc.arg(course_id)
      and (case when sqlc.narg(status)::enrollment_status is null then e.status in ('ACTIVE', 'PENDING') else e.status = sqlc.narg(status)::enrollment_status end)
      and (sqlc.narg(role)::enrollment_role is null or e.role_in_course = sqlc.narg(role)::enrollment_role)
      and (sqlc.narg(name_like)::text is null
           or vn_fold(u.full_name) like sqlc.narg(name_like)::text escape '\'
           or e.student_code_snapshot like sqlc.narg(code_like)::text escape '\'
           or u.email like sqlc.narg(email_like)::text escape '\')
      and (sqlc.narg(cur_at)::timestamptz is null
           or e.status_changed_at < sqlc.narg(cur_at)::timestamptz
           or (e.status_changed_at = sqlc.narg(cur_at)::timestamptz and e.user_id > sqlc.narg(cur_id)::uuid))
    order by e.status_changed_at desc, e.user_id
    limit sqlc.arg(lim)
)
select cnt.active, cnt.pending, page.user_id, page.full_name, page.email, page.student_code_snapshot, page.role_in_course, page.status, page.joined_via, page.warning, page.status_changed_at
from cnt left join page on true
order by page.status_changed_at desc nulls last, page.user_id;
