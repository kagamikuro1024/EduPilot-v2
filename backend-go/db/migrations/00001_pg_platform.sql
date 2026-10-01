-- Nền dữ liệu EduPilot (SRS FEAT-pg-foundation mục 5): extension -> enum -> hàm -> bảng -> index -> trigger.
-- Khoá chính: uuid DEFAULT uuidv7() (hàm native PostgreSQL 18). Mọi thời điểm là timestamptz.

-- +goose Up
CREATE EXTENSION IF NOT EXISTS vector;

CREATE TYPE user_role AS ENUM ('ADMIN', 'TEACHER', 'TA', 'STUDENT');
CREATE TYPE user_status AS ENUM ('PENDING_VERIFICATION', 'INVITED', 'ACTIVE', 'DISABLED');
CREATE TYPE job_status AS ENUM ('QUEUED', 'RUNNING', 'SUCCEEDED', 'FAILED');

-- +goose StatementBegin
CREATE FUNCTION set_updated_at() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    -- clock_timestamp(): trong CÙNG transaction, now() đứng yên nên updated_at không lớn hơn created_at.
    NEW.updated_at := clock_timestamp();
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE FUNCTION audit_log_block_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_log is append-only' USING ERRCODE = '42501';
END;
$$;
-- +goose StatementEnd

CREATE TABLE users (
    id                     uuid        NOT NULL DEFAULT uuidv7(),
    email                  text        NOT NULL,
    password_hash          text,
    full_name              text        NOT NULL,
    role                   user_role   NOT NULL,
    student_code           text,
    email_verified_at      timestamptz,
    failed_logins          integer     NOT NULL DEFAULT 0,
    locked_until           timestamptz,
    status                 user_status NOT NULL DEFAULT 'INVITED',
    ics_token              text,
    tracking_notice_ack_at timestamptz,
    last_login_at          timestamptz,
    version                integer     NOT NULL DEFAULT 1,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_pkey PRIMARY KEY (id),
    CONSTRAINT users_email_lower_chk CHECK (email = lower(email)),
    CONSTRAINT users_failed_logins_chk CHECK (failed_logins >= 0),
    CONSTRAINT users_active_password_chk CHECK (status <> 'ACTIVE' OR (password_hash IS NOT NULL AND password_hash <> '')),
    CONSTRAINT users_student_code_role_chk CHECK (student_code IS NULL OR role = 'STUDENT'),
    CONSTRAINT users_version_chk CHECK (version >= 1)
);

CREATE TABLE audit_log (
    id         uuid        NOT NULL DEFAULT uuidv7(),
    course_id  uuid,
    actor_id   uuid,
    entity     text        NOT NULL,
    entity_id  text        NOT NULL,
    action     text        NOT NULL,
    before     jsonb,
    after      jsonb,
    trace_id   text,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT audit_log_pkey PRIMARY KEY (id)
);

CREATE TABLE outbox (
    id              uuid        NOT NULL DEFAULT uuidv7(),
    topic           text        NOT NULL,
    payload         jsonb       NOT NULL DEFAULT '{}',
    created_at      timestamptz NOT NULL DEFAULT now(),
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    enqueued_at     timestamptz,
    dispatched_at   timestamptz,
    attempts        integer     NOT NULL DEFAULT 0,
    last_error      text,
    dead_at         timestamptz,
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT outbox_pkey PRIMARY KEY (id),
    CONSTRAINT outbox_topic_chk CHECK (topic ~ '^[a-z][a-z0-9_.]{0,63}$'),
    CONSTRAINT outbox_attempts_chk CHECK (attempts BETWEEN 0 AND 4),
    CONSTRAINT outbox_terminal_chk CHECK (NOT (dispatched_at IS NOT NULL AND dead_at IS NOT NULL))
);

CREATE TABLE jobs (
    id          uuid        NOT NULL DEFAULT uuidv7(),
    kind        text        NOT NULL,
    status      job_status  NOT NULL DEFAULT 'QUEUED',
    progress    smallint    NOT NULL DEFAULT 0,
    result      jsonb,
    error       jsonb,
    owner_id    uuid        NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    finished_at timestamptz,
    CONSTRAINT jobs_pkey PRIMARY KEY (id),
    CONSTRAINT jobs_progress_chk CHECK (progress BETWEEN 0 AND 100),
    CONSTRAINT jobs_finished_chk CHECK ((status IN ('SUCCEEDED', 'FAILED')) = (finished_at IS NOT NULL))
);

