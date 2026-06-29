-- =============================================
-- documents: add columns required by Python RAG pipeline
-- =============================================
ALTER TABLE public.documents ADD COLUMN IF NOT EXISTS filename VARCHAR(255);
ALTER TABLE public.documents ADD COLUMN IF NOT EXISTS source_uri TEXT;
ALTER TABLE public.documents ADD COLUMN IF NOT EXISTS content_hash VARCHAR(64);
ALTER TABLE public.documents ADD COLUMN IF NOT EXISTS metadata JSONB DEFAULT '{}'::jsonb;
ALTER TABLE public.documents ADD COLUMN IF NOT EXISTS version INTEGER DEFAULT 1;

-- Required for ON CONFLICT (source_uri, content_hash) in vector_repo.py
ALTER TABLE public.documents
    DROP CONSTRAINT IF EXISTS documents_source_uri_content_hash_key;
ALTER TABLE public.documents
    ADD CONSTRAINT documents_source_uri_content_hash_key
    UNIQUE (source_uri, content_hash);

-- =============================================
-- document_chunks: align with Python insertion contract
-- =============================================
ALTER TABLE public.document_chunks
    ADD COLUMN IF NOT EXISTS chunk_index INTEGER DEFAULT 0;

-- Python pipeline does not supply source_url on insert
ALTER TABLE public.document_chunks ALTER COLUMN source_url DROP NOT NULL;
ALTER TABLE public.document_chunks ALTER COLUMN source_url SET DEFAULT '';

-- Change uniqueness from global content_hash to (document_id, content_hash)
ALTER TABLE public.document_chunks
    DROP CONSTRAINT IF EXISTS document_chunks_content_hash_key;
ALTER TABLE public.document_chunks
    DROP CONSTRAINT IF EXISTS document_chunks_document_id_content_hash_key;
ALTER TABLE public.document_chunks
    ADD CONSTRAINT document_chunks_document_id_content_hash_key
    UNIQUE (document_id, content_hash);
