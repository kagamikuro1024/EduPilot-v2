-- Forum analytics and chronological thread reads require this timestamp. The
-- original V4 table omitted it even though the Java entity persists it.
ALTER TABLE public.forum_posts
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

UPDATE public.forum_posts
SET created_at = CURRENT_TIMESTAMP
WHERE created_at IS NULL;

ALTER TABLE public.forum_posts
    ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX IF NOT EXISTS idx_forum_posts_created_at
    ON public.forum_posts (created_at);
