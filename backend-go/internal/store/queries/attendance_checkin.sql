-- Điểm danh tự phục vụ (FEAT-attendance-selfcheck, sprint 10.8). Mọi truy vấn có `course_id` (luật 13). Không toạ độ, không IP.

-- ---- cửa sổ ------------------------------------------------------------------------------------------------------------------------
-- name: AttWindowInsert :one
insert into checkin_windows (course_id, session_id, round, rotate_seconds, opened_by, opened_at, closes_at)
values (sqlc.arg(course_id), sqlc.arg(session_id), sqlc.arg(round), sqlc.arg(rotate_seconds), sqlc.arg(opened_by), sqlc.arg(opened_at), sqlc.arg(closes_at))
returning *;

-- name: AttWindowByID :one
select * from checkin_windows where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: AttWindowLock :one
-- Sau khi đã khoá buổi (cùng thứ tự khoá với PUT …/attendance).
select * from checkin_windows where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) for update;

-- name: AttWindowOpenForSession :one
select * from checkin_windows where course_id = sqlc.arg(course_id) and session_id = sqlc.arg(session_id) and closed_at is null for update;

-- name: AttWindowMaxRound :one
select coalesce(max(round), 0)::int as max_round from checkin_windows where course_id = sqlc.arg(course_id) and session_id = sqlc.arg(session_id);

-- name: AttWindowsBySession :many
select * from checkin_windows where course_id = sqlc.arg(course_id) and session_id = sqlc.arg(session_id) order by round limit 3;

-- name: AttWindowSetHeadcount :one
update checkin_windows set headcount = sqlc.narg(headcount) where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) returning *;

-- name: AttWindowClose :one
update checkin_windows set closed_at = sqlc.arg(closed_at), closed_by = sqlc.narg(closed_by)
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and closed_at is null returning *;

-- name: AttWindowsForCode :many
-- Dò mã ngắn: cửa sổ của lớp còn trong `CHECKIN_CODE_LOOKBACK_MIN` (kể cả đã đóng: để trả WINDOW_CLOSED). Tối đa 3 vòng × 2 buổi chồng nhau là quá đủ.
select * from checkin_windows where course_id = sqlc.arg(course_id) and closes_at > sqlc.arg(since) order by closes_at desc limit 6;

-- name: AttWindowCheckedIn :one
select count(*)::int from attendance_checkins where course_id = sqlc.arg(course_id) and window_id = sqlc.arg(window_id);

-- name: AttSweepIDs :many
-- Dùng checkin_windows_sweep_idx, không khoá (SweepJob gọi Close riêng từng cửa sổ).
select id, course_id from checkin_windows where closed_at is null and closes_at < sqlc.arg(now) order by closes_at limit 100;

-- name: AttSessionHasWindow :one
select exists (select 1 from checkin_windows where course_id = sqlc.arg(course_id) and session_id = sqlc.arg(session_id))::bool;

-- name: AttSessionOpenWindow :one
select exists (select 1 from checkin_windows where course_id = sqlc.arg(course_id) and session_id = sqlc.arg(session_id) and closed_at is null)::bool;

-- ---- thiết bị ----------------------------------------------------------------------------------------------------------------------
-- name: AttDeviceUpsertWeb :one
-- DO UPDATE rỗng (không DO NOTHING): trả hàng ở cả hai nhánh và giữ khoá hàng tới hết giao dịch (ARCHITECTURE 11.2).
insert into user_devices (key_sha256, public_key, kind) values (sqlc.arg(key_sha256), sqlc.arg(public_key), 'WEB_KEY')
on conflict (key_sha256) do update set key_sha256 = excluded.key_sha256
returning *;

-- name: AttDeviceByHashLock :one
select * from user_devices where key_sha256 = sqlc.arg(key_sha256) for update;

-- name: AttDeviceUpsertAttested :one
insert into user_devices (key_sha256, public_key, kind, attested, boot_verified, security_level)
values (sqlc.arg(key_sha256), sqlc.arg(public_key), 'ANDROID_KEY', sqlc.arg(attested), sqlc.narg(boot_verified), sqlc.narg(security_level))
on conflict (key_sha256) do update set attested = excluded.attested, boot_verified = excluded.boot_verified, security_level = excluded.security_level
returning *;

-- ---- lượt điểm danh ----------------------------------------------------------------------------------------------------------------
-- name: AttCheckinFind :one
select * from attendance_checkins where course_id = sqlc.arg(course_id) and window_id = sqlc.arg(window_id) and student_id = sqlc.arg(student_id);

-- name: AttCheckinInsert :one
insert into attendance_checkins (course_id, session_id, window_id, student_id, device_id, channel, assurance, token_age_ms, geo_distance_m, geo_accuracy_m, in_campus, flags, created_at)
values (sqlc.arg(course_id), sqlc.arg(session_id), sqlc.arg(window_id), sqlc.arg(student_id), sqlc.narg(device_id), sqlc.arg(channel), sqlc.arg(assurance), sqlc.arg(token_age_ms),
        sqlc.narg(geo_distance_m), sqlc.narg(geo_accuracy_m), sqlc.narg(in_campus), sqlc.arg(flags), sqlc.arg(created_at))
