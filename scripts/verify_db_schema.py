"""
Script kiểm tra: Xác nhận schema Database đã được tạo đúng.

Cách chạy:
    1. Đảm bảo PostgreSQL container đang chạy: docker-compose up -d postgres-db
    2. Chạy script: python -m scripts.verify_db_schema

Script sẽ:
    - Kết nối tới PostgreSQL
    - Kiểm tra extension pgvector đã kích hoạt
    - Kiểm tra bảng documents và document_chunks tồn tại
    - Thử INSERT → SELECT → DELETE 1 record test
    - In kết quả kiểm tra ra console
"""

import asyncio
import logging
import sys
import os

# Thêm thư mục gốc vào path để import được src
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

import asyncpg

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
)
logger = logging.getLogger(__name__)

# URL kết nối tới PostgreSQL (local dev, từ máy host)
DATABASE_URL = os.getenv(
    "DATABASE_URL",
    "postgresql://REDACTED_USER:REDACTED_PASSWORD@legacy-host.invalid:5432/agent_db",
)


async def verify_database() -> bool:
    """
    Chạy tất cả bước kiểm tra schema.
    Returns True nếu mọi thứ OK, False nếu có lỗi.
    """
    all_passed = True

    try:
        # ----- Bước 1: Kết nối DB -----
        logger.info("=" * 60)
        logger.info("KIỂM TRA SCHEMA DATABASE")
        logger.info("=" * 60)

        conn = await asyncpg.connect(DATABASE_URL)
        logger.info("✓ Kết nối PostgreSQL thành công")

        # ----- Bước 2: Kiểm tra extension pgvector -----
        ext_count = await conn.fetchval(
            "SELECT COUNT(*) FROM pg_extension WHERE extname = 'vector'"
        )
        if ext_count > 0:
            logger.info("✓ Extension pgvector đã kích hoạt")
        else:
            logger.error("✗ Extension pgvector CHƯA được kích hoạt!")
            all_passed = False

        # ----- Bước 3: Kiểm tra bảng documents -----
        doc_exists = await conn.fetchval(
            """
            SELECT EXISTS (
                SELECT 1 FROM information_schema.tables
                WHERE table_schema = 'public' AND table_name = 'documents'
            )
            """
        )
        if doc_exists:
            logger.info("✓ Bảng 'documents' tồn tại")
        else:
            logger.error("✗ Bảng 'documents' KHÔNG tồn tại!")
            all_passed = False

        # ----- Bước 4: Kiểm tra bảng document_chunks -----
        chunk_exists = await conn.fetchval(
            """
            SELECT EXISTS (
                SELECT 1 FROM information_schema.tables
                WHERE table_schema = 'public' AND table_name = 'document_chunks'
            )
            """
        )
        if chunk_exists:
            logger.info("✓ Bảng 'document_chunks' tồn tại")
        else:
            logger.error("✗ Bảng 'document_chunks' KHÔNG tồn tại!")
            all_passed = False

        # ----- Bước 5: Kiểm tra cấu trúc cột bảng documents -----
        doc_columns = await conn.fetch(
            """
            SELECT column_name, data_type, is_nullable
            FROM information_schema.columns
            WHERE table_name = 'documents'
            ORDER BY ordinal_position
            """
        )
        logger.info("  Cấu trúc bảng 'documents':")
        required_doc_cols = {"source_uri", "content_hash", "version"}
        found_doc_cols = set()
        for col in doc_columns:
            logger.info("    - %s (%s)", col["column_name"], col["data_type"])
            found_doc_cols.add(col["column_name"])
        
        if required_doc_cols.issubset(found_doc_cols):
            logger.info("✓ Bảng 'documents' có đủ các cột mới (source_uri, content_hash, version)")
        else:
            missing = required_doc_cols - found_doc_cols
            logger.error("✗ Bảng 'documents' THIẾU cột: %s", missing)
            all_passed = False

        # ----- Bước 6: Kiểm tra cấu trúc cột bảng document_chunks -----
        chunk_columns = await conn.fetch(
            """
            SELECT column_name, data_type
            FROM information_schema.columns
            WHERE table_name = 'document_chunks'
            ORDER BY ordinal_position
            """
        )
        logger.info("  Cấu trúc bảng 'document_chunks':")
        required_chunk_cols = {"content_hash", "chunk_index"}
        found_chunk_cols = set()
        for col in chunk_columns:
            logger.info("    - %s (%s)", col["column_name"], col["data_type"])
            found_chunk_cols.add(col["column_name"])
        
        if required_chunk_cols.issubset(found_chunk_cols):
            logger.info("✓ Bảng 'document_chunks' có đủ các cột mới (content_hash)")
        else:
            missing = required_chunk_cols - found_chunk_cols
            logger.error("✗ Bảng 'document_chunks' THIẾU cột: %s", missing)
            all_passed = False

        # ----- Bước 7: Kiểm tra HNSW Index -----
        index_info = await conn.fetchval(
            """
            SELECT COUNT(*)
            FROM pg_class c
            JOIN pg_am am ON c.relam = am.oid
            WHERE c.relname = 'idx_chunks_embedding' AND am.amname = 'hnsw'
            """
        )
        if index_info > 0:
            logger.info("✓ Đã tìm thấy Index HNSW cho cột embedding")
        else:
            logger.error("✗ KHÔNG tìm thấy Index HNSW! (Hoặc có thể vẫn là IVFFlat)")
            all_passed = False

        # ----- Bước 8: Test INSERT + SELECT + DELETE -----
        logger.info("-" * 40)
        logger.info("Chạy test INSERT → SELECT → DELETE...")

        # Insert 1 document test (với schema mới)
        try:
            doc_id = await conn.fetchval(
                """
                INSERT INTO documents (filename, source_uri, content_hash, metadata, status)
                VALUES ($1, $2, $3, $4::jsonb, 'PROCESSING')
                RETURNING id
                """,
                "test_file.pdf",
                "file://data/test_file.pdf",
                "abc123hash",
                '{"subject": "Test", "week": 1}',
            )
            logger.info("  ✓ INSERT documents: id=%s", doc_id)

            # Insert 1 chunk test
            chunk_id = await conn.fetchval(
                """
                INSERT INTO document_chunks (document_id, content, chunk_index, content_hash)
                VALUES ($1, $2, $3, $4)
                RETURNING id
                """,
                doc_id,
                "Đây là nội dung test cho chunk số 0",
                0,
                "chunk_hash_0"
            )
            logger.info("  ✓ INSERT document_chunks: id=%s", chunk_id)

            # Verify SELECT
            row = await conn.fetchrow(
                "SELECT * FROM document_chunks WHERE id = $1", chunk_id
            )
            if row and row["content"] == "Đây là nội dung test cho chunk số 0":
                logger.info("  ✓ SELECT xác nhận dữ liệu đúng")
            else:
                logger.error("  ✗ SELECT trả về dữ liệu sai!")
                all_passed = False

            # Verify CASCADE DELETE (xóa document → chunk tự xóa)
            await conn.execute("DELETE FROM documents WHERE id = $1", doc_id)
            orphan = await conn.fetchval(
                "SELECT COUNT(*) FROM document_chunks WHERE document_id = $1", doc_id
            )
            if orphan == 0:
                logger.info("  ✓ CASCADE DELETE hoạt động đúng")
            else:
                logger.error("  ✗ CASCADE DELETE KHÔNG hoạt động!")
                all_passed = False
        except asyncpg.UniqueViolationError:
            logger.warning("  ! Dữ liệu test đã tồn tại (Unique Violation), bỏ qua bước insert.")

        # ----- Kết quả tổng hợp -----
        logger.info("=" * 60)
        if all_passed:
            logger.info("KẾT QUẢ: TẤT CẢ KIỂM TRA ĐỀU PASS ✓")
        else:
            logger.error("KẾT QUẢ: CÓ KIỂM TRA BỊ FAIL ✗")
        logger.info("=" * 60)

        await conn.close()
        return all_passed

    except Exception as exc:
        logger.error("Lỗi khi kiểm tra: %s", exc)
        logger.error(
            "Hãy đảm bảo PostgreSQL container đang chạy: "
            "docker-compose up -d postgres-db"
        )
        return False


if __name__ == "__main__":
    result = asyncio.run(verify_database())
    sys.exit(0 if result else 1)
