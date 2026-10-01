-- ChatMessage persists and orders messages by this column. Older databases were
-- created from V3 before the Java entity added created_at.
ALTER TABLE public.chat_messages
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

UPDATE public.chat_messages
SET created_at = CURRENT_TIMESTAMP
WHERE created_at IS NULL;

ALTER TABLE public.chat_messages
    ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP;
