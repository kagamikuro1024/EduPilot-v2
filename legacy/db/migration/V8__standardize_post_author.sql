-- Migration: V8__standardize_post_author.sql
-- Goal: Eliminate heuristic author guessing (TD-04)

-- 1. Add author_id column
ALTER TABLE forum_posts ADD COLUMN author_id UUID;
ALTER TABLE forum_posts ADD CONSTRAINT fk_forum_posts_author 
    FOREIGN KEY (author_id) REFERENCES users(id) ON DELETE SET NULL;

-- 2. Create System User for AI Identification
-- Using fixed UUID '00000000-0000-0000-0000-000000000000'
INSERT INTO users (id, role, student_code, full_name, email, password_hash)
VALUES (
    '00000000-0000-0000-0000-000000000000', 
    'ADMIN', 
    'SYSTEM', 
    'AI_TUTOR', 
    'ai.tutor@vibecode.com', 
    'SYSTEM_INTERNAL_USER'
) ON CONFLICT (id) DO NOTHING;

-- 3. Data Migration: Populate author_id for existing posts (optional but good practice)
-- If student post, use thread author (existing heuristic in SQL form)
UPDATE forum_posts p
SET author_id = t.author_id
FROM forum_threads t
WHERE p.thread_id = t.id AND p.author_type = 'STUDENT' AND p.author_id IS NULL;

-- If AI post, use AI_TUTOR system id
UPDATE forum_posts 
SET author_id = '00000000-0000-0000-0000-000000000000' 
WHERE author_type = 'AI' AND author_id IS NULL;
