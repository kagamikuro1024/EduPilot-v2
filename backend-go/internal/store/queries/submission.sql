-- US-P7-03 — bài nộp (InAppSource) và đọc phía sinh viên. Mọi truy vấn bắt đầu từ course_id (luật 13).

-- name: SubmissionLockStudent :exec
-- Khoá cấp `version` theo (bài, sinh viên): mọi nguồn (IN_APP, UPLOAD, IMAP, TEAMS) đi qua cùng một khoá (SRS 4.3.2, mẫu TicketLockStudentCap).
select pg_advisory_xact_lock(hashtextextended('submission:' || sqlc.arg(assignment_id)::text || ':' || sqlc.arg(student_id)::text, 0));

-- name: SubmissionLatestMatched :one
-- Bản MATCHED mới nhất của (bài, sinh viên) kèm trạng thái kết quả hiện hành (vòng chấm cao nhất) — null nếu chưa có kết quả.
select s.id, s.version, s.file_name, s.received_at, s.submitted_at, coalesce(r.status, '')::text as result_status, coalesce(r.run_state, '')::text as result_run_state
from submissions s
left join lateral (select gr.status, gr.run_state from grading_results gr where gr.course_id = s.course_id and gr.submission_id = s.id order by gr.round desc limit 1) r on true
where s.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and s.student_id = sqlc.arg(student_id)::uuid and s.match_status = 'MATCHED'
order by s.version desc
limit 1;

-- name: SubmissionInsertInApp :one
insert into submissions (course_id, assignment_id, student_id, source, version, blob_key, file_name, file_size, file_mime, sha256, match_status, match_method, submitted_at, received_at)
values (sqlc.arg(course_id), sqlc.arg(assignment_id), sqlc.arg(student_id), 'IN_APP',
        (select coalesce(max(v.version), 0) + 1 from submissions v where v.course_id = sqlc.arg(course_id) and v.assignment_id = sqlc.arg(assignment_id)::uuid and v.student_id = sqlc.arg(student_id)::uuid),
        sqlc.arg(blob_key), sqlc.arg(file_name), sqlc.arg(file_size), sqlc.arg(file_mime), sqlc.arg(sha256), 'MATCHED', 'IN_APP', sqlc.arg(at)::timestamptz, sqlc.arg(at)::timestamptz)
returning id, version, received_at, submitted_at;

-- name: SubmissionMineBatch :many
-- Bản MATCHED mới nhất của CHÍNH sinh viên cho các bài của một trang danh sách (một truy vấn, không N+1); bài chưa nộp không có dòng.
select distinct on (s.assignment_id) s.assignment_id, s.id as submission_id, s.version, s.file_name, s.received_at, s.submitted_at, coalesce(r.status, '')::text as result_status
from submissions s
left join lateral (select gr.status from grading_results gr where gr.course_id = s.course_id and gr.submission_id = s.id order by gr.round desc limit 1) r on true
where s.course_id = sqlc.arg(course_id) and s.student_id = sqlc.arg(student_id)::uuid and s.match_status = 'MATCHED' and s.assignment_id::text = any(sqlc.arg(ids)::text[])
order by s.assignment_id, s.version desc;

-- name: SubmissionBlobKeysUsed :many
-- Trong danh sách khoá, những khoá đã có bài nộp trỏ tới (job dọn tệp mồ côi giữ lại).
select s.blob_key from submissions s where s.blob_key = any(sqlc.arg(keys)::text[]);

-- name: TodayStudentAssignments :many
-- "Hôm nay" của sinh viên (ASSIGNMENT_DUE, bậc 41): bài đã công bố, CHƯA nộp (không có bài MATCHED của mình), hạn trong `horizon` hoặc đã quá hạn nhưng còn cửa muộn — MỘT truy vấn cho mọi lớp.
select a.id, a.course_id, a.title, a.due_at, a.allow_late, a.late_max_days
from assignments a
where a.course_id = any(sqlc.arg(course_ids)::text[]::uuid[])
  and a.published_at is not null
  and 'IN_APP' = any(a.channels)
  and a.due_at <= sqlc.arg(horizon)::timestamptz
  and (a.due_at > sqlc.arg(now)::timestamptz or (a.allow_late and a.due_at + make_interval(days => a.late_max_days::int) > sqlc.arg(now)::timestamptz))
  and not exists (select 1 from submissions s where s.course_id = a.course_id and s.assignment_id = a.id and s.student_id = sqlc.arg(user_id)::uuid and s.match_status = 'MATCHED')
order by a.due_at, a.id;
