-- Escalation, thông báo, kiểm duyệt Threads, tri thức của lớp (SRS FEAT-escalation-notify mục 5.1–5.4, proposals #2).
-- CREATE: escalation_tickets, post_reports, knowledge_items (+ enum, chỉ mục, trigger); hai CREATE INDEX trên bảng cũ
-- (documents_knowledge_key, mail_outbox_dead_idx). ALTER có chủ ý — đúng bốn câu (chủ dự án duyệt 2026-10-13, cùng tiền lệ 00010 / 00011).
-- Không mail_inbound (D6; P7 tạo khi cần).

-- +goose Up
CREATE TYPE ticket_status AS ENUM ('OPEN', 'CLAIMED', 'ANSWERED', 'CLOSED');
CREATE TYPE ticket_kind AS ENUM ('QUESTION', 'GRADE_APPEAL', 'TEACHER_OUTREACH');
CREATE TYPE ticket_origin AS ENUM ('AUTO_LOW_CONFIDENCE', 'AUTO_NO_FORMULA', 'STUDENT_REQUEST', 'STAFF_OUTREACH', 'STUDENT_APPEAL');
CREATE TYPE post_report_reason AS ENUM ('SPAM', 'OFF_TOPIC', 'PERSONAL_INFO', 'OFFENSIVE', 'WRONG_ANSWER', 'OTHER');
CREATE TYPE post_report_status AS ENUM ('OPEN', 'DISMISSED', 'ACTIONED');

-- 5.2 escalation_tickets ------------------------------------------------------------------------------------------------------
CREATE TABLE escalation_tickets (
    id                uuid          NOT NULL DEFAULT uuidv7(),
    course_id         uuid          NOT NULL,
    kind              ticket_kind   NOT NULL DEFAULT 'QUESTION',
    origin            ticket_origin NOT NULL,
    status            ticket_status NOT NULL DEFAULT 'OPEN',
    source_type       text          NOT NULL,
    source_id         uuid          NOT NULL,
    message_id        uuid,
    session_id        uuid,
    student_id        uuid          NOT NULL,
    confidence        numeric(4,3),
    appeal_payload    jsonb,
    claimed_by        uuid,
    claimed_at        timestamptz,
    answer            text,
    answered_by       uuid,
    answered_at       timestamptz,
    closed_at         timestamptz,
    closed_reason     text,
    dismiss_reason    text,
    reopened_count    smallint      NOT NULL DEFAULT 0,
    sla_stage         smallint      NOT NULL DEFAULT 0,
    sla_stage_at      timestamptz,
    waiting_since     timestamptz   NOT NULL DEFAULT now(),
    save_as_knowledge boolean       NOT NULL DEFAULT false,
    version           integer       NOT NULL DEFAULT 1,
    created_at        timestamptz   NOT NULL DEFAULT now(),
    updated_at        timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT escalation_tickets_pkey PRIMARY KEY (id),
    CONSTRAINT escalation_tickets_course_id_key UNIQUE (course_id, id),
    CONSTRAINT escalation_tickets_message_key UNIQUE (message_id),
    CONSTRAINT escalation_tickets_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT escalation_tickets_student_fkey FOREIGN KEY (student_id) REFERENCES users (id),
    CONSTRAINT escalation_tickets_claimed_by_fkey FOREIGN KEY (claimed_by) REFERENCES users (id),
    CONSTRAINT escalation_tickets_answered_by_fkey FOREIGN KEY (answered_by) REFERENCES users (id),
    CONSTRAINT escalation_tickets_message_fkey FOREIGN KEY (course_id, message_id) REFERENCES chat_messages (course_id, id),
    CONSTRAINT escalation_tickets_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES chat_sessions (course_id, id),
    CONSTRAINT escalation_tickets_source_type_chk CHECK (source_type ~ '^[A-Z_]{3,30}$'),
    CONSTRAINT escalation_tickets_confidence_chk CHECK (confidence BETWEEN 0 AND 1),
    CONSTRAINT escalation_tickets_open_chk CHECK (status <> 'OPEN' OR claimed_by IS NULL),
    CONSTRAINT escalation_tickets_claimed_chk CHECK (status <> 'CLAIMED' OR claimed_by IS NOT NULL),
    CONSTRAINT escalation_tickets_answered_chk CHECK (status <> 'ANSWERED' OR answered_by IS NOT NULL),
    CONSTRAINT escalation_tickets_closed_chk CHECK (status <> 'CLOSED' OR closed_at IS NOT NULL),
    CONSTRAINT escalation_tickets_question_chk CHECK (kind <> 'QUESTION' OR message_id IS NOT NULL),
    CONSTRAINT escalation_tickets_appeal_chk CHECK ((kind = 'GRADE_APPEAL') = (appeal_payload IS NOT NULL)),
    CONSTRAINT escalation_tickets_appeal_type_chk CHECK (appeal_payload IS NULL OR jsonb_typeof(appeal_payload) = 'object'),
    CONSTRAINT escalation_tickets_answer_chk CHECK (char_length(answer) <= 4000),
    CONSTRAINT escalation_tickets_closed_reason_chk CHECK (closed_reason IS NULL OR closed_reason IN ('STUDENT_CONFIRMED', 'AUTO_7D', 'STAFF_DISMISSED')),
    CONSTRAINT escalation_tickets_dismiss_reason_chk CHECK (dismiss_reason IS NULL OR dismiss_reason IN ('OUT_OF_SCOPE', 'DUPLICATE', 'NEED_MORE_INFO')),
    CONSTRAINT escalation_tickets_sla_chk CHECK (sla_stage BETWEEN 0 AND 2),
    CONSTRAINT escalation_tickets_version_chk CHECK (version >= 1)
);
CREATE INDEX escalation_tickets_course_status_idx ON escalation_tickets (course_id, status, created_at, id);
CREATE INDEX escalation_tickets_student_idx ON escalation_tickets (course_id, student_id, status, created_at DESC, id DESC);
CREATE INDEX escalation_tickets_sla_idx ON escalation_tickets (waiting_since, id) WHERE status IN ('OPEN', 'CLAIMED');
CREATE INDEX escalation_tickets_autoclose_idx ON escalation_tickets (answered_at) WHERE status = 'ANSWERED';
CREATE INDEX escalation_tickets_claimed_idx ON escalation_tickets (course_id, claimed_by) WHERE status = 'CLAIMED';
CREATE TRIGGER escalation_tickets_set_updated_at BEFORE UPDATE ON escalation_tickets FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 5.3 post_reports ------------------------------------------------------------------------------------------------------------
CREATE TABLE post_reports (
    id          uuid               NOT NULL DEFAULT uuidv7(),
    course_id   uuid               NOT NULL,
    thread_id   uuid,
    post_id     uuid,
    target_id   uuid GENERATED ALWAYS AS (COALESCE(thread_id, post_id)) STORED,
    reporter_id uuid               NOT NULL,
    reason      post_report_reason NOT NULL,
    note        text,
    status      post_report_status NOT NULL DEFAULT 'OPEN',
    handled_by  uuid,
    handled_at  timestamptz,
    created_at  timestamptz        NOT NULL DEFAULT now(),
    updated_at  timestamptz        NOT NULL DEFAULT now(),
    CONSTRAINT post_reports_pkey PRIMARY KEY (id),
    CONSTRAINT post_reports_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT post_reports_thread_fkey FOREIGN KEY (course_id, thread_id) REFERENCES forum_threads (course_id, id),
    CONSTRAINT post_reports_post_fkey FOREIGN KEY (course_id, post_id) REFERENCES forum_posts (course_id, id),
    CONSTRAINT post_reports_reporter_fkey FOREIGN KEY (reporter_id) REFERENCES users (id),
    CONSTRAINT post_reports_handled_by_fkey FOREIGN KEY (handled_by) REFERENCES users (id),
    CONSTRAINT post_reports_target_chk CHECK ((thread_id IS NULL) <> (post_id IS NULL)),
    CONSTRAINT post_reports_note_chk CHECK (char_length(note) <= 200 AND (note IS NULL OR reason = 'OTHER')),
    CONSTRAINT post_reports_handled_chk CHECK ((handled_by IS NULL) = (handled_at IS NULL)),
    CONSTRAINT post_reports_reporter_target_key UNIQUE (reporter_id, target_id)
);
CREATE INDEX post_reports_open_idx ON post_reports (course_id, status, created_at, id);
CREATE INDEX post_reports_target_idx ON post_reports (target_id) WHERE status = 'OPEN';
CREATE TRIGGER post_reports_set_updated_at BEFORE UPDATE ON post_reports FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 5.4 knowledge_items ---------------------------------------------------------------------------------------------------------
CREATE TABLE knowledge_items (
    id          uuid        NOT NULL DEFAULT uuidv7(),
    course_id   uuid        NOT NULL,
    source_type text        NOT NULL,
    source_id   uuid        NOT NULL,
    chunk_id    uuid        NOT NULL,
    created_by  uuid,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT knowledge_items_pkey PRIMARY KEY (id),
    CONSTRAINT knowledge_items_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT knowledge_items_chunk_fkey FOREIGN KEY (chunk_id) REFERENCES content_chunks (id) ON DELETE CASCADE,
    CONSTRAINT knowledge_items_created_by_fkey FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT knowledge_items_source_type_chk CHECK (source_type IN ('POST', 'TICKET')),
    CONSTRAINT knowledge_items_source_key UNIQUE (source_type, source_id)
);
CREATE INDEX knowledge_items_course_idx ON knowledge_items (course_id, created_at DESC);
CREATE TRIGGER knowledge_items_set_updated_at BEFORE UPDATE ON knowledge_items FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Hai CREATE INDEX trên bảng cũ (không phải ALTER TABLE) ------------------------------------------------------------------------
CREATE UNIQUE INDEX documents_knowledge_key ON documents (course_id) WHERE category = 'THREAD_KNOWLEDGE';
CREATE INDEX mail_outbox_dead_idx ON mail_outbox (updated_at DESC, id DESC) WHERE status = 'DEAD';

-- Bốn ALTER có chủ ý (SRS 5.1) ------------------------------------------------------------------------------------------------
ALTER TABLE forum_threads ADD COLUMN hidden_at timestamptz, ADD COLUMN hidden_reason text, ADD COLUMN hidden_by uuid REFERENCES users (id), ADD COLUMN edited_at timestamptz,
    ADD CONSTRAINT forum_threads_hidden_chk CHECK ((hidden_at IS NULL) = (hidden_reason IS NULL) AND char_length(hidden_reason) <= 200);
CREATE INDEX forum_threads_hidden_idx ON forum_threads (course_id, hidden_at) WHERE hidden_at IS NOT NULL;
ALTER TABLE forum_posts ADD COLUMN edited_at timestamptz;
ALTER TABLE user_settings ALTER COLUMN notify_ticket_by_mail SET DEFAULT false;
UPDATE user_settings SET notify_ticket_by_mail = false;
ALTER TABLE courses ALTER COLUMN escalation_threshold SET DEFAULT 0.80;
UPDATE courses SET escalation_threshold = 0.80 WHERE escalation_threshold = 0.60;

-- +goose Down
alter table courses ALTER COLUMN escalation_threshold SET DEFAULT 0.60;
alter table user_settings ALTER COLUMN notify_ticket_by_mail SET DEFAULT true;
alter table forum_posts DROP COLUMN IF EXISTS edited_at;
DROP INDEX IF EXISTS forum_threads_hidden_idx;
alter table forum_threads DROP CONSTRAINT IF EXISTS forum_threads_hidden_chk,
    DROP COLUMN IF EXISTS edited_at, DROP COLUMN IF EXISTS hidden_by, DROP COLUMN IF EXISTS hidden_reason, DROP COLUMN IF EXISTS hidden_at;
DROP INDEX IF EXISTS mail_outbox_dead_idx;
DROP INDEX IF EXISTS documents_knowledge_key;
DROP TABLE IF EXISTS knowledge_items;
DROP TABLE IF EXISTS post_reports;
DROP TABLE IF EXISTS escalation_tickets;
DROP TYPE IF EXISTS post_report_status;
DROP TYPE IF EXISTS post_report_reason;
DROP TYPE IF EXISTS ticket_origin;
DROP TYPE IF EXISTS ticket_kind;
DROP TYPE IF EXISTS ticket_status;
