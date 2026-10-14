-- At-risk và danh sách sinh viên (FEAT-crm-360 US-P5-04). Chỉ gói internal/crm gọi các hàm sinh từ tệp này (student_risk là dữ liệu nhạy cảm: sinh viên không bao giờ thấy).
-- Tuần ISO thứ Hai–Chủ nhật, Asia/Ho_Chi_Minh: `from_day` = thứ Hai của tuần W4 (cũ nhất), chỉ số tuần k = (ngày ICT − from_day) / 7 ∈ 0..3 (3 = W1, tuần trọn gần nhất).

-- name: CrmRiskActiveCourses :many
select id from courses where status = 'ACTIVE' order by id;

-- name: CrmRiskCourse :one
select settings, allow_study_time_opt_out from courses where id = sqlc.arg(course_id);

-- name: CrmRiskStudents :many
-- Sinh viên ACTIVE + điều kiện có dữ liệu thời gian học: đã xác nhận thông báo VÀ (lớp không cho tắt đo HOẶC sinh viên chưa tắt).
select e.user_id as student_id, u.tracking_notice_ack_at as ack_at, coalesce(us.track_study_time, true)::bool as track_study_time
from enrollments e
join users u on u.id = e.user_id
left join user_settings us on us.user_id = e.user_id
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE';

-- name: CrmRiskAttendance :many
select a.student_id, (((s.starts_at at time zone 'Asia/Ho_Chi_Minh')::date - sqlc.arg(from_day)::date) / 7)::int as k,
       count(*)::int as recorded,
       (count(*) filter (where a.status <> 'ABSENT'))::int as ok,
       (count(*) filter (where a.status = 'ABSENT'))::int as absent
from attendance_records a
join class_sessions s on s.course_id = a.course_id and s.id = a.session_id
where a.course_id = sqlc.arg(course_id)
  and s.starts_at >= sqlc.arg(from_at)::timestamptz and s.starts_at < sqlc.arg(to_at)::timestamptz
group by a.student_id, k;

-- name: CrmRiskActivity :many
-- hidden-ok: hoạt động của sinh viên tính cả bài đang bị ẩn kiểm duyệt (sinh viên vẫn đã làm việc đó); chỉ loại bài đã xoá (SRS crm-360 4.7.1)
-- asks = tin USER ở phiên chat riêng chưa xoá mềm; posts = thread + bình luận HUMAN chưa xoá. Đếm theo created_at (không theo khoảng id).
select x.student_id, x.k, (sum(x.asks))::int as asks, (sum(x.posts))::int as posts
from (
    select m.user_id as student_id, (((m.created_at at time zone 'Asia/Ho_Chi_Minh')::date - sqlc.arg(from_day)::date) / 7)::int as k, 1 as asks, 0 as posts
    from chat_messages m join chat_sessions cs on cs.course_id = m.course_id and cs.id = m.session_id
    where m.course_id = sqlc.arg(course_id) and m.role = 'USER' and cs.deleted_at is null
      and m.created_at >= sqlc.arg(from_at)::timestamptz and m.created_at < sqlc.arg(to_at)::timestamptz
    union all
    select t.author_id, (((t.created_at at time zone 'Asia/Ho_Chi_Minh')::date - sqlc.arg(from_day)::date) / 7)::int, 0, 1
    from forum_threads t
    where t.course_id = sqlc.arg(course_id) and t.deleted_at is null
      and t.created_at >= sqlc.arg(from_at)::timestamptz and t.created_at < sqlc.arg(to_at)::timestamptz
    union all
    select p.author_id, (((p.created_at at time zone 'Asia/Ho_Chi_Minh')::date - sqlc.arg(from_day)::date) / 7)::int, 0, 1
    from forum_posts p
    where p.course_id = sqlc.arg(course_id) and p.kind = 'HUMAN' and p.deleted_at is null and p.author_id is not null
      and p.created_at >= sqlc.arg(from_at)::timestamptz and p.created_at < sqlc.arg(to_at)::timestamptz
) x
group by x.student_id, x.k;

-- name: CrmRiskStudy :many
select student_id, (((date - sqlc.arg(from_day)::date) / 7))::int as k, (sum(active_seconds))::int as seconds
from study_time_daily
where course_id = sqlc.arg(course_id) and date >= sqlc.arg(from_day)::date and date < sqlc.arg(to_day)::date
group by student_id, k;

-- name: CrmRiskUpsert :exec
-- Một câu cho cả lớp. `entered_at` giữ nguyên khi đêm trước đã AT_RISK; đặt now() khi vừa vào; NULL khi rời (ràng buộc student_risk_entered_chk).
insert into student_risk (course_id, student_id, level, reasons, weeks, last_week_asks, last_week_posts, entered_at, computed_at)
select sqlc.arg(course_id)::uuid, x.student_id, x.level::risk_level, x.reasons, x.weeks, x.asks, x.posts,
       case when x.level = 'AT_RISK' then sqlc.arg(now_at)::timestamptz end, sqlc.arg(now_at)::timestamptz
