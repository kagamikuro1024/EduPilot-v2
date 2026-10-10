-- Tài liệu của lớp (US-P8-01: tải lên / hoàn tất / thử lại). Mọi truy vấn bắt đầu từ course_id (luật 13).

-- name: DocCourseStatus :one
select status from courses where id = sqlc.arg(id);

-- name: DocInsert :one
insert into documents (course_id, title, type, filename, mime_type, size_bytes, sha256, blob_key, status, visible_to_students, use_for_rag, category, week_no, uploaded_by)
values (sqlc.arg(course_id), sqlc.arg(title), sqlc.arg(type), sqlc.arg(filename), sqlc.arg(mime_type), sqlc.arg(size_bytes), sqlc.arg(sha256), sqlc.arg(blob_key),
        'QUEUED', sqlc.arg(visible_to_students), sqlc.arg(use_for_rag), sqlc.narg(category), sqlc.narg(week_no), sqlc.arg(uploaded_by))
returning *;

-- name: DocLockHash :exec
-- Khoá tư vấn trong giao dịch: hai complete song song cùng tệp (khác Idempotency-Key) → đúng một dòng documents.
select pg_advisory_xact_lock(hashtextextended(sqlc.arg(course_id)::text || sqlc.arg(sha256)::text, 0));

-- name: DocFindDuplicateInCourse :one
-- Cùng băm ở CÙNG lớp, kể cả tài liệu được chia sẻ vào lớp.
select d.id from documents d
where d.sha256 = sqlc.arg(sha256)
  and (d.course_id = sqlc.arg(course_id) or exists (select 1 from document_courses dc where dc.document_id = d.id and dc.course_id = sqlc.arg(course_id)))
order by d.created_at limit 1;

-- name: DocFindDuplicateElsewhere :one
-- Cùng băm ở lớp KHÁC mà người gọi là Staff ACTIVE (lớp không thuộc người gọi: coi như không trùng — không lộ).
select d.id, c.class_code
from documents d
join courses c on c.id = d.course_id
join enrollments e on e.course_id = d.course_id and e.user_id = sqlc.arg(user_id) and e.role_in_course in ('TEACHER', 'TA') and e.status = 'ACTIVE'
where d.sha256 = sqlc.arg(sha256) and d.course_id <> sqlc.arg(course_id)
order by d.created_at limit 1;

-- name: DocGetInCourse :one
-- Tài liệu thuộc lớp (của lớp hoặc được chia sẻ vào lớp).
select d.* from documents d
where d.id = sqlc.arg(id)
  and (d.course_id = sqlc.arg(course_id) or exists (select 1 from document_courses dc where dc.document_id = d.id and dc.course_id = sqlc.arg(course_id)));

-- name: DocRetry :one
-- FAILED → QUEUED (chỉ tài liệu của chính lớp này).
update documents set status = 'QUEUED', error = null
where id = sqlc.arg(id) and course_id = sqlc.arg(course_id) and status = 'FAILED'
returning *;
