-- Harness học từ lần giảng viên sửa (SRS FEAT-grading-guidance, D61 / D62). Mọi truy vấn bắt đầu bằng course_id (luật 13).

-- ---- cờ theo lớp (mục 11 TL-1) ---------------------------------------------------------------------------------------------
-- name: GuidanceFlagsAll :many
select course_id, feature from course_guidance_flags where enabled;

-- name: GuidanceFlagList :many
select feature, enabled, version from course_guidance_flags where course_id = sqlc.arg(course_id);

-- Đặt cờ theo version (hàng chưa có = version 1 ảo, enabled = false). Trả hàng khi thắng; không trả = version cũ (409).
-- name: GuidanceFlagSet :one
insert into course_guidance_flags (course_id, feature, enabled, version, updated_by)
values (sqlc.arg(course_id), sqlc.arg(feature), sqlc.arg(enabled), 2, sqlc.arg(actor))
on conflict (course_id, feature) do update
   set enabled = excluded.enabled, version = course_guidance_flags.version + 1, updated_by = excluded.updated_by
 where course_guidance_flags.version = sqlc.arg(version)
returning feature, enabled, version;

-- ---- phiên bản hiện hành và tập lưu ý ----------------------------------------------------------------------------------------
-- name: GuidanceCurrentVersion :one
select guidance_version, type from assignments where course_id = sqlc.arg(course_id) and id = sqlc.arg(assignment_id);

-- Tập lưu ý tại phiên bản v (SRS 4.2.3), hàm thuần theo (assignment_id, v).
-- name: GuidanceSetAt :many
select id, criterion_id, text, approved_in_version
from grading_guidance_items
where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id)
  and status in ('APPROVED', 'RETIRED') and approved_in_version is not null and approved_in_version <= sqlc.arg(v)::smallint
  and (retired_in_version is null or retired_in_version > sqlc.arg(v)::smallint)
order by approved_in_version, id;

-- name: GuidanceItems :many
select id, criterion_id, text, text_original, direction, support, needs_recheck, status, pending_retire, approved_in_version, retired_in_version, version, created_at
from grading_guidance_items
where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id)
  and not (status = 'REJECTED' and decided_at < now() - interval '30 days')
  and (sqlc.narg(cursor_at)::timestamptz is null or (created_at, id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by created_at desc, id desc
limit sqlc.arg(page_limit);

-- name: GuidanceItemCounts :many
select status, pending_retire, count(*)::int as n from grading_guidance_items
where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id)
group by status, pending_retire;

-- Mọi mục (kể cả REJECTED < 30 ngày) để Jaccard chống trùng (SRS 4.2.8 (vii)).
-- name: GuidanceTextsForDedupe :many
select criterion_id, text from grading_guidance_items
where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id)
  and not (status = 'REJECTED' and decided_at < now() - interval '30 days');

-- name: GuidanceItemGet :one
select * from grading_guidance_items where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id) and id = sqlc.arg(id) for update;

-- name: GuidanceInsertDraft :one
insert into grading_guidance_items (course_id, assignment_id, criterion_id, text, direction, support, source_result_ids, origin_job)
values (sqlc.arg(course_id), sqlc.arg(assignment_id), sqlc.arg(criterion_id), sqlc.arg(text), sqlc.arg(direction), sqlc.arg(support), sqlc.arg(source_result_ids)::text[]::uuid[], sqlc.narg(origin_job))
returning id;

-- APPROVE / REJECT (chỉ từ DRAFT) hoặc RETIRE (mục đang hiệu lực: pending_retire). `text` sửa chỉ ở DRAFT; text_original lưu một lần.
-- name: GuidanceDecide :execrows
update grading_guidance_items set
    status = sqlc.arg(status)::guidance_item_status,
    decided_by = sqlc.arg(actor), decided_at = now(),
    text = coalesce(sqlc.narg(new_text), text),
    text_original = case when sqlc.narg(new_text)::text is not null then coalesce(text_original, text) else text_original end,
    version = version + 1
