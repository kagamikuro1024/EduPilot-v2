-- Hồ sơ 360, ghi chú quan sát (FEAT-crm-360 US-P5-05). Dữ liệu nhạy cảm: chỉ internal/crm gọi; mọi truy vấn có course_id; không đọc chat_messages.

-- name: CrmProfileGet :one
-- Một sinh viên ACTIVE (v_student_360 chỉ chứa ACTIVE) + lý do rủi ro đã lưu + cờ `tracked` (như danh sách).
select v.student_id, v.full_name, v.student_code, v.joined_at, v.recorded, v.present, v.late, v.excused, v.absent, v.speak_count, v.points_total,
       v.study_seconds_7d, v.last_week_asks, v.last_week_posts, v.risk_level, v.risk_computed_at, coalesce(r.reasons, '[]'::jsonb)::jsonb as reasons,
       (u.tracking_notice_ack_at is not null and (not c.allow_study_time_opt_out or coalesce(us.track_study_time, true)))::bool as tracked
from v_student_360 v
join users u on u.id = v.student_id
join courses c on c.id = v.course_id
left join user_settings us on us.user_id = v.student_id
left join student_risk r on r.course_id = v.course_id and r.student_id = v.student_id
where v.course_id = sqlc.arg(course_id) and v.student_id = sqlc.arg(student_id);

-- name: CrmProfileStudyWeeks :many
-- Phút học theo tuần ISO (thứ Hai là đầu tuần) trong [from_day, to_day).
select (date_trunc('week', date::timestamp))::date as week_start, (sum(active_seconds) / 60)::int as minutes
from study_time_daily
where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id) and date >= sqlc.arg(from_day)::date and date < sqlc.arg(to_day)::date
group by 1 order by 1;

-- name: CrmStudentAttendance :many
-- Tab Chuyên cần: buổi × trạng thái × ghi chú (nhân viên được thấy `note`), buổi mới nhất trước.
select s.id as session_id, s.session_no, s.starts_at, a.status, a.note
from attendance_records a
join class_sessions s on s.course_id = a.course_id and s.id = a.session_id
where a.course_id = sqlc.arg(course_id) and a.student_id = sqlc.arg(student_id)
  and (sqlc.narg(cur_at)::timestamptz is null or (s.starts_at, s.id) < (sqlc.narg(cur_at)::timestamptz, sqlc.narg(cur_id)::uuid))
order by s.starts_at desc, s.id desc
limit sqlc.arg(lim);

-- name: CrmObsInsert :one
insert into student_observations (course_id, student_id, author_id, kind, tags, body, ai_edited)
values (sqlc.arg(course_id), sqlc.arg(student_id), sqlc.arg(author_id), sqlc.arg(kind), sqlc.arg(tags), sqlc.arg(body), sqlc.arg(ai_edited))
returning *;

-- name: CrmObsList :many
select o.*, u.full_name as author_name
from student_observations o join users u on u.id = o.author_id
where o.course_id = sqlc.arg(course_id) and o.student_id = sqlc.arg(student_id)
  and (sqlc.narg(cur_at)::timestamptz is null or (o.created_at, o.id) < (sqlc.narg(cur_at)::timestamptz, sqlc.narg(cur_id)::uuid))
order by o.created_at desc, o.id desc
limit sqlc.arg(lim);

-- name: CrmObsGet :one
select * from student_observations where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id) and id = sqlc.arg(id);

-- name: CrmObsUpdate :one
-- Chỉ tác giả (người gọi kiểm trước); sai version → không trả hàng.
update student_observations set tags = sqlc.arg(tags), body = sqlc.arg(body), ai_edited = case when kind = 'AI_SUMMARY' then true else false end, version = version + 1
where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id) and id = sqlc.arg(id) and author_id = sqlc.arg(author_id) and version = sqlc.arg(version)
returning *;

-- name: CrmObsDelete :execrows
delete from student_observations where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id) and id = sqlc.arg(id) and author_id = sqlc.arg(author_id);
