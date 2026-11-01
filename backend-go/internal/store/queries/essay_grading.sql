-- name: EssayGradeContext :one
-- Mọi thứ Grading Engine cần cho một việc chấm: kết quả, bài nộp, bài tập (rubric + đáp án chuẩn + quy tắc nộp muộn HIỆN TẠI — TLR-8).
select r.id as result_id, r.course_id, r.submission_id, r.round, r.status, r.run_state, r.late_penalty_manual, r.late_penalty as manual_late_penalty,
       s.assignment_id, s.student_id, s.source, s.blob_key, s.file_name, s.file_mime, s.extracted_text, s.extract_status, s.submitted_at,
       a.title, a.description, a.max_score, a.due_at, a.allow_late, a.late_penalty_per_day, a.late_max_days, a.reference_answer, a.rubric
from grading_results r
join submissions s on s.course_id = r.course_id and s.id = r.submission_id
join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
where r.course_id = sqlc.arg(course_id) and r.id = sqlc.arg(result_id);

-- name: EssaySetExtract :exec
-- `extracted_text` ghi MỘT lần (trigger bất biến); lần sau chỉ đổi extract_status khi chưa có chữ.
update submissions set extracted_text = coalesce(extracted_text, sqlc.narg(text)), extracted_at = coalesce(extracted_at, now()), extract_status = sqlc.arg(status)
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: EssaySaveDraft :execrows
-- Ghi bản nháp (DONE / DRAFT). `ai_scores` và `ai_total_before_late` chỉ ghi lần đầu (trigger chặn đổi). Không đụng kết quả đã REVIEWED / PUBLISHED.
update grading_results set
    run_state = 'DONE', error_code = null, lease_owner = null, lease_until = null,
    criteria = sqlc.arg(criteria), overall_feedback = sqlc.arg(overall_feedback), total = sqlc.arg(total),
    ai_scores = coalesce(ai_scores, sqlc.arg(ai_scores)), ai_total_before_late = coalesce(ai_total_before_late, sqlc.arg(ai_total_before_late)),
    late_days = sqlc.arg(late_days), late_penalty = sqlc.arg(late_penalty), llm_reported_total = sqlc.narg(llm_reported_total),
    flags = sqlc.arg(flags), confidence = sqlc.narg(confidence), pass_detail = sqlc.narg(pass_detail), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'DRAFT'
  and (sqlc.narg(owner)::text is null or (run_state = 'RUNNING' and lease_owner = sqlc.narg(owner)));

-- name: EssaySaveFailed :execrows
-- Bài lỗi: mọi tiêu chí NULL (không ghi 0 điểm), `ai_scores` NULL.
update grading_results set
    run_state = 'FAILED', error_code = sqlc.arg(error_code), lease_owner = null, lease_until = null,
    criteria = sqlc.arg(criteria), overall_feedback = null, total = null, flags = sqlc.arg(flags), version = version + 1
where course_id = sqlc.arg(course_id) and id = sqlc.arg(id) and status = 'DRAFT'
  and (sqlc.narg(owner)::text is null or (run_state = 'RUNNING' and lease_owner = sqlc.narg(owner)));
