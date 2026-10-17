-- Thời gian học (FEAT-crm-360 US-P5-03). Heartbeat KHÔNG chạm Postgres; chỉ rollup (mỗi phút) và các route đọc dùng các truy vấn này.

-- name: ActivityUserTracking :one
-- Ack minh bạch + cờ đo toàn tài khoản của MỘT người (đệm Redis 60 s ở tầng gọi). Chưa có user_settings = mặc định bật.
select u.tracking_notice_ack_at, coalesce(us.track_study_time, true)::bool as track_study_time
from users u left join user_settings us on us.user_id = u.id
where u.id = sqlc.arg(user_id);

-- name: ActivityEnrollment :one
-- Sinh viên có ACTIVE ở lớp không + cờ "cho tắt" của lớp (đệm Redis 60 s ở tầng gọi).
select e.role_in_course::text as role_in_course, e.status::text as enrollment_status, c.allow_study_time_opt_out
from enrollments e join courses c on c.id = e.course_id
where e.course_id = sqlc.arg(course_id) and e.user_id = sqlc.arg(user_id);

-- name: ActivityAckTracking :one
-- Đặt lần đầu, giữ lần đầu (idempotent).
update users set tracking_notice_ack_at = coalesce(tracking_notice_ack_at, now()) where id = sqlc.arg(id) returning tracking_notice_ack_at;

-- name: ActivityUpsertStudyTime :exec
-- Rollup: ghi TUYỆT ĐỐI, GREATEST với giá trị cũ (chạy lại / giao đôi không đổi kết quả). Danh sách song song, ≤ 500 hàng một câu.
insert into study_time_daily (course_id, student_id, date, area, active_seconds)
select c.cid::uuid, s.sid::uuid, d.dt::date, a.ar, v.secs::int
from unnest(sqlc.arg(course_ids)::text[]) with ordinality as c(cid, n)
join unnest(sqlc.arg(student_ids)::text[]) with ordinality as s(sid, n) on s.n = c.n
join unnest(sqlc.arg(dates)::text[]) with ordinality as d(dt, n) on d.n = c.n
join unnest(sqlc.arg(areas)::text[]) with ordinality as a(ar, n) on a.n = c.n
join unnest(sqlc.arg(seconds)::int[]) with ordinality as v(secs, n) on v.n = c.n
on conflict (course_id, student_id, date, area) do update set active_seconds = greatest(study_time_daily.active_seconds, excluded.active_seconds);

-- name: ActivityEnrolledPairs :many
-- Cặp (lớp, sinh viên) còn tồn tại trong enrollments: rollup bỏ qua khoá Redis của cặp đã bị xoá (khoá ngoại sẽ làm cả lô lỗi).
select e.course_id, e.user_id
from enrollments e
join unnest(sqlc.arg(course_ids)::text[]) with ordinality as c(cid, n) on c.cid::uuid = e.course_id
join unnest(sqlc.arg(student_ids)::text[]) with ordinality as s(sid, n) on s.n = c.n and s.sid::uuid = e.user_id;

-- name: ActivityStudyTimeWeeks :many
-- Tổng giây theo (tuần ISO bắt đầu thứ Hai, khu vực) của MỘT sinh viên ở MỘT lớp, từ ngày `from_date`.
select date_trunc('week', date)::date as week_start, area, sum(active_seconds)::bigint as seconds
from study_time_daily
where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id) and date >= sqlc.arg(from_date)::date
group by 1, 2
order by 1, 2;

-- name: ActivitySettingsCourses :many
-- Lớp sinh viên đang ACTIVE (một dòng mỗi lớp) kèm cờ khoá: cho nhóm "Thời gian học" ở /settings.
select c.id as course_id, c.class_code, c.name, (not c.allow_study_time_opt_out)::bool as locked
from enrollments e join courses c on c.id = e.course_id
where e.user_id = sqlc.arg(user_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE' and c.status = 'ACTIVE'
order by c.class_code;
