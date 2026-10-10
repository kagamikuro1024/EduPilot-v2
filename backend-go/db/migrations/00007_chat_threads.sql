-- Hội thoại riêng và Threads (SRS FEAT-private-chat-pii mục 5.1–5.4, 5.6): chat_sessions, chat_messages, forum_threads, forum_posts.
-- Tạo MỘT lần ở dạng cuối, không ALTER bảng cũ (luật 6). Mọi FK tới bảng thuộc lớp là FK PHỨC HỢP (course_id, …) để DB chặn trộn lớp (như 00006).
-- Cột `embedding` của forum_*: P3 chỉ tạo cột; HNSW ở P10 (00015). Nội dung chat chỉ nằm ở chat_messages (US-P3-01 AC9).

-- +goose Up
CREATE TYPE chat_channel AS ENUM ('PRIVATE', 'PUBLIC');
CREATE TYPE chat_role AS ENUM ('USER', 'ASSISTANT');
CREATE TYPE chat_stream_status AS ENUM ('STREAMING', 'DONE', 'FAILED', 'CANCELLED');
CREATE TYPE chat_feedback AS ENUM ('HELPFUL', 'NOT_HELPFUL');
CREATE TYPE thread_state AS ENUM ('OPEN', 'CLOSED');
CREATE TYPE thread_ai_state AS ENUM ('PENDING', 'ANSWERED', 'SKIPPED');
CREATE TYPE post_kind AS ENUM ('AI', 'HUMAN');
CREATE TYPE post_verification AS ENUM ('NONE', 'PENDING', 'VERIFIED', 'CORRECTED', 'REJECTED');

-- Thẻ của thread: ≤ 5 phần tử, mỗi phần tử 1–30 ký tự (CHECK không dùng được subquery nên đi qua hàm IMMUTABLE).
-- +goose StatementBegin
CREATE FUNCTION forum_tags_valid(tags text[]) RETURNS boolean LANGUAGE sql IMMUTABLE AS $$
    SELECT cardinality(tags) <= 5 AND NOT EXISTS (SELECT 1 FROM unnest(tags) t WHERE t IS NULL OR char_length(t) NOT BETWEEN 1 AND 30)
$$;
-- +goose StatementEnd

-- 5.2 chat_sessions -----------------------------------------------------------------------------------------------------------
CREATE TABLE chat_sessions (
    id              uuid         NOT NULL DEFAULT uuidv7(),
    course_id       uuid         NOT NULL,
    user_id         uuid         NOT NULL,
    channel         chat_channel NOT NULL DEFAULT 'PRIVATE',
    title           text,
    document_id     uuid,
    last_message_at timestamptz  NOT NULL DEFAULT now(),
    deleted_at      timestamptz,
    created_at      timestamptz  NOT NULL DEFAULT now(),
    updated_at      timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT chat_sessions_pkey PRIMARY KEY (id),
    CONSTRAINT chat_sessions_course_id_key UNIQUE (course_id, id),
    CONSTRAINT chat_sessions_id_user_key UNIQUE (id, user_id),
    CONSTRAINT chat_sessions_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT chat_sessions_user_fkey FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT chat_sessions_document_fkey FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE SET NULL,
    CONSTRAINT chat_sessions_title_chk CHECK (char_length(title) <= 120)
);
CREATE INDEX chat_sessions_user_idx ON chat_sessions (course_id, user_id, last_message_at DESC, id DESC) WHERE deleted_at IS NULL;