from jsonb_to_recordset(sqlc.arg(rows)::jsonb) as x(student_id uuid, level text, reasons jsonb, weeks jsonb, asks int, posts int)
on conflict (course_id, student_id) do update set
    level = excluded.level, reasons = excluded.reasons, weeks = excluded.weeks,
    last_week_asks = excluded.last_week_asks, last_week_posts = excluded.last_week_posts, computed_at = excluded.computed_at,
    entered_at = case when student_risk.level = 'AT_RISK' and excluded.level = 'AT_RISK' then student_risk.entered_at else excluded.entered_at end;

-- name: CrmRiskDeleteStale :exec
-- Sinh viên không còn ACTIVE (REMOVED / PENDING) không có dòng.
delete from student_risk r
where r.course_id = sqlc.arg(course_id)
  and not exists (select 1 from enrollments e where e.course_id = r.course_id and e.user_id = r.student_id and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE');

-- name: CrmStudentsList :many
-- Danh sách (AC1/AC2): v_student_360 (không chạm chat_messages) + cờ `tracked`. Khoá con trỏ (risk_rank giảm, vn_fold(tên), id).
with base as (
    select v.student_id, v.full_name, v.student_code, v.recorded, v.present, v.late, v.excused, v.absent, v.speak_count, v.points_total,
           v.study_seconds_7d, v.last_week_asks, v.last_week_posts, v.risk_level, v.risk_computed_at,
           (case v.risk_level when 'AT_RISK' then 2 when 'WATCH' then 1 else 0 end)::int as risk_rank,
           vn_fold(v.full_name)::text as name_fold,
           (u.tracking_notice_ack_at is not null and (not c.allow_study_time_opt_out or coalesce(us.track_study_time, true)))::bool as tracked
    from v_student_360 v
    join users u on u.id = v.student_id
    join courses c on c.id = v.course_id
    left join user_settings us on us.user_id = v.student_id
    where v.course_id = sqlc.arg(course_id)
      and (not sqlc.arg(f_attention)::bool or v.risk_level = 'AT_RISK')
      and (not sqlc.arg(f_absent)::bool or v.absent >= sqlc.arg(absent_alert)::int)
      and (not sqlc.arg(f_inactive)::bool or (v.last_week_asks + v.last_week_posts = 0 and v.study_seconds_7d < sqlc.arg(study_min_minutes)::int * 60))
      and (sqlc.arg(q)::text = '' or strpos(vn_fold(v.full_name), vn_fold(sqlc.arg(q)::text)) > 0 or starts_with(v.student_code, sqlc.arg(q)::text))
)
select * from base
where (sqlc.narg(cur_rank)::int is null
       or risk_rank < sqlc.narg(cur_rank)::int
       or (risk_rank = sqlc.narg(cur_rank)::int and (name_fold > sqlc.narg(cur_fold)::text or (name_fold = sqlc.narg(cur_fold)::text and student_id > sqlc.narg(cur_id)::uuid))))
order by risk_rank desc, name_fold, student_id
limit sqlc.arg(lim);

-- name: CrmStudentsCount :one
select count(*)::int
from v_student_360 v
where v.course_id = sqlc.arg(course_id)
  and (not sqlc.arg(f_attention)::bool or v.risk_level = 'AT_RISK')
  and (not sqlc.arg(f_absent)::bool or v.absent >= sqlc.arg(absent_alert)::int)
  and (not sqlc.arg(f_inactive)::bool or (v.last_week_asks + v.last_week_posts = 0 and v.study_seconds_7d < sqlc.arg(study_min_minutes)::int * 60))
  and (sqlc.arg(q)::text = '' or strpos(vn_fold(v.full_name), vn_fold(sqlc.arg(q)::text)) > 0 or starts_with(v.student_code, sqlc.arg(q)::text));

-- name: CrmAttention :many
-- "Hôm nay" của Staff (US-P5-04 AC7): MỘT truy vấn cho cả hai nhu cầu. kind 'N' = số sinh viên mới vào AT_RISK từ `since` ở mỗi lớp (mục STUDENT_ATTENTION);
-- kind 'S' = tối đa `lim` sinh viên AT_RISK, mới vào trước, kèm tổng (khối `attention`).
with sel as (
    select course_id, student_id, entered_at, reasons from student_risk
    where course_id = any(sqlc.arg(course_ids)::text[]::uuid[]) and level = 'AT_RISK'
), top as (
    select s.*, (count(*) over ())::int as total, (row_number() over (order by s.entered_at desc, s.student_id))::int as rk from sel s order by s.entered_at desc, s.student_id limit sqlc.arg(lim)
)
select 'N'::text as kind, n.course_id::uuid as course_id, ''::text as class_code, '00000000-0000-0000-0000-000000000000'::uuid as student_id, ''::text as full_name, '[]'::jsonb as reasons, n.n::int as n, 0::int as total, 0::int as rk
from (select course_id, count(*) as n from sel where entered_at >= sqlc.arg(since)::timestamptz group by course_id) n
union all
select 'S'::text, t.course_id::uuid, c.class_code::text, t.student_id::uuid, u.full_name::text, t.reasons::jsonb, 0::int, t.total::int, t.rk
from top t join courses c on c.id = t.course_id join users u on u.id = t.student_id
order by kind, rk;
