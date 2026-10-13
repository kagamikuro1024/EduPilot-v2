-- Kho tri thức của lớp (US-P4-05 AC10–AC11, SRS FEAT-escalation-notify 4.7): MỘT dòng `documents` ảo mỗi lớp (category = 'THREAD_KNOWLEDGE'),
-- mỗi mục = một content_chunks + một knowledge_items. Tệp này và rag.sql là hai nơi DUY NHẤT đọc `documents` mà không loại tài liệu tri thức.

-- name: KnowledgeEnsureDoc :one
-- Tạo lười, an toàn khi song song (chỉ mục duy nhất documents_knowledge_key). Không trả dòng = đã có.
insert into documents (course_id, title, type, status, visible_to_students, use_for_rag, category)
values (sqlc.arg(course_id), sqlc.arg(title), 'OTHER', 'READY', true, true, 'THREAD_KNOWLEDGE')
on conflict (course_id) where category = 'THREAD_KNOWLEDGE' do nothing
returning id;

-- name: KnowledgeDocID :one
select id from documents where course_id = sqlc.arg(course_id) and category = 'THREAD_KNOWLEDGE';

-- name: KnowledgeItemBySource :one
select * from knowledge_items where source_type = sqlc.arg(source_type) and source_id = sqlc.arg(source_id);

-- name: KnowledgeNextOrd :one
select (coalesce(max(ord), -1) + 1)::int from content_chunks where document_id = sqlc.arg(document_id) and course_ids @> array[sqlc.arg(course_id)::uuid];

-- name: KnowledgeChunkInsert :one
-- embedding = NULL: RAG đã lọc `embedding IS NOT NULL` nên đoạn chưa nhúng không lọt ra kết quả tìm.
insert into content_chunks (document_id, course_ids, audience, ord, heading, text)
values (sqlc.arg(document_id), array[sqlc.arg(course_id)::uuid], 'ALL', sqlc.arg(ord), sqlc.arg(heading), sqlc.arg(text))
returning id;

-- name: KnowledgeChunkRewrite :exec
-- Lưu lại bản sửa: thay chữ, đưa embedding về NULL (nhúng lại bản mới).
update content_chunks set heading = sqlc.arg(heading), text = sqlc.arg(text), embedding = null, updated_at = now() where id = sqlc.arg(id) and course_ids @> array[sqlc.arg(course_id)::uuid];

-- name: KnowledgeItemInsert :one
insert into knowledge_items (course_id, source_type, source_id, chunk_id, created_by)
values (sqlc.arg(course_id), sqlc.arg(source_type), sqlc.arg(source_id), sqlc.arg(chunk_id), sqlc.narg(created_by))
returning *;

-- name: KnowledgeItemTouch :exec
update knowledge_items set updated_at = now(), created_by = coalesce(sqlc.narg(created_by), created_by) where id = sqlc.arg(id);

-- name: KnowledgeChunkDelete :one
-- Xoá đoạn (knowledge_items theo CASCADE); trả document_id để phát document.changed.
delete from content_chunks where id = sqlc.arg(id) returning document_id;

-- name: KnowledgeChunkForSync :one
select c.id, c.document_id, c.course_ids, c.text, md5(c.text)::text as digest, (c.embedding is null)::boolean as unembedded from content_chunks c where c.id = sqlc.arg(id) and c.course_ids @> array[sqlc.arg(course_id)::uuid];

-- name: KnowledgeSetEmbedding :execrows
-- Chỉ ghi khi chữ chưa đổi (bản sửa mới hơn thì bỏ qua: handler của bản đó sẽ nhúng).
update content_chunks set embedding = sqlc.arg(embedding), updated_at = now() where id = sqlc.arg(id) and course_ids @> array[sqlc.arg(course_id)::uuid] and md5(text) = sqlc.arg(digest)::text;

-- name: KnowledgeForTicket :one
-- Trạng thái mục tri thức của ticket (giao diện giảng viên: "Chưa nạp" khi embedding NULL).
select ki.id, ki.chunk_id, ki.updated_at, (c.embedding is not null)::boolean as loaded, c.text
from knowledge_items ki join content_chunks c on c.id = ki.chunk_id and c.course_ids @> array[sqlc.arg(course_id)::uuid]
where ki.source_type = 'TICKET' and ki.source_id = sqlc.arg(ticket_id) and ki.course_id = sqlc.arg(course_id);
