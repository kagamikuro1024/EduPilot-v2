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

-- US-P8-02 — quản lý tài liệu của Staff.

-- name: DocList :many
-- Tài liệu của lớp + được chia sẻ vào lớp (shared_from = mã lớp gốc, chỉ đọc). Lọc type / status / q (vn_fold) ; keyset (updated_at, id).
select d.id, d.course_id, d.title, d.type, d.filename, d.mime_type, d.size_bytes, d.status, d.error, d.page_count, d.visible_to_students, d.use_for_rag,
       d.category, d.week_no, d.version, d.created_at, d.updated_at,
       case when d.course_id = sqlc.arg(course_id) then '' else c.class_code end::text as shared_from
from documents d join courses c on c.id = d.course_id
where (d.course_id = sqlc.arg(course_id) or exists (select 1 from document_courses dc where dc.document_id = d.id and dc.course_id = sqlc.arg(course_id)))
  and (sqlc.narg(type)::text is null or d.type::text = sqlc.narg(type)::text)
  and (sqlc.narg(status)::text is null or d.status::text = sqlc.narg(status)::text)
  and (sqlc.narg(q)::text is null or vn_fold(d.title) like '%' || vn_fold(sqlc.narg(q)::text) || '%' escape '\' or vn_fold(coalesce(d.filename, '')) like '%' || vn_fold(sqlc.narg(q)::text) || '%' escape '\')
  and (sqlc.narg(cursor_at)::timestamptz is null or (d.updated_at, d.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by d.updated_at desc, d.id desc
limit sqlc.arg(page_limit);

-- name: DocGetForUpdate :one
-- Khoá hàng để PATCH / xoá: chỉ tài liệu của CHÍNH lớp (chia sẻ vào → không thấy ở đây; người gọi tự phân biệt bằng DocGetInCourse).
select * from documents where id = sqlc.arg(id) and course_id = sqlc.arg(course_id) for update;

-- name: DocUpdate :one
update documents set title = sqlc.arg(title), type = sqlc.arg(type), category = sqlc.narg(category), week_no = sqlc.narg(week_no),
       use_for_rag = sqlc.arg(use_for_rag), visible_to_students = sqlc.arg(visible_to_students), version = version + 1
where id = sqlc.arg(id) and course_id = sqlc.arg(course_id)
returning *;

-- name: DocSetChunkAudience :execrows
update content_chunks c set audience = sqlc.arg(new_audience) where c.document_id = sqlc.arg(doc_id) and c.course_ids @> array[sqlc.arg(course_id)::uuid] and c.audience is distinct from sqlc.arg(new_audience);

-- name: DocDelete :execrows
delete from documents where id = sqlc.arg(id) and course_id = sqlc.arg(course_id);

-- name: DocDeleteImpact :one
-- Số đoạn và số lớp khác đang dùng tài liệu (chia sẻ) — cho câu xác nhận xoá.
select (select count(*) from content_chunks c where c.document_id = sqlc.arg(id)::uuid and c.course_ids @> array[sqlc.arg(course_id)::uuid])::bigint as chunks,
       (select count(*) from document_courses dc where dc.document_id = sqlc.arg(id)::uuid)::bigint as courses;

-- name: DocChunks :many
select c.id, c.ord, c.page_no, c.heading, c.text, c.audience, (c.embedding is not null)::bool as embedded from content_chunks c
where c.document_id = sqlc.arg(doc_id) and c.course_ids @> array[sqlc.arg(course_id)::uuid] and c.ord > sqlc.arg(after_ord)
order by c.ord limit sqlc.arg(page_limit);

-- name: DocChunkLock :one
select c.id, c.document_id, c.text, c.audience from content_chunks c where c.id = sqlc.arg(id) and c.document_id = sqlc.arg(doc_id) and c.course_ids @> array[sqlc.arg(course_id)::uuid] for update;

-- name: DocChunkSetText :one
update content_chunks c set text = sqlc.arg(new_text), embedding = sqlc.narg(new_embedding), token_count = null
where c.id = sqlc.arg(id) and c.document_id = sqlc.arg(doc_id) and c.course_ids @> array[sqlc.arg(course_id)::uuid]
returning id, ord, page_no, heading, text, audience;

-- name: DocTouch :exec
update documents set updated_at = now(), version = version + 1 where id = sqlc.arg(id);

-- name: DocStats :one
-- Một truy vấn tổng hợp (tài liệu của lớp + chia sẻ vào).
with docs as (
  select d.* from documents d
  where d.course_id = sqlc.arg(course_id) or exists (select 1 from document_courses dc where dc.document_id = d.id and dc.course_id = sqlc.arg(course_id))
), ch as (
  select count(*) as chunks, count(embedding) as embedded from content_chunks c join docs on docs.id = c.document_id where c.course_ids @> array[sqlc.arg(course_id)::uuid]
)
select (select count(*) from docs)::bigint as total,
       (select count(*) from docs where status in ('QUEUED', 'PROCESSING'))::bigint as processing,
       (select count(*) from docs where status = 'FAILED')::bigint as failed,
       (select count(*) from docs where status = 'READY')::bigint as ready,
       (select count(*) from docs where status = 'READY' and type = 'COURSE_POLICY')::bigint as policy_ready,
       (select chunks from ch)::bigint as chunks, (select embedded from ch)::bigint as embedded_chunks,
       coalesce((select sum(page_count) from docs), 0)::bigint as pages, coalesce((select sum(size_bytes) from docs), 0)::bigint as bytes,
       coalesce((select max(created_at) from docs), 'epoch'::timestamptz)::timestamptz as last_upload_at;

-- name: DocStatsByType :many
select d.type::text as k, count(*)::bigint as n from documents d
where d.course_id = sqlc.arg(course_id) or exists (select 1 from document_courses dc where dc.document_id = d.id and dc.course_id = sqlc.arg(course_id))
group by d.type;

-- name: DocStatsByStatus :many
select d.status::text as k, count(*)::bigint as n from documents d
where d.course_id = sqlc.arg(course_id) or exists (select 1 from document_courses dc where dc.document_id = d.id and dc.course_id = sqlc.arg(course_id))
group by d.status;
