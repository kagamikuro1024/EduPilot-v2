-- 0. Ensure System AI User exists (Idempotent Insertion for TD-02)
-- Must run before UPDATE to ensure referential integrity
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
INSERT INTO users (id, role, full_name, email) 
VALUES ('00000000-0000-0000-0000-000000000000', 'ADMIN', 'AI_TUTOR', 'ai.tutor@vibecode.com') 
ON CONFLICT (id) DO NOTHING;


-- 1. Update Foreign Key to CASCADE
-- First drop the existing constraint from V8
ALTER TABLE forum_posts DROP CONSTRAINT IF EXISTS fk_forum_posts_author;

-- Add it back with ON DELETE CASCADE to prevent orphaned posts
ALTER TABLE forum_posts ADD CONSTRAINT fk_forum_posts_author 
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE CASCADE;

-- 2. Data Cleanup: Ensure no NULL author_ids exist before applying NOT NULL constraint
-- Use the AI_TUTOR System UUID as fallback (Mentor Feedback)
UPDATE forum_posts SET author_id = '00000000-0000-0000-0000-000000000000' WHERE author_id IS NULL;

-- 3. Enforce NOT NULL on author_id
ALTER TABLE forum_posts ALTER COLUMN author_id SET NOT NULL;