where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id) and id = sqlc.arg(id) and version = sqlc.arg(version) and status = 'DRAFT';

-- name: GuidanceEditText :execrows
update grading_guidance_items set text = sqlc.arg(new_text), text_original = coalesce(text_original, text), version = version + 1
where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id) and id = sqlc.arg(id) and version = sqlc.arg(version) and status = 'DRAFT';

-- name: GuidanceMarkRetire :execrows
update grading_guidance_items set pending_retire = true, version = version + 1
where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id) and id = sqlc.arg(id) and version = sqlc.arg(version)
  and status = 'APPROVED' and approved_in_version is not null and not pending_retire;

-- Đặt lại / Hoàn tác (P-106-8, RESET): đưa lưu ý đã duyệt / bỏ NHƯNG CHƯA ÁP DỤNG về DRAFT; GIỮ text và text_original (chữ đã sửa không bị đảo).
-- name: GuidanceReset :execrows
update grading_guidance_items set status = 'DRAFT', decided_by = null, decided_at = null, version = version + 1
where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id) and id = sqlc.arg(id) and version = sqlc.arg(version)
  and status in ('APPROVED', 'REJECTED') and approved_in_version is null;

-- name: GuidanceResetRetire :execrows
update grading_guidance_items set pending_retire = false, version = version + 1
where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id) and id = sqlc.arg(id) and version = sqlc.arg(version)
  and status = 'APPROVED' and approved_in_version is not null and pending_retire;

-- Áp dụng (mục 5 TL-1, một giao dịch): khoá assignments, rồi hai lệnh sau.
-- name: GuidanceLockAssignment :one
select guidance_version from assignments where course_id = sqlc.arg(course_id) and id = sqlc.arg(assignment_id) for update;

-- name: GuidanceApplyApprove :execrows
update grading_guidance_items set approved_in_version = sqlc.arg(new_version)::smallint, version = version + 1
where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id) and status = 'APPROVED' and approved_in_version is null;

-- name: GuidanceApplyRetire :execrows
update grading_guidance_items set status = 'RETIRED', retired_in_version = sqlc.arg(new_version)::smallint, pending_retire = false, version = version + 1
where course_id = sqlc.arg(course_id) and assignment_id = sqlc.arg(assignment_id) and status = 'APPROVED' and pending_retire and approved_in_version is not null;

-- name: GuidanceVersionInsert :exec
insert into grading_guidance_versions (course_id, assignment_id, version, item_count, created_by)
values (sqlc.arg(course_id), sqlc.arg(assignment_id), sqlc.arg(version), sqlc.arg(item_count), sqlc.arg(actor));

-- Đây là CÂU DUY NHẤT đổi assignments.guidance_version (TestGuidanceVersionOnlyViaApply).
-- name: GuidanceSetVersion :exec
UPDATE assignments SET guidance_version = sqlc.arg(new_version)::smallint, updated_at = now() WHERE course_id = sqlc.arg(course_id) AND id = sqlc.arg(assignment_id);

-- ---- ca đầu vào cho reflect và điều kiện (SRS 4.2.5, 4.2.6) --------------------------------------------------------------------
-- Kết quả HIỆN HÀNH (vòng cao nhất, bài nộp mới nhất) REVIEWED / PUBLISHED có ai_scores; KHÔNG trả văn bản bài; `criteria` chỉ để lấy điểm cuối.
-- name: GuidanceCases :many
select r.id, r.criteria, r.ai_scores, r.ai_comments, r.reviewed_at
from grading_results r
join submissions s on s.course_id = r.course_id and s.id = r.submission_id
where r.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and r.status in ('REVIEWED', 'PUBLISHED') and r.ai_scores is not null
  and r.round = (select max(x.round) from grading_results x where x.course_id = r.course_id and x.submission_id = r.submission_id)
  and not exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version)
order by r.reviewed_at desc nulls last, r.id
limit 200;

