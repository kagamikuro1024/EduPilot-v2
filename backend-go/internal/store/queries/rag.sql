-- Truy xuất tất định cho hỏi–đáp (US-P8-01, SRS FEAT-docs-calendar 4.3). MỌI điều kiện (lớp, READY, cờ, audience, không ANSWER_KEY) nằm TRONG từng nhánh,
-- trước ORDER BY … LIMIT: lọc sau khi lấy láng giềng gần nhất sẽ trả thiếu khi 40 láng giềng đầu đều bị cấm.
-- Không có truy vấn nào cho audience 'GRADING' (P7 tạo gói riêng).

-- name: RagSearch :many
WITH q AS (
    SELECT sqlc.arg(vec)::vector AS v,
           CASE WHEN sqlc.arg(query_text)::text LIKE '%"%' THEN websearch_to_tsquery('simple', vn_fold(normalize(sqlc.arg(query_text)::text, NFC)))
                ELSE vn_bigram_query(sqlc.arg(query_text)::text) END AS t
),
vec AS (
    SELECT c.id, ROW_NUMBER() OVER (ORDER BY c.embedding <=> q.v) AS r
    FROM content_chunks c JOIN documents d ON d.id = c.document_id, q
    WHERE c.course_ids @> ARRAY[sqlc.arg(course_id)::uuid]
      AND d.status = 'READY' AND d.use_for_rag AND d.type <> 'ANSWER_KEY'
      AND (sqlc.arg(student_only)::bool = false OR d.visible_to_students)
      AND c.audience::text = ANY(sqlc.arg(audiences)::text[]) AND c.embedding IS NOT NULL
      AND (cardinality(sqlc.arg(document_ids)::text[]::uuid[]) = 0 OR c.document_id = ANY(sqlc.arg(document_ids)::text[]::uuid[]))
    ORDER BY c.embedding <=> q.v
    LIMIT 40
),
kw AS (
    SELECT c.id, ROW_NUMBER() OVER (ORDER BY ts_rank_cd(c.tsv, q.t) DESC) AS r
    FROM content_chunks c JOIN documents d ON d.id = c.document_id, q
    WHERE c.course_ids @> ARRAY[sqlc.arg(course_id)::uuid]
      AND d.status = 'READY' AND d.use_for_rag AND d.type <> 'ANSWER_KEY'
      AND (sqlc.arg(student_only)::bool = false OR d.visible_to_students)
      AND c.audience::text = ANY(sqlc.arg(audiences)::text[]) AND c.embedding IS NOT NULL
      AND (cardinality(sqlc.arg(document_ids)::text[]::uuid[]) = 0 OR c.document_id = ANY(sqlc.arg(document_ids)::text[]::uuid[]))
      AND c.tsv @@ q.t
    ORDER BY ts_rank_cd(c.tsv, q.t) DESC
    LIMIT 40
),
fused AS (
    SELECT x.id, SUM(1.0 / (60 + x.r)) AS s
    FROM (SELECT vec.id, vec.r FROM vec UNION ALL SELECT kw.id, kw.r FROM kw) x
    GROUP BY x.id ORDER BY s DESC LIMIT sqlc.arg(k)
)
SELECT c.id AS chunk_id, c.document_id, d.title, c.page_no, c.heading, c.text,
       (1 - (c.embedding <=> (SELECT v FROM q)))::float8 AS cosine, f.s::float8 AS score
FROM fused f JOIN content_chunks c ON c.id = f.id JOIN documents d ON d.id = c.document_id
WHERE c.course_ids @> ARRAY[sqlc.arg(course_id)::uuid]
ORDER BY f.s DESC;
