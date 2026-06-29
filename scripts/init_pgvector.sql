-- ====================================================================
-- FILE: scripts/init_pgvector.sql
-- MÔ TẢ: Script SQL chạy TỰ ĐỘNG khi PostgreSQL container khởi động
--         lần đầu tiên.
--
-- CÁCH HOẠT ĐỘNG:
--   Docker image ankane/pgvector tự động chạy mọi file .sql trong
--   thư mục /docker-entrypoint-initdb.d/ khi container lần đầu start.
--   File này được mount vào thư mục đó qua docker-compose.yml.
--
-- NỘI DUNG:
--   1. Kích hoạt extension pgvector
--   2. Tạo Enum trạng thái tài liệu
--   3. Tạo bảng documents (mục lục tài liệu)
--   4. Tạo bảng document_chunks (đoạn văn bản + vector embedding)
--   5. Tạo index tối ưu truy vấn
--
-- LƯU Ý QUAN TRỌNG:
--   Script này CHỈ chạy khi container khởi tạo LẦN ĐẦU (volume trống).
--   Nếu volume postgres_data đã tồn tại (đã khởi tạo trước đó),
--   script SẼ KHÔNG chạy lại.
--   Muốn chạy lại: docker-compose down -v (xóa volume) rồi up lại.
-- ====================================================================

-- -------------------------------------------------
-- 1. Kích hoạt extension pgvector
-- Biến PostgreSQL thành Vector Database
-- -------------------------------------------------
CREATE EXTENSION IF NOT EXISTS vector;

-- -------------------------------------------------
-- 2. Tạo Enum trạng thái xử lý tài liệu
-- PROCESSING = đang xử lý (cắt chunk, tạo embedding)
-- READY      = sẵn sàng tìm kiếm
-- FAILED     = xử lý thất bại
-- -------------------------------------------------
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'document_status') THEN
        CREATE TYPE document_status AS ENUM ('PROCESSING', 'READY', 'FAILED');
    END IF;
END
$$;

-- -------------------------------------------------
-- 3. Bảng documents: Mục lục tài liệu
-- Mỗi row = 1 file (VD: slide, đề thi, giáo trình)
-- -------------------------------------------------
CREATE TABLE IF NOT EXISTS documents (
    -- UUID tự sinh: đảm bảo ID duy nhất khi scale nhiều server
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Tên file gốc hoặc đường dẫn tài liệu (VD: "data/Bai3_VongLap.pdf")
    source_uri    VARCHAR NOT NULL,

    -- Tên file hiển thị (VD: "Bai3_VongLap.pdf")
    filename      VARCHAR(500) NOT NULL,

    -- Thông tin bổ sung dạng JSON linh hoạt
    metadata      JSONB DEFAULT '{}'::jsonb,

    -- Mã băm nội dung (SHA256)
    content_hash  VARCHAR NOT NULL,

    -- Phiên bản tài liệu
    version       INTEGER DEFAULT 1,

    -- Trạng thái xử lý hiện tại
    status        document_status DEFAULT 'PROCESSING',

    -- Thời điểm thêm vào hệ thống
    created_at    TIMESTAMPTZ DEFAULT now(),

    -- Chống duplicate tài liệu
    UNIQUE (source_uri, content_hash)
);

-- -------------------------------------------------
-- 4. Bảng document_chunks: Đoạn văn bản + Vector
-- Mỗi row = 1 đoạn nhỏ đã cắt từ tài liệu gốc
-- kèm vector embedding 1536 chiều
-- -------------------------------------------------
CREATE TABLE IF NOT EXISTS document_chunks (
    -- UUID cho mỗi chunk
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Liên kết về tài liệu gốc
    -- ON DELETE CASCADE = xóa tài liệu → chunk tự xóa theo
    document_id     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,

    -- Nội dung văn bản gốc
    content         TEXT NOT NULL,

    -- Vector embedding 1536 chiều (OpenAI text-embedding-3-small)
    -- Đây là "bản dịch" văn bản sang ngôn ngữ số học để so sánh ngữ nghĩa
    embedding       VECTOR(1536),

    -- Thứ tự chunk (0, 1, 2...) để ghép lại đúng trật tự
    chunk_index     INTEGER NOT NULL DEFAULT 0,

    -- Mã băm nội dung của đoạn chunk
    content_hash    VARCHAR NOT NULL,

    -- Metadata bổ sung cho chunk (VD: Tiêu đề H1, H2 của đoạn văn) - Action 1: HOTFIX-005
    metadata        JSONB DEFAULT '{}'::jsonb,

    -- Chống duplicate chunk trong cùng 1 tài liệu
    UNIQUE (document_id, content_hash)
);

-- -------------------------------------------------
-- 5. Tạo Index tối ưu hiệu suất
-- -------------------------------------------------

-- Tìm tài liệu theo trạng thái (VD: WHERE status = 'READY')
CREATE INDEX IF NOT EXISTS idx_documents_status
    ON documents (status);

-- Tìm kiếm trong metadata JSON (dùng GIN index)
CREATE INDEX IF NOT EXISTS idx_documents_metadata
    ON documents USING GIN (metadata);

-- Tăng tốc JOIN chunks với documents
CREATE INDEX IF NOT EXISTS idx_chunks_document_id
    ON document_chunks (document_id);

-- Index vector dùng HNSW (Hierarchical Navigable Small World)
-- Cung cấp hiệu năng và độ chính xác vượt trội so với IVFFlat
CREATE INDEX IF NOT EXISTS idx_chunks_embedding
    ON document_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);

-- -------------------------------------------------
-- Hoàn tất! Ghi log xác nhận
-- -------------------------------------------------
DO $$
BEGIN
    RAISE NOTICE '====================================';
    RAISE NOTICE 'Schema khởi tạo thành công!';
    RAISE NOTICE '  ✓ Extension pgvector: OK';
    RAISE NOTICE '  ✓ Bảng documents: OK';
    RAISE NOTICE '  ✓ Bảng document_chunks: OK';
    RAISE NOTICE '  ✓ Indexes: OK';
    RAISE NOTICE '====================================';
END
$$;
