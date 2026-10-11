-- PoC tìm lai trên PostgreSQL 18 + pgvector (image pgvector/pgvector:pg18), container tạm `poc-pg`.
-- Bảng rút gọn từ 00003 content_chunks; vn_fold chép nguyên từ 00005. Vector ngẫu nhiên = trường hợp xấu nhất cho HNSW.
-- Chạy: docker exec -i poc-pg psql -U postgres -v ON_ERROR_STOP=1 < hybrid.sql
\timing off
CREATE EXTENSION IF NOT EXISTS vector;
SELECT extversion AS pgvector FROM pg_extension WHERE extname = 'vector';
SELECT version();

CREATE FUNCTION vn_fold(text) RETURNS text
    LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT
    AS $$ SELECT translate(lower($1), 'àáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđ', 'aaaaaaaaaaaaaaaaaeeeeeeeeeeeiiiiiooooooooooooooooouuuuuuuuuuuyyyyyd') $$;

-- 1. Độ biến động của các hàm định dùng trong index biểu thức
SELECT p.oid::regprocedure AS fn, p.provolatile AS volatile_i_s_v
FROM pg_proc p WHERE p.proname IN ('normalize', 'vn_fold') OR p.oid = 'to_tsvector(regconfig,text)'::regprocedure;

-- 2. vn_fold với chữ dựng sẵn (NFC) và chữ tổ hợp (NFD): NFD còn dấu tổ hợp → không khớp
SELECT vn_fold('Học vụ ĐIỀU 5') AS nfc,
       vn_fold(normalize('Học vụ ĐIỀU 5', NFD)) AS nfd,
       vn_fold(normalize(normalize('Học vụ ĐIỀU 5', NFD), NFC)) AS nfd_then_nfc;

-- 3. Tách token của cấu hình simple trên chữ đã gập dấu; websearch_to_tsquery cho cụm từ có ngoặc kép
SELECT to_tsvector('simple', vn_fold('Cảnh báo học vụ: sinh viên bị cảnh báo khi điểm TBC < 1,0')) AS tsv;
SELECT websearch_to_tsquery('simple', vn_fold('"cảnh báo học vụ"')) AS phrase_q,
       plainto_tsquery('simple', vn_fold('canh bao hoc vu')) AS and_q;

CREATE TYPE chunk_audience AS ENUM ('ALL', 'STAFF', 'GRADING');
CREATE TABLE content_chunks (
    id          bigserial PRIMARY KEY,
    document_id integer        NOT NULL,
    course_ids  uuid[]         NOT NULL,
    audience    chunk_audience NOT NULL DEFAULT 'ALL',
    ord         integer        NOT NULL,
    text        text           NOT NULL,
    embedding   vector(1536)
);
CREATE INDEX content_chunks_course_ids_gin ON content_chunks USING gin (course_ids);

CREATE FUNCTION rand_vec() RETURNS vector LANGUAGE sql VOLATILE
    AS $$ SELECT array_agg(random() - 0.5)::vector(1536) FROM generate_series(1, 1536) $$;

-- T1 (SYSTEM_DESIGN mục 1): 100 tài liệu × 200 chunk = 20.000 chunk; 20 lớp, mỗi tài liệu thuộc 1 lớp; 10% chunk STAFF / GRADING
CREATE TABLE courses AS SELECT gen_random_uuid() AS id, g AS n FROM generate_series(1, 20) g;
\timing on
INSERT INTO content_chunks (document_id, course_ids, audience, ord, text, embedding)
SELECT d, ARRAY[(SELECT id FROM courses WHERE n = 1 + d % 20)],
       (CASE WHEN o % 10 = 0 THEN 'STAFF' WHEN o % 10 = 5 THEN 'GRADING' ELSE 'ALL' END)::chunk_audience,
       o,
       (SELECT string_agg(w, ' ') FROM (SELECT (ARRAY['sinh','viên','điểm','môn','học','phần','tín','chỉ','giảng','bài','tập',
            'kiểm','tra','mạng','máy','tính','bảo','mật','dữ','liệu','chương','trình','thi','cuối','kỳ','quy','định',
            'trường','lớp','đề','cương','tài','liệu','thời','gian','nộp','hạn','chót'])[1 + floor(random() * 38)::int] AS w
            FROM generate_series(1, 150 + 0 * d * o)) s),  -- tham chiếu d, o để Postgres sinh lại chữ cho từng dòng
       rand_vec()
FROM generate_series(1, 100) d, generate_series(0, 199) o;
\timing off
ANALYZE content_chunks;
SELECT count(*) AS chunks, pg_size_pretty(pg_total_relation_size('content_chunks')) AS size FROM content_chunks;