CREATE TABLE idempotency_keys (
    id           uuid        NOT NULL DEFAULT uuidv7(),
    user_id      uuid        NOT NULL,
    endpoint     text        NOT NULL,
    key          text        NOT NULL,
    request_hash text        NOT NULL,
    status_code  smallint    NOT NULL,
    response     jsonb       NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT idempotency_keys_pkey PRIMARY KEY (id),
    CONSTRAINT idempotency_keys_user_endpoint_key_key UNIQUE (user_id, endpoint, key)
);

CREATE UNIQUE INDEX users_email_key ON users (email);
CREATE UNIQUE INDEX users_ics_token_key ON users (ics_token) WHERE ics_token IS NOT NULL;
CREATE INDEX users_student_code_idx ON users (student_code) WHERE student_code IS NOT NULL;

CREATE INDEX audit_log_actor_created_idx ON audit_log (actor_id, created_at DESC, id DESC) WHERE actor_id IS NOT NULL;
CREATE INDEX audit_log_course_created_idx ON audit_log (course_id, created_at DESC, id DESC) WHERE course_id IS NOT NULL;
CREATE INDEX audit_log_entity_idx ON audit_log (entity, entity_id, created_at DESC);

CREATE INDEX outbox_pending_idx ON outbox (next_attempt_at, id) WHERE dispatched_at IS NULL AND dead_at IS NULL AND enqueued_at IS NULL;
CREATE INDEX outbox_stale_idx ON outbox (enqueued_at) WHERE dispatched_at IS NULL AND dead_at IS NULL AND enqueued_at IS NOT NULL;

CREATE INDEX jobs_owner_created_idx ON jobs (owner_id, created_at DESC, id DESC);
CREATE INDEX jobs_active_idx ON jobs (status, created_at) WHERE status IN ('QUEUED', 'RUNNING');

CREATE INDEX idempotency_keys_created_idx ON idempotency_keys (created_at);

CREATE TRIGGER users_set_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER outbox_set_updated_at BEFORE UPDATE ON outbox FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER jobs_set_updated_at BEFORE UPDATE ON jobs FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER audit_log_no_update BEFORE UPDATE OR DELETE ON audit_log FOR EACH ROW EXECUTE FUNCTION audit_log_block_mutation();
CREATE TRIGGER audit_log_no_truncate BEFORE TRUNCATE ON audit_log FOR EACH STATEMENT EXECUTE FUNCTION audit_log_block_mutation();

-- ===========================================================================
-- Quy ước vector (P1/P2 chép lại khối này, KHÔNG tạo bảng nhúng ở PG).
-- Lưu embedding ở cột vector(1536); đánh index HNSW trên biểu thức halfvec
-- (một nửa bộ nhớ, cùng thứ hạng cosine) và luôn lọc theo course_id (luật 13).
--
--   CREATE TABLE doc_chunks (
--       id         uuid        NOT NULL DEFAULT uuidv7(),
--       course_id  uuid        NOT NULL,
--       content    text        NOT NULL,
--       embedding  vector(1536) NOT NULL,
--       created_at timestamptz NOT NULL DEFAULT now(),
--       CONSTRAINT doc_chunks_pkey PRIMARY KEY (id)
--   );
--
--   CREATE INDEX doc_chunks_embedding_hnsw_idx ON doc_chunks
--       USING hnsw ((embedding::halfvec(1536)) halfvec_cosine_ops)
--       WITH (m = 16, ef_construction = 64);
--
--   -- Truy vấn: lọc course_id trong CÙNG câu, sắp theo khoảng cách cosine.
--   SELECT id, content
--     FROM doc_chunks
--    WHERE course_id = $2
--    ORDER BY embedding::halfvec(1536) <=> $1::halfvec(1536)
--    LIMIT 5;
--
-- Đọc: SET hnsw.ef_search = 40 (mặc định) — tăng khi cần recall cao hơn.
-- ===========================================================================

-- +goose Down
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS jobs;
DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS audit_log;
DROP TABLE IF EXISTS users;

DROP FUNCTION IF EXISTS audit_log_block_mutation();
DROP FUNCTION IF EXISTS set_updated_at();

DROP TYPE IF EXISTS job_status;
DROP TYPE IF EXISTS user_status;
DROP TYPE IF EXISTS user_role;
-- Extension `vector` được GIỮ LẠI có chủ ý (phần khác có thể đang dùng — US-PG-02 AC6).
