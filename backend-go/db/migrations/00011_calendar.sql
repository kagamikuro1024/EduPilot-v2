-- Lịch và nhắc (SRS FEAT-docs-calendar 5.1, 5.2, 5.7): calendar_events, reminder_log; users.ics_token chỉ lưu SHA-256 hex (CHECK).
-- Buổi học (class_sessions) và bài thi (exams) KHÔNG nhân bản vào đây: lịch gộp lúc đọc (UNION).

-- +goose Up
CREATE TYPE calendar_event_type AS ENUM ('EXAM', 'OTHER');
CREATE TYPE reminder_source AS ENUM ('CLASS_SESSION', 'WEEKLY_EXAM', 'CALENDAR_EVENT');

CREATE TABLE calendar_events (
    id          uuid                NOT NULL DEFAULT uuidv7(),
    course_id   uuid                NOT NULL,
    type        calendar_event_type NOT NULL,
    title       text                NOT NULL,
    starts_at   timestamptz         NOT NULL,
    ends_at     timestamptz,
    location    text,
    description text,
    ref_type    text,
    ref_id      uuid,
    created_by  uuid                NOT NULL,
    version     integer             NOT NULL DEFAULT 1,
    created_at  timestamptz         NOT NULL DEFAULT now(),
    updated_at  timestamptz         NOT NULL DEFAULT now(),
    CONSTRAINT calendar_events_pkey PRIMARY KEY (id),
    CONSTRAINT calendar_events_course_id_key UNIQUE (course_id, id),
    CONSTRAINT calendar_events_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT calendar_events_created_by_fkey FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT calendar_events_title_chk CHECK (char_length(title) BETWEEN 1 AND 120),
    CONSTRAINT calendar_events_time_chk CHECK (ends_at IS NULL OR ends_at > starts_at),
    CONSTRAINT calendar_events_location_chk CHECK (char_length(location) <= 80),
    CONSTRAINT calendar_events_description_chk CHECK (char_length(description) <= 1000),
    CONSTRAINT calendar_events_ref_chk CHECK ((ref_type IS NULL) = (ref_id IS NULL)),
    CONSTRAINT calendar_events_version_chk CHECK (version >= 1)
);
CREATE INDEX calendar_events_course_starts_idx ON calendar_events (course_id, starts_at, id);
CREATE TRIGGER calendar_events_set_updated_at BEFORE UPDATE ON calendar_events FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE reminder_log (
    id          uuid            NOT NULL DEFAULT uuidv7(),
    user_id     uuid            NOT NULL,
    course_id   uuid            NOT NULL,
    source_type reminder_source NOT NULL,
    source_id   uuid            NOT NULL,
    starts_at   timestamptz     NOT NULL,
    kind        text            NOT NULL DEFAULT 'T24H',
    created_at  timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT reminder_log_pkey PRIMARY KEY (id),
    CONSTRAINT reminder_log_user_fkey FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT reminder_log_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT reminder_log_kind_chk CHECK (kind = 'T24H'),
    CONSTRAINT reminder_log_unique UNIQUE (user_id, source_type, source_id, starts_at, kind)
);
CREATE INDEX reminder_log_starts_idx ON reminder_log (starts_at);

ALTER TABLE users ADD CONSTRAINT users_ics_token_hash_chk CHECK (ics_token IS NULL OR ics_token ~ '^[0-9a-f]{64}$');

-- +goose Down
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_ics_token_hash_chk;
DROP TABLE IF EXISTS reminder_log;
DROP TABLE IF EXISTS calendar_events;
DROP TYPE IF EXISTS reminder_source;
DROP TYPE IF EXISTS calendar_event_type;
