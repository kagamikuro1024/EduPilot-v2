-- Sự kiện PII (SRS FEAT-private-chat-pii 5.5): chỉ đếm loại + số lượng, KHÔNG có cột văn bản, chỉ thêm (như audit_log).

-- +goose Up
CREATE TYPE pii_kind AS ENUM ('MSSV', 'EMAIL', 'PHONE', 'CCCD', 'NAME', 'PERSONAL_QUESTION', 'OTHER_PERSON');
CREATE TYPE pii_action AS ENUM ('BLOCKED', 'REDACTED', 'SWITCHED', 'MASKED');

CREATE TABLE pii_events (
    id         uuid         NOT NULL DEFAULT uuidv7(),
    course_id  uuid         NOT NULL,
    session_id uuid,
    user_id    uuid         NOT NULL,
    channel    chat_channel NOT NULL,
    pii_type   pii_kind     NOT NULL,
    count      integer      NOT NULL,
    action     pii_action   NOT NULL,
    created_at timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT pii_events_pkey PRIMARY KEY (id),
    CONSTRAINT pii_events_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT pii_events_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES chat_sessions (course_id, id),
    CONSTRAINT pii_events_user_fkey FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT pii_events_count_chk CHECK (count >= 1)
);
CREATE INDEX pii_events_course_idx ON pii_events (course_id, created_at DESC);
CREATE INDEX pii_events_user_idx ON pii_events (user_id, created_at DESC);

CREATE TRIGGER pii_events_no_update BEFORE UPDATE OR DELETE ON pii_events FOR EACH ROW EXECUTE FUNCTION audit_log_block_mutation();
CREATE TRIGGER pii_events_no_truncate BEFORE TRUNCATE ON pii_events FOR EACH STATEMENT EXECUTE FUNCTION audit_log_block_mutation();

-- +goose Down
DROP TABLE IF EXISTS pii_events;
DROP TYPE IF EXISTS pii_action;
DROP TYPE IF EXISTS pii_kind;