on conflict (course_id, window_id, student_id) do nothing
returning *;

-- name: AttSharedDeviceStudents :many
-- Sinh viên KHÁC đã điểm danh bằng cùng máy trong lớp (chỉ mục (course_id, device_id)).
select distinct student_id from attendance_checkins
where course_id = sqlc.arg(course_id) and device_id = sqlc.arg(device_id) and student_id <> sqlc.arg(student_id);

-- name: AttStudentDevices :one
-- Có lượt bằng máy KHÁC / bằng chính máy này (DEVICE_NEW = có khác và chưa có máy này).
select
  (select count(*) from attendance_checkins x where x.course_id = sqlc.arg(course_id) and x.student_id = sqlc.arg(student_id) and x.device_id is not null and x.device_id <> sqlc.arg(device_id)) > 0 as other,
  (select count(*) from attendance_checkins y where y.course_id = sqlc.arg(course_id) and y.student_id = sqlc.arg(student_id) and y.device_id = sqlc.arg(device_id)) > 0 as this;

-- name: AttFlagShared :exec
-- Gắn DEVICE_SHARED vào mọi lượt của các sinh viên đó trong cùng buổi (FR-12). `text[]` rồi ép `::uuid[]`: PgBouncer (QueryExecModeExec) không mã hoá được []uuid.UUID (D-AS-01).
update attendance_checkins set flags = array_append(flags, 'DEVICE_SHARED')
where course_id = sqlc.arg(course_id) and session_id = sqlc.arg(session_id) and student_id = any(sqlc.arg(student_ids)::text[]::uuid[]) and not ('DEVICE_SHARED' = any(flags));

-- ---- đóng cửa sổ -------------------------------------------------------------------------------------------------------------------
-- name: AttCloseWriteRecords :one
-- FR-6: một câu; PRESENT / LATE theo lượt ĐẦU TIÊN của sinh viên trong buổi; không bao giờ ABSENT; ON CONFLICT DO NOTHING giữ dòng của Staff.
with ins as (
  insert into attendance_records (course_id, session_id, student_id, status, marked_by, version)
  select c.course_id, c.session_id, c.student_id,
         (case when min(c.created_at) > s.starts_at + make_interval(mins => sqlc.arg(late_after_min)::int) then 'LATE' else 'PRESENT' end)::attendance_status,
         sqlc.arg(actor)::uuid, 1
  from attendance_checkins c join class_sessions s on s.course_id = c.course_id and s.id = c.session_id
  where c.course_id = sqlc.arg(course_id) and c.session_id = sqlc.arg(session_id)
  group by c.course_id, c.session_id, c.student_id, s.starts_at
  on conflict (course_id, session_id, student_id) do nothing
  returning status
)
select (count(*) filter (where status = 'PRESENT'))::int as present, (count(*) filter (where status = 'LATE'))::int as late from ins;

-- name: AttSessionCheckinStudents :one
select count(distinct student_id)::int from attendance_checkins where course_id = sqlc.arg(course_id) and session_id = sqlc.arg(session_id);

-- name: AttUnresolvedCount :one
-- Sinh viên ACTIVE của lớp chưa có dòng điểm danh ở buổi.
select count(*)::int from enrollments e
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE'
  and not exists (select 1 from attendance_records r where r.course_id = e.course_id and r.session_id = sqlc.arg(session_id) and r.student_id = e.user_id);

-- name: AttRecheckMiss :exec
-- FR-14: người có lượt ở vòng trước mà không có lượt ở vòng này.
update attendance_checkins c set flags = array_append(c.flags, 'RECHECK_MISS')
where c.course_id = sqlc.arg(course_id) and c.session_id = sqlc.arg(session_id)
  and c.window_id in (select w.id from checkin_windows w where w.course_id = sqlc.arg(course_id) and w.session_id = sqlc.arg(session_id) and w.round < sqlc.arg(round)::smallint)
  and not exists (select 1 from attendance_checkins d where d.course_id = sqlc.arg(course_id) and d.window_id = sqlc.arg(window_id) and d.student_id = c.student_id)
  and not ('RECHECK_MISS' = any(c.flags));

-- name: AttAbsentUnchecked :many
-- FR-8: ABSENT cho sinh viên ACTIVE chưa có dòng; DO NOTHING giữ mọi dòng đã có.
insert into attendance_records (course_id, session_id, student_id, status, marked_by, version)
select e.course_id, sqlc.arg(session_id)::uuid, e.user_id, 'ABSENT', sqlc.arg(actor)::uuid, 1
from enrollments e
where e.course_id = sqlc.arg(course_id) and e.role_in_course = 'STUDENT' and e.status = 'ACTIVE'
  and not exists (select 1 from attendance_records r where r.course_id = e.course_id and r.session_id = sqlc.arg(session_id) and r.student_id = e.user_id)
on conflict (course_id, session_id, student_id) do nothing
returning student_id;

