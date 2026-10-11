-- Tìm từ khoá trên đoạn tài liệu (SRS FEAT-docs-calendar 5.7, proposals #7): cột `tsv` lưu sẵn + GIN, và hàm dựng truy vấn theo cặp âm tiết liền nhau.
-- Chủ dự án đồng ý `ALTER TABLE content_chunks ADD COLUMN` (D45 "tạo một lần" bị phá có chủ ý); không đụng cột có sẵn.
-- `normalize(…, NFC)` là bắt buộc: `vn_fold` trên chữ tổ hợp NFD để lọt dấu.

-- +goose Up
-- vn_bigram_query: "cảnh báo học vụ" → 'canh <-> bao | bao <-> hoc | hoc <-> vu'; một âm tiết → chính nó; rỗng → truy vấn rỗng (không khớp gì).
-- Âm tiết chỉ giữ chữ-số: không thể chèn cú pháp tsquery từ câu hỏi của người dùng. Tối đa 40 âm tiết đầu.
-- +goose StatementBegin
CREATE FUNCTION vn_bigram_query(q text) RETURNS tsquery
    LANGUAGE plpgsql IMMUTABLE PARALLEL SAFE STRICT
    AS $$
DECLARE
    w text[];
    parts text[] := '{}';
    i int;
BEGIN
    w := (SELECT coalesce(array_agg(x), '{}') FROM (
            SELECT x FROM regexp_split_to_table(regexp_replace(vn_fold(normalize(q, NFC)), '[^[:alnum:]]+', ' ', 'g'), ' ') AS x WHERE x <> '' LIMIT 40) s);
    IF cardinality(w) = 0 THEN
        RETURN ''::tsquery;
    ELSIF cardinality(w) = 1 THEN
        RETURN to_tsquery('simple', w[1]);
    END IF;
    FOR i IN 1 .. cardinality(w) - 1 LOOP
        parts := parts || (w[i] || ' <-> ' || w[i + 1]);
    END LOOP;
    RETURN to_tsquery('simple', array_to_string(parts, ' | '));
END
$$;
-- +goose StatementEnd

ALTER TABLE content_chunks ADD COLUMN tsv tsvector GENERATED ALWAYS AS (to_tsvector('simple', vn_fold(normalize(text, NFC)))) STORED;
CREATE INDEX content_chunks_tsv_gin ON content_chunks USING gin (tsv);

-- +goose Down
DROP INDEX IF EXISTS content_chunks_tsv_gin;
ALTER TABLE content_chunks DROP COLUMN IF EXISTS tsv;
DROP FUNCTION IF EXISTS vn_bigram_query(text);
