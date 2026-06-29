-- =============================================
-- Add citations JSONB column to forum_posts and chat_messages
-- Allows persisting RAG citation metadata so it survives page refresh.
-- Non-breaking: nullable column with no default.
-- =============================================
ALTER TABLE public.forum_posts ADD COLUMN IF NOT EXISTS citations JSONB;
ALTER TABLE public.chat_messages ADD COLUMN IF NOT EXISTS citations JSONB;
