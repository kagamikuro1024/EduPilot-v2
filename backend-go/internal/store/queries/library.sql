-- US-P8-02 — thư viện của sinh viên: READY + visible_to_students + không ANSWER_KEY, của lớp hoặc được chia sẻ vào lớp.

-- name: LibList :many
-- `q` (≥ 2 ký tự — dưới đó người gọi truyền NULL): khớp tên / tệp / chủ đề (vn_fold, LIKE) HOẶC từ khoá trên đoạn (tsv). snippet = ts_headline của đoạn khớp đầu tiên.
select d.id, d.title, d.type, d.filename, d.mime_type, d.category, d.week_no, d.updated_at, d.page_count,
       (d.use_for_rag and exists (select 1 from content_chunks c where c.document_id = d.id and c.course_ids @> array[sqlc.arg(course_id)::uuid] and c.embedding is not null))::bool as can_ask_ai,
       case when sqlc.narg(q)::text is null then '' else coalesce((
         select left(regexp_replace(c.text, '\s+', ' ', 'g'), 160) from content_chunks c
         where c.document_id = d.id and c.course_ids @> array[sqlc.arg(course_id)::uuid] and c.tsv @@ vn_bigram_query(sqlc.narg(q)::text) order by c.ord limit 1), '') end::text as snippet
from documents d
where (d.course_id = sqlc.arg(course_id) or exists (select 1 from document_courses dc where dc.document_id = d.id and dc.course_id = sqlc.arg(course_id)))
  and d.status = 'READY' and d.visible_to_students and d.type <> 'ANSWER_KEY'
  and (sqlc.narg(type)::text is null or d.type::text = sqlc.narg(type)::text)
  and (sqlc.narg(week_no)::int is null or d.week_no = sqlc.narg(week_no)::int)
  and (sqlc.narg(category)::text is null or d.category = sqlc.narg(category)::text)
  and (sqlc.narg(q)::text is null
       or vn_fold(d.title) like '%' || vn_fold(sqlc.narg(q)::text) || '%' escape '\'
       or vn_fold(coalesce(d.filename, '')) like '%' || vn_fold(sqlc.narg(q)::text) || '%' escape '\'
       or vn_fold(coalesce(d.category, '')) like '%' || vn_fold(sqlc.narg(q)::text) || '%' escape '\'
       or exists (select 1 from content_chunks c where c.document_id = d.id and c.course_ids @> array[sqlc.arg(course_id)::uuid] and c.tsv @@ vn_bigram_query(sqlc.narg(q)::text)))
  and (sqlc.narg(cursor_at)::timestamptz is null or (d.updated_at, d.id) < (sqlc.narg(cursor_at)::timestamptz, sqlc.narg(cursor_id)::uuid))
order by d.updated_at desc, d.id desc
limit sqlc.arg(page_limit);

-- name: LibGet :one
select d.id, d.title, d.type, d.filename, d.mime_type, d.size_bytes, d.category, d.week_no, d.updated_at, d.page_count, d.blob_key,
       (d.use_for_rag and exists (select 1 from content_chunks c where c.document_id = d.id and c.course_ids @> array[sqlc.arg(course_id)::uuid] and c.embedding is not null))::bool as can_ask_ai
from documents d
where d.id = sqlc.arg(id)
  and (d.course_id = sqlc.arg(course_id) or exists (select 1 from document_courses dc where dc.document_id = d.id and dc.course_id = sqlc.arg(course_id)))
  and d.status = 'READY' and d.visible_to_students and d.type <> 'ANSWER_KEY';

-- name: LibCountDownload :exec
update documents set download_count = download_count + 1 where id = sqlc.arg(id);

-- name: LibSearchTool :many
-- Cho tool search_library: tối đa 5 tài liệu cùng bộ lọc của danh sách, kèm đoạn khớp đầu tiên (trang).
select d.id, d.title, d.type, d.week_no,
       (select c.page_no from content_chunks c where c.document_id = d.id and c.course_ids @> array[sqlc.arg(course_id)::uuid] and c.tsv @@ vn_bigram_query(sqlc.arg(q)::text) order by c.ord limit 1)::int as page_no,
       coalesce((select left(regexp_replace(c.text, '\s+', ' ', 'g'), 160) from content_chunks c where c.document_id = d.id and c.course_ids @> array[sqlc.arg(course_id)::uuid] and c.tsv @@ vn_bigram_query(sqlc.arg(q)::text) order by c.ord limit 1), '')::text as snippet
from documents d
where (d.course_id = sqlc.arg(course_id) or exists (select 1 from document_courses dc where dc.document_id = d.id and dc.course_id = sqlc.arg(course_id)))
  and d.status = 'READY' and d.visible_to_students and d.type <> 'ANSWER_KEY'
  and (vn_fold(d.title) like '%' || vn_fold(sqlc.arg(q)::text) || '%' escape '\'
       or exists (select 1 from content_chunks c where c.document_id = d.id and c.course_ids @> array[sqlc.arg(course_id)::uuid] and c.tsv @@ vn_bigram_query(sqlc.arg(q)::text)))
order by d.updated_at desc, d.id desc
limit 5;