-- 5.3 chat_messages -----------------------------------------------------------------------------------------------------------
CREATE TABLE chat_messages (
    id              uuid               NOT NULL DEFAULT uuidv7(),
    course_id       uuid               NOT NULL,
    session_id      uuid               NOT NULL,
    user_id         uuid               NOT NULL,
    role            chat_role          NOT NULL,
    content         text               NOT NULL DEFAULT '',
    partial_content text,
    stream_status   chat_stream_status NOT NULL DEFAULT 'DONE',
    client_msg_id   uuid,
    reply_to        uuid,
    attempt         smallint           NOT NULL DEFAULT 1,
    intent          text,
    citations       jsonb              NOT NULL DEFAULT '[]',
    blocks          jsonb              NOT NULL DEFAULT '[]',
    confidence      numeric(4,3),
    low_confidence  boolean            NOT NULL DEFAULT false,
    no_context      boolean            NOT NULL DEFAULT false,
    degraded        boolean            NOT NULL DEFAULT false,
    masked_count    integer            NOT NULL DEFAULT 0,
    feedback        chat_feedback,
    error_code      text,
    trace_id        text,
    completed_at    timestamptz,
    created_at      timestamptz        NOT NULL DEFAULT now(),
    updated_at      timestamptz        NOT NULL DEFAULT now(),
    CONSTRAINT chat_messages_pkey PRIMARY KEY (id),
    CONSTRAINT chat_messages_course_id_key UNIQUE (course_id, id),
    CONSTRAINT chat_messages_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES chat_sessions (course_id, id) ON DELETE CASCADE,
    CONSTRAINT chat_messages_owner_fkey FOREIGN KEY (session_id, user_id) REFERENCES chat_sessions (id, user_id) ON DELETE CASCADE,
    CONSTRAINT chat_messages_reply_fkey FOREIGN KEY (course_id, reply_to) REFERENCES chat_messages (course_id, id),
    CONSTRAINT chat_messages_content_chk CHECK (char_length(content) <= 20000),
    CONSTRAINT chat_messages_partial_chk CHECK (stream_status <> 'DONE' OR partial_content IS NULL),
    CONSTRAINT chat_messages_user_chk CHECK (role <> 'USER' OR (stream_status = 'DONE' AND confidence IS NULL)),
    CONSTRAINT chat_messages_attempt_chk CHECK (attempt >= 1),
    CONSTRAINT chat_messages_intent_chk CHECK (intent ~ '^[A-Z_]{3,40}$'),
    CONSTRAINT chat_messages_citations_chk CHECK (jsonb_typeof(citations) = 'array'),
    CONSTRAINT chat_messages_blocks_chk CHECK (jsonb_typeof(blocks) = 'array'),
    CONSTRAINT chat_messages_confidence_chk CHECK (confidence BETWEEN 0 AND 1),
    CONSTRAINT chat_messages_masked_chk CHECK (masked_count >= 0),
    CONSTRAINT chat_messages_completed_chk CHECK (stream_status <> 'STREAMING' OR completed_at IS NULL)
);
CREATE UNIQUE INDEX chat_messages_idem_key ON chat_messages (session_id, client_msg_id) WHERE client_msg_id IS NOT NULL;
CREATE INDEX chat_messages_session_idx ON chat_messages (session_id, created_at DESC, id DESC);
CREATE INDEX chat_messages_streaming_idx ON chat_messages (updated_at) WHERE stream_status = 'STREAMING';

