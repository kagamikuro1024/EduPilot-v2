-- Nạp tài liệu và lập chỉ mục lại (US-P8-01, SRS FEAT-docs-calendar 4.2). Mọi đường ĐỌC content_chunks lọc theo course_ids.

-- name: IngestClaim :one
-- Nhận việc: QUEUED, hoặc PROCESSING mà nhịp gia hạn cuối đã quá `idle` giây; tối đa MỘT tài liệu PROCESSING (còn hạn) mỗi lớp. Không trả dòng = không nhận được.
update documents d
set status = 'PROCESSING', updated_at = now(), error = null
where d.id = sqlc.arg(id)
  and (d.status = 'QUEUED' or (d.status = 'PROCESSING' and d.updated_at < now() - make_interval(secs => sqlc.arg(idle_secs)::float8)))
  and not exists (select 1 from documents x where x.course_id = d.course_id and x.status = 'PROCESSING' and x.id <> d.id
                  and x.updated_at > now() - make_interval(secs => sqlc.arg(idle_secs)::float8))
returning d.id, d.course_id, d.title, d.type, d.filename, d.mime_type, d.blob_key, d.sha256, d.visible_to_students, d.use_for_rag, d.uploaded_by, d.version;

-- name: IngestRenewLease :execrows
update documents set updated_at = now() where id = sqlc.arg(id) and status = 'PROCESSING';

-- name: IngestDocumentState :one
select id, course_id, status, error, sha256, use_for_rag, uploaded_by from documents where id = sqlc.arg(id);

-- name: IngestDeleteChunks :exec
delete from content_chunks where document_id = sqlc.arg(document_id) and course_ids @> array[sqlc.arg(course_id)::uuid];

-- name: IngestInsertChunk :exec
-- Chữ chuẩn hoá NFC ở DB (không có thư viện chuẩn hoá trong Go). course_ids = lớp chủ + lớp được chia sẻ.
insert into content_chunks (document_id, course_ids, audience, ord, page_no, heading, text, embedding)
values (sqlc.arg(document_id),
        (select array_agg(c) from (select sqlc.arg(course_id)::uuid as c union select dc.course_id from document_courses dc where dc.document_id = sqlc.arg(document_id)) s),
        sqlc.arg(audience)::chunk_audience, sqlc.arg(ord), sqlc.narg(page_no), normalize(sqlc.narg(heading)::text, NFC), normalize(sqlc.arg(text)::text, NFC), sqlc.narg(embedding));

-- name: IngestMarkReady :execrows
-- Chỉ khi tài liệu còn PROCESSING và băm không đổi (không ghi đè tệp mới hơn).
update documents set status = 'READY', page_count = sqlc.arg(page_count), error = null
where id = sqlc.arg(id) and status = 'PROCESSING' and sha256 is not distinct from sqlc.narg(sha256)::char(64);

-- name: IngestMarkFailed :execrows
update documents set status = 'FAILED', error = sqlc.arg(error)
where id = sqlc.arg(id) and status = 'PROCESSING';

-- name: ListChunkTexts :many
-- Đọc chữ đã lưu để lập chỉ mục lại (không gọi docling). Lọc theo tài liệu VÀ lớp.
select c.id, c.ord, c.heading, c.text
from content_chunks c
where c.document_id = sqlc.arg(document_id) and c.course_ids @> array[sqlc.arg(course_id)::uuid]
order by c.ord;

-- name: SetChunkEmbedding :execrows
update content_chunks set embedding = sqlc.arg(embedding)
where id = sqlc.arg(id) and document_id = sqlc.arg(document_id) and course_ids @> array[sqlc.arg(course_id)::uuid];

-- name: ListCourseDocumentsForReindex :many
select id from documents where course_id = sqlc.arg(course_id) and status = 'READY' and use_for_rag order by id;