-- ---- đọc cho Staff -----------------------------------------------------------------------------------------------------------------
-- name: AttCheckinsList :many
select c.id, c.student_id, c.window_id, c.channel, c.assurance, c.flags, c.flags_cleared_at, c.geo_distance_m, c.in_campus, c.created_at, w.round,
       u.full_name, coalesce(e.student_code_snapshot, '')::text as student_code
from attendance_checkins c
join checkin_windows w on w.course_id = c.course_id and w.id = c.window_id
join users u on u.id = c.student_id
join enrollments e on e.course_id = c.course_id and e.user_id = c.student_id
where c.course_id = sqlc.arg(course_id) and c.session_id = sqlc.arg(session_id)
  and (sqlc.arg(round)::int = 0 or w.round = sqlc.arg(round)::int)
  and (not sqlc.arg(flagged)::bool or (cardinality(c.flags) > 0 and c.flags_cleared_at is null))
  and (sqlc.narg(after_at)::timestamptz is null or (c.created_at, c.id) > (sqlc.narg(after_at)::timestamptz, sqlc.narg(after_id)::uuid))
order by c.created_at, c.id
limit sqlc.arg(row_limit);

-- name: AttCheckinByID :one
select * from attendance_checkins where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: AttCheckinClearFlags :one
update attendance_checkins set flags_cleared_by = sqlc.arg(actor), flags_cleared_at = sqlc.arg(at)
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and flags_cleared_at is null returning *;

-- name: AttSpotFlagged :many
select c.student_id, u.full_name, coalesce(e.student_code_snapshot, '')::text as student_code
from attendance_checkins c join users u on u.id = c.student_id join enrollments e on e.course_id = c.course_id and e.user_id = c.student_id
where c.course_id = sqlc.arg(course_id) and c.window_id = sqlc.arg(window_id) and cardinality(c.flags) > 0 and c.flags_cleared_at is null
order by c.created_at, c.id;

-- name: AttSpotAll :many
select c.student_id, u.full_name, coalesce(e.student_code_snapshot, '')::text as student_code
from attendance_checkins c join users u on u.id = c.student_id join enrollments e on e.course_id = c.course_id and e.user_id = c.student_id
where c.course_id = sqlc.arg(course_id) and c.window_id = sqlc.arg(window_id)
order by c.created_at, c.id;

-- ---- cho sinh viên / Hôm nay -------------------------------------------------------------------------------------------------------
-- name: AttMyCheckins :many
-- Giờ điểm danh sớm nhất của sinh viên theo buổi (chỉ thời điểm; không cờ, không mức tin cậy).
select session_id, min(created_at)::timestamptz as checked_in_at from attendance_checkins
where course_id = sqlc.arg(course_id) and student_id = sqlc.arg(student_id) group by session_id;

-- name: AttReviewSessions :many
-- Nguồn «Hôm nay» CHECKIN_REVIEW: buổi có cửa sổ đã đóng trong `since` và còn lượt có cờ chưa bỏ.
select s.course_id, s.id as session_id, s.session_no, count(*)::int as flagged, max(w.closed_at)::timestamptz as closed_at
from attendance_checkins c
join checkin_windows w on w.course_id = c.course_id and w.id = c.window_id and w.closed_at is not null and w.closed_at >= sqlc.arg(since)
join class_sessions s on s.course_id = c.course_id and s.id = c.session_id
where c.course_id = any(sqlc.arg(course_ids)::text[]::uuid[]) and cardinality(c.flags) > 0 and c.flags_cleared_at is null
group by s.course_id, s.id, s.session_no
order by max(w.closed_at) desc
limit 50;

-- ---- giữ dữ liệu -------------------------------------------------------------------------------------------------------------------
-- name: AttPurgeCheckins :execrows
delete from attendance_checkins where id in (
  select c.id from attendance_checkins c join checkin_windows w on w.course_id = c.course_id and w.id = c.window_id
  where w.closed_at < sqlc.arg(cutoff) limit 1000);

-- name: AttPurgeDevices :execrows
delete from user_devices where id in (
  select d.id from user_devices d
  where d.created_at < sqlc.arg(cutoff) and not exists (select 1 from attendance_checkins c where c.device_id = d.id)
  limit 1000 for update skip locked);

-- name: AttForgetStudent :execrows
delete from attendance_checkins where student_id = sqlc.arg(student_id);

-- name: AttClosedWritten :one
-- Kết quả đã lưu của một lần đóng (gọi lại close): dòng điểm danh do chính người đóng ghi cho những sinh viên có lượt trong buổi.
select (count(*) filter (where r.status = 'PRESENT'))::int as present, (count(*) filter (where r.status = 'LATE'))::int as late
from attendance_records r
where r.course_id = sqlc.arg(course_id) and r.session_id = sqlc.arg(session_id) and r.marked_by = sqlc.arg(actor)
  and r.student_id in (select c.student_id from attendance_checkins c where c.course_id = sqlc.arg(course_id) and c.session_id = sqlc.arg(session_id));

-- name: AttSessionExists :one
select id from class_sessions where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);