-- name: GuidanceMaxReviewedAt :one
select coalesce(max(r.reviewed_at), 'epoch'::timestamptz)::timestamptz as at
from grading_results r join submissions s on s.course_id = r.course_id and s.id = r.submission_id
where r.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and r.status in ('REVIEWED', 'PUBLISHED');

-- ---- chấm lại và số bài cũ (SRS 4.2.11, §7) ---------------------------------------------------------------------------------------
-- Kết quả hiện hành chưa công bố có guidance_version cũ: ba nhóm đếm ở Go (cần changedCriteria):
-- name: GuidanceStaleRows :many
select r.id, r.status, r.criteria, r.ai_scores, r.first_reviewed_at, r.run_state, r.round, r.submission_id, r.guidance_version
from grading_results r
join submissions s on s.course_id = r.course_id and s.id = r.submission_id
join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
where r.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and r.status in ('DRAFT', 'REVIEWED') and r.run_state in ('DONE', 'FAILED')
  and r.guidance_version < a.guidance_version
  and r.round = (select max(x.round) from grading_results x where x.course_id = r.course_id and x.submission_id = r.submission_id)
  and not exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version)
order by r.id;

-- Ứng viên chấm lại: DRAFT bản cũ, chưa từng duyệt, bài nộp mới nhất, round < 20. Lọc changedCriteria = ∅ ở Go (so decimal).
-- name: GuidanceStaleCandidates :many
select r.id, r.criteria, r.ai_scores, r.submission_id, r.round
from grading_results r
join submissions s on s.course_id = r.course_id and s.id = r.submission_id
join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
where r.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and r.status = 'DRAFT' and r.run_state in ('DONE', 'FAILED')
  and r.first_reviewed_at is null and r.guidance_version < a.guidance_version and r.round < 20
  and r.round = (select max(x.round) from grading_results x where x.course_id = r.course_id and x.submission_id = r.submission_id)
  and not exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version)
order by r.id
limit sqlc.arg(page_limit)
for update of r;

-- name: GuidanceStatsRows :many
select r.guidance_version, r.criteria, r.ai_scores, r.ai_total_before_late, r.total, r.late_days, r.late_penalty
from grading_results r
join submissions s on s.course_id = r.course_id and s.id = r.submission_id
where r.course_id = sqlc.arg(course_id) and s.assignment_id = sqlc.arg(assignment_id)::uuid and r.status in ('REVIEWED', 'PUBLISHED') and r.ai_scores is not null
  and r.round = (select max(x.round) from grading_results x where x.course_id = r.course_id and x.submission_id = r.submission_id)
  and not exists (select 1 from submissions n where n.course_id = s.course_id and n.assignment_id = s.assignment_id and n.student_id = s.student_id and n.match_status = 'MATCHED' and n.version > s.version);

-- ---- M2 ca hồi quy -----------------------------------------------------------------------------------------------------------------
-- name: RegressionCaseUpsert :exec
insert into grading_regression_cases (course_id, kind, source_id, criterion_id, expected)
values (sqlc.arg(course_id), sqlc.arg(kind), sqlc.arg(source_id), sqlc.narg(criterion_id), sqlc.arg(expected))
on conflict (course_id, kind, source_id, coalesce(criterion_id, '')) do update set expected = excluded.expected, updated_at = now();

-- name: RegressionCaseCount :many
select kind, count(*)::int as n from grading_regression_cases where course_id = sqlc.arg(course_id) group by kind;

-- name: RegressionCases :many
select id, kind, source_id, criterion_id, expected from grading_regression_cases
where course_id = sqlc.arg(course_id) and (sqlc.narg(kind)::text is null or kind like sqlc.narg(kind)::text || '%')
order by created_at, id limit sqlc.arg(page_limit);

