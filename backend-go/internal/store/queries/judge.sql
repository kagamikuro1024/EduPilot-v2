-- Hàng đợi chấm code (SRS FEAT-weekly-exam 4.5.7, góp ý #3). Trạng thái nhận việc / thuê / thử lại nằm Ở ĐÂY; Redis Stream chỉ là tín hiệu đánh thức.

-- name: JudgeSupersedeQueued :one
-- Bản SUBMIT đang xếp hàng mà đã có SUBMIT mới hơn của cùng (lượt, mục) → SUPERSEDED, không chạy sandbox. `Chạy thử` (RUN) không bao giờ bị thay.
update code_submissions c
set status = 'SUPERSEDED', enqueued_at = null
where c.id = sqlc.arg(id) and c.status = 'QUEUED' and c.kind = 'SUBMIT'
  and exists (
    select 1 from code_submissions n
    where n.attempt_id = c.attempt_id and n.item_id = c.item_id and n.kind = 'SUBMIT' and n.status <> 'SUPERSEDED'
      and (n.created_at, n.id) > (c.created_at, c.id)
  )
returning c.id;

-- name: JudgeClaim :one
-- Nhận việc bằng MỘT lệnh: dòng QUEUED đến hạn HOẶC RUNNING hết thuê. 0 dòng ⇒ nơi khác đang chấm / đã xong / chưa đến hạn thử lại.
update code_submissions
set status = 'RUNNING', lease_until = now() + sqlc.arg(lease_ms)::int * interval '1 millisecond', attempts = attempts + 1, enqueued_at = coalesce(enqueued_at, now())
where id = sqlc.arg(id)
  and ((status = 'QUEUED' and next_attempt_at <= now()) or (status = 'RUNNING' and lease_until < now()))
returning id, course_id, exam_id, attempt_id, item_id, problem_id, kind, language, source, fail_count, lease_until;

-- name: JudgeRenew :one
update code_submissions
set lease_until = now() + sqlc.arg(lease_ms)::int * interval '1 millisecond'
where id = sqlc.arg(id) and status = 'RUNNING' and lease_until = sqlc.arg(old_lease)::timestamptz
returning lease_until;

-- name: JudgeFinish :one
-- Ghi kết quả, rào chắn bằng thuê của mình: thuê đã bị nhận lại ⇒ 0 dòng ⇒ bỏ kết quả.
update code_submissions
set status = 'DONE', verdict = sqlc.arg(verdict)::judge_verdict, tests_version = sqlc.arg(tests_version), compile_ok = sqlc.arg(compile_ok),
    compile_log = sqlc.narg(compile_log), results = sqlc.narg(results), passed_weight = sqlc.arg(passed_weight), total_weight = sqlc.arg(total_weight),
    time_ms_max = sqlc.narg(time_ms_max), memory_kb_max = sqlc.narg(memory_kb_max), lease_until = null, judged_at = now()
where id = sqlc.arg(id) and status = 'RUNNING' and lease_until = sqlc.arg(lease_until)::timestamptz
returning id, course_id, exam_id, attempt_id, item_id, kind;

-- name: JudgeFail :one
-- Lỗi THẬT (sandbox lỗi / IE): fail_count + 1; < 4 ⇒ QUEUED với next_attempt_at (1 s / 5 s / 30 s); = 4 ⇒ ERROR + IE.
update code_submissions
set fail_count = fail_count + 1,
    status = case when fail_count + 1 >= 4 then 'ERROR'::submission_status else 'QUEUED'::submission_status end,
    verdict = case when fail_count + 1 >= 4 then 'IE'::judge_verdict end,
    lease_until = null, enqueued_at = null, next_attempt_at = now() + sqlc.arg(backoff_ms)::int * interval '1 millisecond', judged_at = case when fail_count + 1 >= 4 then now() end
where id = sqlc.arg(id) and status = 'RUNNING' and lease_until = sqlc.arg(lease_until)::timestamptz
returning id, status, fail_count, kind, course_id, exam_id, attempt_id;

-- name: JudgeConfigError :one
-- Lỗi cấu hình của bài (Σ weight = 0 …): IE ngay, không thử lại, không điểm 0 cho sinh viên (attempt ở lại GRADING cho tới khi sửa + chấm lại).
update code_submissions
set status = 'ERROR', verdict = 'IE', compile_log = sqlc.arg(compile_log), lease_until = null, judged_at = now()
where id = sqlc.arg(id) and status = 'RUNNING' and lease_until = sqlc.arg(lease_until)::timestamptz
returning id, kind, course_id, exam_id, attempt_id;

-- name: JudgeMarkEnqueued :execrows
update code_submissions set enqueued_at = now() where id = sqlc.arg(id) and status = 'QUEUED';

-- name: JudgeDueUnqueued :many
-- Tick: dòng QUEUED đến hạn mà chưa XADD (hoặc bị đặt lại để đưa lại).
select id, kind from code_submissions
where status = 'QUEUED' and enqueued_at is null and next_attempt_at <= now()
order by next_attempt_at, id
limit sqlc.arg(max_rows);

-- name: JudgeRequeueExpired :many
-- Tick: RUNNING hết thuê (worker chết / treo) ⇒ QUEUED, KHÔNG tăng fail_count (chưa chắc lỗi của bản nộp); lần sau tick XADD lại.
update code_submissions
set status = 'QUEUED', lease_until = null, enqueued_at = null
where status = 'RUNNING' and lease_until < now()
returning id, kind;

-- name: JudgeResetStaleSignal :execrows
-- Tick: QUEUED đã có enqueued_at quá lâu mà chưa ai nhận (Stream mất tin) ⇒ đưa lại.
update code_submissions set enqueued_at = null
where status = 'QUEUED' and enqueued_at is not null and enqueued_at < now() - sqlc.arg(idle_ms)::int * interval '1 millisecond';

-- name: JudgeProblem :one
select time_limit_ms, memory_limit_mb, output_limit_kb, checker, float_eps, tests_version
from code_problems where course_id = sqlc.arg(course_id) and question_id = sqlc.arg(question_id);

-- name: JudgeApprovedTests :many
select id, position, is_sample, weight, input, input_blob_key, expected, expected_blob_key
from code_testcases
where course_id = sqlc.arg(course_id) and problem_id = sqlc.arg(problem_id) and approved
order by position;
