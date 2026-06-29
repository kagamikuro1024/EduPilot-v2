-- Fix for PSQLException: there is no unique or exclusion constraint matching the ON CONFLICT specification
-- This index supports the Java backend's "INSERT ... ON CONFLICT (content_hash)" logic.
-- It ensures that a specific piece of content is unique globally, which is required for 
-- Forum Corrections where document_id is NULL.

CREATE UNIQUE INDEX IF NOT EXISTS idx_document_chunks_content_hash_global 
ON public.document_chunks (content_hash);