-- ---- M4 câu tương tự đã xác nhận ---------------------------------------------------------------------------------------------------
-- name: KnowledgeSimilarPosts :many
select p.thread_id, t.title, (1 - (c.embedding <=> sqlc.arg(vec)::vector))::float8 as cosine
from content_chunks c
join knowledge_items k on k.chunk_id = c.id and k.source_type = 'POST' and k.course_id = sqlc.arg(course_id)
join forum_posts p on p.id = k.source_id and p.course_id = k.course_id
join forum_threads t on t.id = p.thread_id and t.course_id = p.course_id
where c.course_ids @> array[sqlc.arg(course_id)::uuid] and c.embedding is not null
  and p.verification_state in ('VERIFIED', 'CORRECTED') and p.hidden_at is null and p.deleted_at is null
  and t.hidden_at is null and t.deleted_at is null
order by c.embedding <=> sqlc.arg(vec)::vector
limit 3;

-- ---- M6 lý do loại ------------------------------------------------------------------------------------------------------------------
-- name: QuestionRejectHints :many
select reject_reason, count(*)::int as n from question_bank
where course_id = sqlc.arg(course_id) and review_status = 'REJECTED' and reject_reason is not null and reject_reason <> 'UNSPECIFIED'
  and reviewed_at >= now() - make_interval(days => sqlc.arg(window_days)::int)
group by reject_reason order by n desc, reject_reason;

-- name: GuidanceForResult :one
-- Phiên bản lưu ý đã ghim của kết quả + phiên bản hiện hành của bài tập (màn duyệt, SRS 4.2.3).
select r.guidance_version, a.guidance_version as current_version, a.id as assignment_id, r.status::text as status, (r.first_reviewed_at is not null)::boolean as was_reviewed
from grading_results r
join submissions s on s.course_id = r.course_id and s.id = r.submission_id
join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
where r.course_id = sqlc.arg(course_id) and r.id = sqlc.arg(id);

-- name: PracticeGenHints :one
-- Lý do «Tránh» đã dùng cho một lô sinh câu (lưu vào jobs.result lúc chạy, không tính lại).
select coalesce(result -> 'hints', '[]'::jsonb) as hints from jobs where id = sqlc.arg(job_id) and payload ->> 'course_id' = sqlc.arg(course_id)::text;

-- name: RegressionAIScores :one
select coalesce(ai_scores, '{}'::jsonb) as ai_scores from grading_results where course_id = sqlc.arg(course_id) and id = sqlc.arg(id);

-- name: RegressionRunCases :many
-- Ca chấm cùng nội dung ngay trong DB cho `cmd/regress run` (môi trường đo). Văn bản không bao giờ ra khỏi DB trừ khi chạy `export`.
select c.id, c.kind, c.source_id, c.criterion_id, c.expected, s.extracted_text, a.id as assignment_id, a.title, a.description, a.reference_answer, a.rubric, a.guidance_version
from grading_regression_cases c
join grading_results r on r.course_id = c.course_id and r.id = c.source_id
join submissions s on s.course_id = r.course_id and s.id = r.submission_id
join assignments a on a.course_id = s.course_id and a.id = s.assignment_id
where c.course_id = sqlc.arg(course_id) and c.kind in ('GRADING_EDIT', 'GRADING_APPEAL')
order by c.created_at, c.id limit sqlc.arg(page_limit);

-- name: RegressionThreadCases :many
-- Ca THREAD_* cho chế độ --e2: độ tự tin của bài AI bị sửa / loại so với ngưỡng chuyển giảng viên của lớp.
-- hidden-ok: công cụ vận hành ở môi trường đo (cmd/regress); chỉ đếm độ tự tin của bài đã bị sửa / loại, không hiển thị cho người dùng.
select c.kind, p.confidence, co.escalation_threshold
from grading_regression_cases c
join forum_posts p on p.course_id = c.course_id and p.id = c.source_id
join courses co on co.id = c.course_id
where c.course_id = sqlc.arg(course_id) and c.kind in ('THREAD_CORRECTED', 'THREAD_REJECTED')
order by c.created_at, c.id limit sqlc.arg(page_limit);
