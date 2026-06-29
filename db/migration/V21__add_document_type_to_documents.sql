-- Classify knowledge documents: course materials vs school regulations (quy chế).
ALTER TABLE public.documents
    ADD COLUMN IF NOT EXISTS document_type VARCHAR(32) NOT NULL DEFAULT 'COURSE_MATERIAL';

ALTER TABLE public.documents
    DROP CONSTRAINT IF EXISTS documents_document_type_check;

ALTER TABLE public.documents
    ADD CONSTRAINT documents_document_type_check
    CHECK (document_type IN ('COURSE_MATERIAL', 'REGULATION'));

CREATE INDEX IF NOT EXISTS idx_documents_document_type ON public.documents (document_type);
