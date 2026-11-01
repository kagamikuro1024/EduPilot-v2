-- US-P7-07 — hàng đợi chấm: nạp, thuê, gia hạn, thử lại, quét, thay thế bản cũ. DB là nguồn sự thật; Redis Stream chỉ là tín hiệu.

-- name: GradeAllCandidates :many
-- Bài nộp MATCHED PHIÊN BẢN MỚI NHẤT của từng sinh viên chưa có kết quả nào (kể cả đang xếp hàng / đang chạy / đã chấm).
select s.id as submission_id
from submissions s
where s.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and s.match_status = 'MATCHED'
  and s.version = (select max(v.version) from submissions v where v.course_id = s.course_id and v.assignment_id = s.assignment_id and v.student_id = s.student_id and v.match_status = 'MATCHED')
  and not exists (select 1 from grading_results r where r.course_id = s.course_id and r.submission_id = s.id)
order by s.received_at, s.id
for update of s;

-- name: GradeFailedCandidates :many
-- Kết quả FAILED hiện hành (vòng cao nhất) của bản nộp mới nhất — `retry_failed` của grade-all (cùng round, attempts = 0).
select r.id
from grading_results r join submissions s on s.course_id = r.course_id and s.id = r.submission_id
where r.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and s.match_status = 'MATCHED' and r.run_state = 'FAILED' and r.status = 'DRAFT'
  and coalesce(r.error_code, '') <> 'SUPERSEDED'
  and r.round = (select max(x.round) from grading_results x where x.course_id = r.course_id and x.submission_id = r.submission_id)
  and s.version = (select max(v.version) from submissions v where v.course_id = s.course_id and v.assignment_id = s.assignment_id and v.student_id = s.student_id and v.match_status = 'MATCHED')
for update of r;

-- name: GradingInsertQueued :one
insert into grading_results (course_id, submission_id, round, run_state, not_before)
values (sqlc.arg(course_id), sqlc.arg(submission_id), sqlc.arg(round), 'QUEUED', now())
on conflict (course_id, submission_id, round) do nothing
returning id;

-- name: GradingCurrent :one
-- Kết quả hiện hành (vòng cao nhất) của một bài nộp.
select id, round, status, run_state, version, (select s.assignment_id from submissions s where s.course_id = r.course_id and s.id = r.submission_id) as assignment_id
from grading_results r
where r.course_id = sqlc.arg(course_id) and r.submission_id = sqlc.arg(submission_id)
order by r.round desc limit 1;

-- name: GradingSubmissionOK :one
-- Bài nộp có MATCHED và thuộc lớp (chấm một bài).
select s.id, s.match_status from submissions s where s.course_id = sqlc.arg(course_id) and s.id = sqlc.arg(id);

-- name: GradingRetryFailed :one
-- `retry`: chỉ FAILED → QUEUED, attempts = 0, cùng round.
update grading_results set run_state = 'QUEUED', attempts = 0, error_code = null, not_before = now(), lease_owner = null, lease_until = null, updated_at = now(), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and run_state = 'FAILED' and status = 'DRAFT'
returning id, submission_id;

-- name: GradingResultState :one
select r.id, r.run_state, r.status, r.submission_id, r.round, r.error_code from grading_results r where r.course_id = sqlc.arg(course_id) and r.id = sqlc.arg(id);

-- name: GradingClaim :one
-- Nhận việc (thuê). 0 dòng → bỏ qua. Mọi so sánh thời gian dùng now() của DB.
update grading_results set run_state = 'RUNNING', lease_owner = sqlc.arg(owner), lease_until = now() + make_interval(secs => sqlc.arg(lease_secs)::float8), attempts = attempts + 1, updated_at = now()
where id = sqlc.arg(id) and ((run_state = 'QUEUED' and not_before <= now()) or (run_state = 'RUNNING' and lease_until < now()))
returning id, course_id, submission_id, attempts, lease_until;

-- name: GradingRenew :execrows
update grading_results set lease_until = now() + make_interval(secs => sqlc.arg(lease_secs)::float8)
where id = sqlc.arg(id) and run_state = 'RUNNING' and lease_owner = sqlc.arg(owner);