-- 5.4 forum_threads -----------------------------------------------------------------------------------------------------------
CREATE TABLE forum_threads (
    id               uuid            NOT NULL DEFAULT uuidv7(),
    course_id        uuid            NOT NULL,
    author_id        uuid            NOT NULL,
    title            text            NOT NULL,
    body             text            NOT NULL,
    tags             text[]          NOT NULL DEFAULT '{}',
    week_no          smallint,
    state            thread_state    NOT NULL DEFAULT 'OPEN',
    ai_state         thread_ai_state NOT NULL DEFAULT 'PENDING',
    ai_skip_reason   text,
    similar_of       uuid,
    pinned_at        timestamptz,
    reply_count      integer         NOT NULL DEFAULT 0,
    last_activity_at timestamptz     NOT NULL DEFAULT now(),
    embedding        vector(1536),
    deleted_at       timestamptz,
    version          integer         NOT NULL DEFAULT 1,
    created_at       timestamptz     NOT NULL DEFAULT now(),
    updated_at       timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT forum_threads_pkey PRIMARY KEY (id),
    CONSTRAINT forum_threads_course_id_key UNIQUE (course_id, id),
    CONSTRAINT forum_threads_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT forum_threads_author_fkey FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT forum_threads_similar_fkey FOREIGN KEY (course_id, similar_of) REFERENCES forum_threads (course_id, id),
    CONSTRAINT forum_threads_title_chk CHECK (char_length(title) BETWEEN 1 AND 200),
    CONSTRAINT forum_threads_body_chk CHECK (char_length(body) BETWEEN 1 AND 8000),
    CONSTRAINT forum_threads_tags_chk CHECK (forum_tags_valid(tags)),
    CONSTRAINT forum_threads_week_chk CHECK (week_no BETWEEN 1 AND 20),
    CONSTRAINT forum_threads_skip_chk CHECK ((ai_state = 'SKIPPED') = (ai_skip_reason IS NOT NULL) AND (ai_skip_reason IS NULL OR ai_skip_reason IN ('NO_CONTEXT', 'LOW_SCORE', 'LLM_UNAVAILABLE'))),
    CONSTRAINT forum_threads_reply_chk CHECK (reply_count >= 0),
    CONSTRAINT forum_threads_version_chk CHECK (version >= 1)
);
CREATE INDEX forum_threads_course_idx ON forum_threads (course_id, last_activity_at DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX forum_threads_week_idx ON forum_threads (course_id, week_no);
CREATE INDEX forum_threads_skipped_idx ON forum_threads (course_id, created_at) WHERE ai_state = 'SKIPPED' AND deleted_at IS NULL;

-- forum_posts -----------------------------------------------------------------------------------------------------------------
CREATE TABLE forum_posts (
    id                 uuid              NOT NULL DEFAULT uuidv7(),
    course_id          uuid              NOT NULL,
    thread_id          uuid              NOT NULL,
    author_id          uuid,
    kind               post_kind         NOT NULL,
    body               text              NOT NULL,
    verification_state post_verification NOT NULL DEFAULT 'NONE',
    citations          jsonb             NOT NULL DEFAULT '[]',
    confidence         numeric(4,3),
    ai_body            text,
    verified_by        uuid,
    verified_at        timestamptz,
    hidden_at          timestamptz,
    hidden_reason      text,
    hidden_by          uuid,
    deleted_at         timestamptz,
    embedding          vector(1536),
    version            integer           NOT NULL DEFAULT 1,
    created_at         timestamptz       NOT NULL DEFAULT now(),
    updated_at         timestamptz       NOT NULL DEFAULT now(),
    CONSTRAINT forum_posts_pkey PRIMARY KEY (id),
    CONSTRAINT forum_posts_course_id_key UNIQUE (course_id, id),
    CONSTRAINT forum_posts_thread_fkey FOREIGN KEY (course_id, thread_id) REFERENCES forum_threads (course_id, id) ON DELETE CASCADE,
    CONSTRAINT forum_posts_author_fkey FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT forum_posts_verified_by_fkey FOREIGN KEY (verified_by) REFERENCES users (id),
    CONSTRAINT forum_posts_hidden_by_fkey FOREIGN KEY (hidden_by) REFERENCES users (id),
    CONSTRAINT forum_posts_author_chk CHECK ((kind = 'AI') = (author_id IS NULL)),
    CONSTRAINT forum_posts_body_chk CHECK (char_length(body) BETWEEN 1 AND 8000),
    CONSTRAINT forum_posts_verification_chk CHECK ((kind = 'HUMAN') = (verification_state = 'NONE')),
    CONSTRAINT forum_posts_citations_chk CHECK (jsonb_typeof(citations) = 'array'),
    CONSTRAINT forum_posts_confidence_chk CHECK (confidence IS NULL OR (kind = 'AI' AND confidence BETWEEN 0 AND 1)),
    CONSTRAINT forum_posts_verified_chk CHECK ((verified_by IS NULL) = (verified_at IS NULL)),
    CONSTRAINT forum_posts_hidden_chk CHECK ((hidden_at IS NULL) = (hidden_reason IS NULL) AND char_length(hidden_reason) <= 200),
    CONSTRAINT forum_posts_version_chk CHECK (version >= 1)
);
CREATE UNIQUE INDEX forum_posts_one_ai_key ON forum_posts (thread_id) WHERE kind = 'AI';
CREATE INDEX forum_posts_thread_idx ON forum_posts (thread_id, created_at, id);
CREATE INDEX forum_posts_pending_idx ON forum_posts (course_id, created_at) WHERE kind = 'AI' AND verification_state = 'PENDING' AND hidden_at IS NULL AND deleted_at IS NULL;

CREATE TRIGGER chat_sessions_set_updated_at BEFORE UPDATE ON chat_sessions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER chat_messages_set_updated_at BEFORE UPDATE ON chat_messages FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER forum_threads_set_updated_at BEFORE UPDATE ON forum_threads FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER forum_posts_set_updated_at BEFORE UPDATE ON forum_posts FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS forum_posts;
DROP TABLE IF EXISTS forum_threads;
DROP TABLE IF EXISTS chat_messages;
DROP TABLE IF EXISTS chat_sessions;
DROP FUNCTION IF EXISTS forum_tags_valid(text[]);
DROP TYPE IF EXISTS post_verification;
DROP TYPE IF EXISTS post_kind;
DROP TYPE IF EXISTS thread_ai_state;
DROP TYPE IF EXISTS thread_state;
DROP TYPE IF EXISTS chat_feedback;
DROP TYPE IF EXISTS chat_stream_status;
DROP TYPE IF EXISTS chat_role;
DROP TYPE IF EXISTS chat_channel;
