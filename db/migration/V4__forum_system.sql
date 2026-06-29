CREATE TABLE forum_threads (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    author_id UUID,
    title VARCHAR(255),
    status thread_status,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE forum_posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    thread_id UUID,
    author_type post_author_type,
    content TEXT,
    original_ai_content TEXT,
    verification_status verification_status,
    verified_by_ta_id UUID
);

CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) UNIQUE
);

CREATE TABLE forum_thread_tags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    thread_id UUID,
    tag_id UUID,
    CONSTRAINT unique_thread_tag UNIQUE (thread_id, tag_id)
);