-- name: GradingRequeue :execrows
-- Lỗi tạm còn lượt: QUEUED lại với not_before (giãn cách làm bằng cột, consumer không ngủ).
update grading_results set run_state = 'QUEUED', not_before = now() + make_interval(secs => sqlc.arg(delay_secs)::float8), lease_owner = null, lease_until = null, updated_at = now()
where id = sqlc.arg(id) and run_state = 'RUNNING' and lease_owner = sqlc.arg(owner);

-- name: GradingFailPermanent :execrows
-- Hết lượt / lỗi vĩnh viễn / bị thay: FAILED ở DB (mọi tiêu chí NULL; không bao giờ 0 điểm). `owner` rỗng = không kiểm thuê (bản cũ bị thay khi còn QUEUED).
update grading_results set run_state = 'FAILED', error_code = sqlc.arg(code), lease_owner = null, lease_until = null, updated_at = now(), version = version + 1
where id = sqlc.arg(id) and status = 'DRAFT' and (run_state = 'QUEUED' or (run_state = 'RUNNING' and (sqlc.arg(owner)::text = '' or lease_owner = sqlc.arg(owner))));

-- name: GradingIsSuperseded :one
-- Bản nộp của việc không còn là bản MATCHED mới nhất của (bài, sinh viên).
select (s.match_status <> 'MATCHED' or s.version < (select max(v.version) from submissions v where v.course_id = s.course_id and v.assignment_id = s.assignment_id and v.student_id = s.student_id and v.match_status = 'MATCHED'))::bool as superseded
from grading_results r join submissions s on s.course_id = r.course_id and s.id = r.submission_id
where r.id = sqlc.arg(id);

-- name: GradingSweep :many
-- Máy quét: QUEUED đã tới giờ mà chưa được nạp lại ≥ requeue_secs (mất tin Redis) hoặc vừa tới giờ thử lại (attempts > 0, chưa nạp lại từ khi tới giờ). Chạm updated_at để không nạp lại liên tục.
update grading_results set updated_at = now()
where id in (
  select r.id from grading_results r
  where r.run_state = 'QUEUED' and r.not_before <= now()
    and (r.updated_at <= now() - make_interval(secs => sqlc.arg(requeue_secs)::float8) or (r.attempts > 0 and r.updated_at <= r.not_before))
  order by r.not_before limit 500 for update skip locked)
returning id, course_id, submission_id, round;

-- name: GradingExpiredLeases :many
-- Việc RUNNING hết thuê (worker chết): nạp lại để consumer khác nhận (XAUTOCLAIM chỉ lo tin treo trong PEL).
select r.id, r.course_id, r.submission_id, r.round from grading_results r where r.run_state = 'RUNNING' and r.lease_until < now() limit 500;

-- name: GradingBatchCounts :one
-- Tiến độ của một lô (danh sách id): xong / lỗi / tổng.
select count(*) filter (where run_state = 'DONE')::int as done, count(*) filter (where run_state = 'FAILED')::int as failed, count(*)::int as total,
       count(*) filter (where run_state = 'FAILED' and error_code = 'LLM_BUDGET_EXHAUSTED')::int as budget,
       count(*) filter (where run_state = 'DONE' and cardinality(flags) > 0)::int as flagged
from grading_results where id = any(sqlc.arg(ids)::text[]::uuid[]);

-- name: GradingQueueDepth :one
select count(*)::int from grading_results where run_state = 'QUEUED';

-- name: GradingStaffIDs :many
-- Nhân viên ACTIVE của lớp (thông báo gộp khi hết ngân sách).
select e.user_id from enrollments e where e.course_id = sqlc.arg(course_id) and e.role_in_course in ('TEACHER', 'TA') and e.status = 'ACTIVE';

-- name: GradingBatchRows :many
select id, course_id, submission_id, round from grading_results where id = any(sqlc.arg(ids)::text[]::uuid[]);

-- name: GradingIsStaff :one
select exists (select 1 from enrollments e where e.course_id = sqlc.arg(course_id) and e.user_id = sqlc.arg(user_id) and e.role_in_course in ('TEACHER', 'TA') and e.status = 'ACTIVE');
