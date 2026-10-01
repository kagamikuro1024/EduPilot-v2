"""
Module: src.database.schema
Description: Database schema initialization script for the Vector Search system.
             When executed, this script:
             1. Enables the pgvector extension.
             2. Creates the `documents` table (metadata repository).
             3. Creates the `document_chunks` table (text segments + embeddings).
             4. Creates optimized indexes (HNSW) for vector similarity search.
"""

import asyncio
import logging
from src.database.connection import init_db_pool

logger = logging.getLogger(__name__)

# ====================================================================
# SQL SCHEMA INITIALIZATION SCRIPT
# ====================================================================
# 1. CREATE EXTENSION vector:
#    - Activates pgvector to support multi-dimensional float arrays (VECTOR type).
# 2. CREATE TYPE document_status:
#    - Enum restricting status values to: PROCESSING, READY, FAILED.
# 3. Table: documents:
#    - UUID primary key for distributed compatibility.
#    - JSONB metadata for flexible schema-less information tracking.
# 4. Table: document_chunks:
#    - VECTOR(1536) embedding for OpenAI text-embedding-3-small compatibility.
#    - ON DELETE CASCADE for automatic chunk purging.
# 5. Indexes:
#    - GIN index for JSONB metadata querying.
#    - HNSW index for high-performance Vector Cosine Similarity search.
# ====================================================================

INIT_SCHEMA_SQL = """
-- STEP 1: Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- STEP 2: Create Enum type for document processing status
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'document_status') THEN
        CREATE TYPE document_status AS ENUM ('PROCESSING', 'READY', 'FAILED');
    END IF;
END
$$;

-- STEP 3: Create 'documents' table (Registry)
CREATE TABLE IF NOT EXISTS documents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_uri    VARCHAR NOT NULL,
    filename      VARCHAR(500) NOT NULL,
    metadata      JSONB DEFAULT '{}'::jsonb,
    content_hash  VARCHAR NOT NULL,
    version       INTEGER DEFAULT 1,
    status        document_status DEFAULT 'PROCESSING',
    created_at    TIMESTAMPTZ DEFAULT now(),
    UNIQUE (source_uri, content_hash)
);

-- STEP 4: Create 'document_chunks' table (Text + Vectors)
CREATE TABLE IF NOT EXISTS document_chunks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id     UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    content         TEXT NOT NULL,
    embedding       VECTOR(1536),
    chunk_index     INTEGER NOT NULL DEFAULT 0,
    content_hash    VARCHAR NOT NULL,
    metadata        JSONB DEFAULT '{}'::jsonb,
    UNIQUE (document_id, content_hash)
);

-- STEP 5: Optimized indexing
-- Status index for processing pipeline monitoring
CREATE INDEX IF NOT EXISTS idx_documents_status
    ON documents (status);

-- GIN index for nested metadata querying
CREATE INDEX IF NOT EXISTS idx_documents_metadata
    ON documents USING GIN (metadata);

-- Foreign Key index for rapid JOINs and cascade purges
CREATE INDEX IF NOT EXISTS idx_chunks_document_id
    ON document_chunks (document_id);

-- HNSW Vector index for high-performance semantic search
CREATE INDEX IF NOT EXISTS idx_chunks_embedding
    ON document_chunks
    USING hnsw (embedding vector_cosine_ops)
    WITH (m = 16, ef_construction = 64);
"""

async def initialize_schema() -> None:
    """
    Executes the SQL script to initialize the database schema.
    """
    pool = await init_db_pool()
    try:
        async with pool.acquire() as conn:
            await conn.execute(INIT_SCHEMA_SQL)
            logger.info("Database schema initialized successfully.")
            logger.info(" - pgvector extension enabled")
            logger.info(" - 'documents' table ready")
            logger.info(" - 'document_chunks' table ready")
            logger.info(" - Optimized indexes created")
    except Exception as exc:
        logger.error("Schema initialization failed: %s", exc)
        raise

async def verify_schema() -> dict:
    """
    Verifies that the schema and expected tables exist.
    """
    pool = await init_db_pool()
    try:
        async with pool.acquire() as conn:
            # Check pgvector
            ext = await conn.fetchval(
                "SELECT COUNT(*) FROM pg_extension WHERE extname = 'vector'"
            )

            # Check core tables
            tables = await conn.fetch(
                """
                SELECT table_name
                FROM information_schema.tables
                WHERE table_schema = 'public'
                  AND table_name IN ('documents', 'document_chunks')
                ORDER BY table_name
                """
            )

            result = {
                "pgvector_installed": ext > 0,
                "tables": [row["table_name"] for row in tables],
            }

            for table_name in result["tables"]:
                columns = await conn.fetch(
                    """
                    SELECT column_name, data_type, is_nullable
                    FROM information_schema.columns
                    WHERE table_name = $1
                    ORDER BY ordinal_position
                    """,
                    table_name,
                )
                result[f"{table_name}_columns"] = [
                    {
                        "name": col["column_name"],
                        "type": col["data_type"],
                        "nullable": col["is_nullable"],
                    }
                    for col in columns
                ]

            return result
    except Exception as exc:
        logger.error("Schema verification failed: %s", exc)
        raise

if __name__ == "__main__":
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s [%(levelname)s] %(message)s",
    )
    asyncio.run(initialize_schema())