-- Chunk thật (câu chữ quy chế OCR) để kiểm tra từ khoá không dấu: một chunk chứa cụm "cảnh báo học vụ"
INSERT INTO content_chunks (document_id, course_ids, ord, text, embedding)
SELECT 1000, ARRAY[(SELECT id FROM courses WHERE n = 1)], 0,
       'Điều 5. Cảnh báo học vụ. Sinh viên bị cảnh báo học vụ nếu điểm trung bình chung học kỳ dưới 1,0.', rand_vec();

-- 4. Index từ khoá: biểu thức, không cần cột mới (00003 không ALTER)
\timing on
CREATE INDEX content_chunks_fts_gin ON content_chunks USING gin (to_tsvector('simple', vn_fold(text)));
\timing off
SELECT pg_size_pretty(pg_relation_size('content_chunks_fts_gin')) AS fts_index_size;

-- tham số truy vấn
SELECT id AS c1 FROM courses WHERE n = 1 \gset
SELECT rand_vec()::text AS qv \gset

-- 5a. Vector chính xác (không HNSW), lọc lớp + audience ngay trong truy vấn
EXPLAIN (ANALYZE, BUFFERS, COSTS OFF)
SELECT id FROM content_chunks
WHERE course_ids @> ARRAY[:'c1'::uuid] AND audience = 'ALL'
ORDER BY embedding <=> :'qv'::vector LIMIT 20;

-- 5b. Từ khoá không dấu khớp chữ có dấu
EXPLAIN (ANALYZE, COSTS OFF)
SELECT id, ts_rank_cd(to_tsvector('simple', vn_fold(text)), q) AS r
FROM content_chunks, websearch_to_tsquery('simple', vn_fold('"canh bao hoc vu"')) q
WHERE course_ids @> ARRAY[:'c1'::uuid] AND audience = 'ALL'
  AND to_tsvector('simple', vn_fold(text)) @@ q
ORDER BY r DESC LIMIT 20;
SELECT id, left(text, 60) FROM content_chunks
WHERE to_tsvector('simple', vn_fold(text)) @@ websearch_to_tsquery('simple', vn_fold('"canh bao hoc vu"'));

-- 6. RRF (k = 60) trong một câu SQL, lọc lớp + audience ở cả hai nhánh
PREPARE hybrid(uuid, vector, text) AS
WITH sem AS (
    SELECT id, row_number() OVER (ORDER BY embedding <=> $2) AS rnk
    FROM content_chunks
    WHERE course_ids @> ARRAY[$1] AND audience = 'ALL'
    ORDER BY embedding <=> $2 LIMIT 40
), kw AS (
    SELECT id, row_number() OVER (ORDER BY ts_rank_cd(to_tsvector('simple', vn_fold(text)), q) DESC) AS rnk
    FROM content_chunks, websearch_to_tsquery('simple', vn_fold($3)) q
    WHERE course_ids @> ARRAY[$1] AND audience = 'ALL' AND to_tsvector('simple', vn_fold(text)) @@ q
    ORDER BY ts_rank_cd(to_tsvector('simple', vn_fold(text)), q) DESC LIMIT 40
)
SELECT coalesce(sem.id, kw.id) AS id,
       coalesce(1.0 / (60 + sem.rnk), 0) + coalesce(1.0 / (60 + kw.rnk), 0) AS score
FROM sem FULL JOIN kw ON sem.id = kw.id
ORDER BY score DESC LIMIT 8;
EXPLAIN (ANALYZE, COSTS OFF) EXECUTE hybrid(:'c1', :'qv', '"canh bao hoc vu"');
EXECUTE hybrid(:'c1', :'qv', '"canh bao hoc vu"');

-- 7. HNSW + lọc: độ phủ so với kết quả chính xác (ef_search mặc định 40, lọc 5% số dòng)
CREATE TEMP TABLE exact AS
SELECT id FROM content_chunks WHERE course_ids @> ARRAY[:'c1'::uuid] AND audience = 'ALL'
ORDER BY embedding <=> :'qv'::vector LIMIT 20;
\timing on
CREATE INDEX content_chunks_embedding_hnsw ON content_chunks
    USING hnsw ((embedding::halfvec(1536)) halfvec_cosine_ops) WITH (m = 16, ef_construction = 64);
\timing off
SELECT pg_size_pretty(pg_relation_size('content_chunks_embedding_hnsw')) AS hnsw_size;
SET hnsw.iterative_scan = off;
SELECT count(*) AS hnsw_rows_no_iter, count(*) FILTER (WHERE id IN (SELECT id FROM exact)) AS hits FROM (
    SELECT id FROM content_chunks WHERE course_ids @> ARRAY[:'c1'::uuid] AND audience = 'ALL'
    ORDER BY embedding::halfvec(1536) <=> :'qv'::halfvec(1536) LIMIT 20) t;
SET hnsw.iterative_scan = relaxed_order;
EXPLAIN (ANALYZE, COSTS OFF)
SELECT id FROM content_chunks WHERE course_ids @> ARRAY[:'c1'::uuid] AND audience = 'ALL'
ORDER BY embedding::halfvec(1536) <=> :'qv'::halfvec(1536) LIMIT 20;
SELECT count(*) AS hnsw_rows_iter, count(*) FILTER (WHERE id IN (SELECT id FROM exact)) AS hits FROM (
    SELECT id FROM content_chunks WHERE course_ids @> ARRAY[:'c1'::uuid] AND audience = 'ALL'
    ORDER BY embedding::halfvec(1536) <=> :'qv'::halfvec(1536) LIMIT 20) t;

-- 8. Câu hỏi tự nhiên: AND mọi âm tiết / OR mọi âm tiết / OR các cặp âm tiết liền nhau (phrase)
\set q 'điều kiện bị cảnh báo học vụ là gì'
SELECT count(*) AS and_hits FROM content_chunks
WHERE to_tsvector('simple', vn_fold(text)) @@ plainto_tsquery('simple', vn_fold(:'q'));
WITH q AS (SELECT to_tsquery('simple', array_to_string(tsvector_to_array(to_tsvector('simple', vn_fold(normalize(:'q', NFC)))), ' | ')) AS q)
SELECT count(*) AS or_hits_course1,
       (SELECT r FROM (SELECT id, rank() OVER (ORDER BY ts_rank_cd(to_tsvector('simple', vn_fold(text)), q.q) DESC) r
                       FROM content_chunks, q WHERE course_ids @> ARRAY[:'c1'::uuid] AND to_tsvector('simple', vn_fold(text)) @@ q.q) t
        WHERE id = 20001) AS rank_of_target
FROM content_chunks, q WHERE course_ids @> ARRAY[:'c1'::uuid] AND to_tsvector('simple', vn_fold(text)) @@ q.q;
CREATE FUNCTION vn_bigram_query(text) RETURNS tsquery LANGUAGE sql IMMUTABLE PARALLEL SAFE STRICT AS $$
  SELECT to_tsquery('simple', string_agg(a || ' <-> ' || b, ' | '))
  FROM (SELECT w AS a, lead(w) OVER (ORDER BY i) AS b
        FROM regexp_split_to_table(vn_fold(normalize($1, NFC)), '[^a-z0-9]+') WITH ORDINALITY t(w, i) WHERE w <> '') s
  WHERE b IS NOT NULL $$;
SELECT vn_bigram_query(:'q')::text AS bq \gset
SELECT count(*) AS bigram_hits_course1,
       (SELECT r FROM (SELECT id, rank() OVER (ORDER BY ts_rank_cd(to_tsvector('simple', vn_fold(text)), :'bq'::tsquery) DESC) r
                       FROM content_chunks WHERE course_ids @> ARRAY[:'c1'::uuid] AND to_tsvector('simple', vn_fold(text)) @@ :'bq'::tsquery) t
        WHERE id = 20001) AS rank_of_target
FROM content_chunks WHERE course_ids @> ARRAY[:'c1'::uuid] AND to_tsvector('simple', vn_fold(text)) @@ :'bq'::tsquery;

-- 9. Index biểu thức (phải tính lại tsvector khi recheck) so với cột tsvector lưu sẵn
\timing on
SELECT id FROM content_chunks WHERE course_ids @> ARRAY[:'c1'::uuid] AND audience = 'ALL' AND to_tsvector('simple', vn_fold(text)) @@ :'bq'::tsquery
ORDER BY ts_rank_cd(to_tsvector('simple', vn_fold(text)), :'bq'::tsquery) DESC LIMIT 40 \g /dev/null
\timing off
ALTER TABLE content_chunks ADD COLUMN tsv tsvector GENERATED ALWAYS AS (to_tsvector('simple', vn_fold(normalize(text, NFC)))) STORED;
CREATE INDEX content_chunks_tsv_gin ON content_chunks USING gin (tsv);
ANALYZE content_chunks;
\timing on
SELECT id FROM content_chunks WHERE course_ids @> ARRAY[:'c1'::uuid] AND audience = 'ALL' AND tsv @@ :'bq'::tsquery
ORDER BY ts_rank_cd(tsv, :'bq'::tsquery) DESC LIMIT 40 \g /dev/null
\timing off
