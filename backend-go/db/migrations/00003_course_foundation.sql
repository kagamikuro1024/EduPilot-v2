-- Nền lớp học (SRS FEAT-course-foundation mục 5): courses, enrollments, class_sessions, notifications, user_settings
-- và — vì P2 là phase đầu tiên dùng tới chúng cho việc chia sẻ tài liệu giữa các lớp — documents, content_chunks, document_courses.
-- Tạo MỘT lần ở dạng cuối, không ALTER về sau (luật 6). Nộp cùng 00004 (goose không chấp nhận khoảng trống số).

-- +goose Up
CREATE TYPE course_status AS ENUM ('ACTIVE', 'ARCHIVED');
CREATE TYPE enrollment_role AS ENUM ('TEACHER', 'TA', 'STUDENT');
CREATE TYPE enrollment_status AS ENUM ('PENDING', 'ACTIVE', 'REMOVED');
CREATE TYPE enrollment_joined_via AS ENUM ('ADMIN', 'ROSTER', 'CODE');
CREATE TYPE document_type AS ENUM ('LECTURE', 'COURSE_POLICY', 'EXAM_PAPER', 'ANSWER_KEY', 'OTHER');
CREATE TYPE document_status AS ENUM ('QUEUED', 'PROCESSING', 'READY', 'FAILED');
CREATE TYPE chunk_audience AS ENUM ('ALL', 'STAFF', 'GRADING');

CREATE TABLE courses (
    id                    uuid          NOT NULL DEFAULT uuidv7(),
    subject_code          text          NOT NULL,
    class_code            text          NOT NULL,
    name                  text          NOT NULL,
    semester              text          NOT NULL,
    status                course_status NOT NULL DEFAULT 'ACTIVE',
    escalation_threshold  numeric(3,2)  NOT NULL DEFAULT 0.60,
    settings              jsonb         NOT NULL DEFAULT '{}',
    join_code             char(7)       NOT NULL,
    join_enabled          boolean       NOT NULL DEFAULT true,
    join_expires_at       timestamptz,
    join_require_approval boolean       NOT NULL DEFAULT false,
    allowed_email_domain  text,
    capacity              integer,
    created_by            uuid          NOT NULL,
    archived_at           timestamptz,
    version               integer       NOT NULL DEFAULT 1,
    created_at            timestamptz   NOT NULL DEFAULT now(),
    updated_at            timestamptz   NOT NULL DEFAULT now(),
    CONSTRAINT courses_pkey PRIMARY KEY (id),
    CONSTRAINT courses_created_by_fkey FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT courses_subject_code_chk CHECK (subject_code ~ '^[A-Z0-9._-]{2,20}$'),
    CONSTRAINT courses_class_code_chk CHECK (class_code ~ '^[A-Za-z0-9._-]{3,20}$'),
    CONSTRAINT courses_name_chk CHECK (char_length(name) BETWEEN 1 AND 120),
    CONSTRAINT courses_semester_chk CHECK (semester ~ '^[0-9]{4}-[0-9]{4}-HK[123]$'),
    CONSTRAINT courses_escalation_chk CHECK (escalation_threshold BETWEEN 0 AND 1),
    CONSTRAINT courses_settings_chk CHECK (jsonb_typeof(settings) = 'object'),
    CONSTRAINT courses_join_code_chk CHECK (join_code ~ '^[ABCDEFGHJKMNPQRSTUVWXYZ23456789]{7}$'),
    CONSTRAINT courses_domain_chk CHECK (allowed_email_domain IS NULL OR allowed_email_domain ~ '^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)+$'),
    CONSTRAINT courses_capacity_chk CHECK (capacity IS NULL OR capacity BETWEEN 1 AND 1000),
    CONSTRAINT courses_version_chk CHECK (version >= 1),
    CONSTRAINT courses_archived_chk CHECK ((status = 'ARCHIVED') = (archived_at IS NOT NULL)),
    CONSTRAINT courses_archived_join_chk CHECK (status = 'ACTIVE' OR join_enabled = false)
);
CREATE UNIQUE INDEX courses_class_code_key ON courses (class_code);
CREATE UNIQUE INDEX courses_join_code_key ON courses (join_code);
CREATE INDEX courses_subject_idx ON courses (subject_code);
CREATE INDEX courses_status_created_idx ON courses (status, created_at DESC, id DESC);

CREATE TABLE enrollments (
    id                    uuid                  NOT NULL DEFAULT uuidv7(),
    course_id             uuid                  NOT NULL,
    user_id               uuid                  NOT NULL,
    role_in_course        enrollment_role       NOT NULL,
    status                enrollment_status     NOT NULL,
    joined_via            enrollment_joined_via NOT NULL,
    student_code_snapshot text,
    warning               text,
    previous_status       enrollment_status,
    status_changed_at     timestamptz           NOT NULL DEFAULT now(),
    status_changed_by     uuid,
    removed_at            timestamptz,
    version               integer               NOT NULL DEFAULT 1,
    created_at            timestamptz           NOT NULL DEFAULT now(),
    updated_at            timestamptz           NOT NULL DEFAULT now(),
    CONSTRAINT enrollments_pkey PRIMARY KEY (id),
    CONSTRAINT enrollments_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT enrollments_user_fkey FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT enrollments_code_chk CHECK (student_code_snapshot IS NULL OR student_code_snapshot ~ '^[A-Z0-9]{6,15}$'),
    CONSTRAINT enrollments_code_role_chk CHECK (role_in_course = 'STUDENT' OR student_code_snapshot IS NULL),
    CONSTRAINT enrollments_warning_chk CHECK (warning IN ('EMAIL_MISMATCH', 'EMAIL_UNVERIFIED')),
    CONSTRAINT enrollments_warning_role_chk CHECK (warning IS NULL OR role_in_course = 'STUDENT'),
    CONSTRAINT enrollments_removed_chk CHECK ((status = 'REMOVED') = (removed_at IS NOT NULL)),
    CONSTRAINT enrollments_version_chk CHECK (version >= 1)
);
CREATE UNIQUE INDEX enrollments_course_user_key ON enrollments (course_id, user_id);
CREATE UNIQUE INDEX enrollments_one_teacher_key ON enrollments (course_id) WHERE role_in_course = 'TEACHER' AND status = 'ACTIVE';
CREATE INDEX enrollments_course_status_idx ON enrollments (course_id, status, role_in_course, status_changed_at DESC, user_id);
CREATE INDEX enrollments_user_idx ON enrollments (user_id, status);
CREATE INDEX enrollments_snapshot_idx ON enrollments (course_id, student_code_snapshot) WHERE student_code_snapshot IS NOT NULL AND status IN ('ACTIVE', 'PENDING');
CREATE INDEX enrollments_pending_idx ON enrollments (course_id, created_at) WHERE status = 'PENDING';

CREATE TABLE class_sessions (
    id         uuid        NOT NULL DEFAULT uuidv7(),
    course_id  uuid        NOT NULL,
    session_no integer     NOT NULL,
    starts_at  timestamptz NOT NULL,
    ends_at    timestamptz NOT NULL,
    room       text,
    topic      text,
    version    integer     NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT class_sessions_pkey PRIMARY KEY (id),
    CONSTRAINT class_sessions_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT class_sessions_no_chk CHECK (session_no >= 1),
    CONSTRAINT class_sessions_time_chk CHECK (ends_at > starts_at),
    CONSTRAINT class_sessions_room_chk CHECK (char_length(room) <= 40),
    CONSTRAINT class_sessions_topic_chk CHECK (char_length(topic) <= 200),
    CONSTRAINT class_sessions_version_chk CHECK (version >= 1)
);
CREATE UNIQUE INDEX class_sessions_course_no_key ON class_sessions (course_id, session_no);
CREATE UNIQUE INDEX class_sessions_course_starts_key ON class_sessions (course_id, starts_at);
CREATE INDEX class_sessions_course_starts_idx ON class_sessions (course_id, starts_at);

CREATE TABLE notifications (
    id         uuid        NOT NULL DEFAULT uuidv7(),
    user_id    uuid        NOT NULL,
    course_id  uuid,
    type       text        NOT NULL,
    title      text        NOT NULL,
    body       text,
    link       text,
    dedupe_key text,
    read_at    timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT notifications_pkey PRIMARY KEY (id),
    CONSTRAINT notifications_user_fkey FOREIGN KEY (user_id) REFERENCES users (id),
    CONSTRAINT notifications_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT notifications_type_chk CHECK (type ~ '^[A-Z][A-Z0-9_]{0,39}$'),
    CONSTRAINT notifications_title_chk CHECK (char_length(title) BETWEEN 1 AND 200),
    CONSTRAINT notifications_body_chk CHECK (char_length(body) <= 1000),
    CONSTRAINT notifications_link_chk CHECK (link IS NULL OR link ~ '^/[^/\\]')
);
CREATE UNIQUE INDEX notifications_user_dedupe_key ON notifications (user_id, dedupe_key) WHERE dedupe_key IS NOT NULL;
CREATE INDEX notifications_user_created_idx ON notifications (user_id, created_at DESC, id DESC);
CREATE INDEX notifications_unread_idx ON notifications (user_id, created_at DESC, id DESC) WHERE read_at IS NULL;
CREATE INDEX notifications_course_created_idx ON notifications (course_id, created_at DESC) WHERE course_id IS NOT NULL;

CREATE TABLE user_settings (
    user_id                  uuid        NOT NULL,
    notify_ticket_by_mail    boolean     NOT NULL DEFAULT true,
    notify_answer_by_mail    boolean     NOT NULL DEFAULT true,
    remind_deadline_by_mail  boolean     NOT NULL DEFAULT true,
    preferences              jsonb       NOT NULL DEFAULT '{}',
    version                  integer     NOT NULL DEFAULT 1,
    created_at               timestamptz NOT NULL DEFAULT now(),
    updated_at               timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_settings_pkey PRIMARY KEY (user_id),
    CONSTRAINT user_settings_user_fkey FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT user_settings_preferences_chk CHECK (jsonb_typeof(preferences) = 'object'),
    CONSTRAINT user_settings_version_chk CHECK (version >= 1)
);

CREATE TABLE documents (
    id                  uuid            NOT NULL DEFAULT uuidv7(),
    course_id           uuid            NOT NULL,
    title               text            NOT NULL,
    type                document_type   NOT NULL DEFAULT 'LECTURE',
    filename            text,
    mime_type           text,
    size_bytes          bigint,
    sha256              char(64),
    blob_key            text,
    status              document_status NOT NULL DEFAULT 'QUEUED',
    error               text,
    page_count          integer,
    visible_to_students boolean         NOT NULL DEFAULT true,
    use_for_rag         boolean         NOT NULL DEFAULT true,
    category            text,
    week_no             smallint,
    download_count      integer         NOT NULL DEFAULT 0,
    uploaded_by         uuid,
    version             integer         NOT NULL DEFAULT 1,
    created_at          timestamptz     NOT NULL DEFAULT now(),
    updated_at          timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT documents_pkey PRIMARY KEY (id),
    CONSTRAINT documents_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT documents_uploaded_by_fkey FOREIGN KEY (uploaded_by) REFERENCES users (id),
    CONSTRAINT documents_title_chk CHECK (char_length(title) BETWEEN 1 AND 200),
    CONSTRAINT documents_filename_chk CHECK (char_length(filename) <= 255),
    CONSTRAINT documents_mime_chk CHECK (char_length(mime_type) <= 100),
    CONSTRAINT documents_size_chk CHECK (size_bytes IS NULL OR size_bytes >= 0),
    CONSTRAINT documents_sha256_chk CHECK (sha256 IS NULL OR sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT documents_error_chk CHECK (char_length(error) <= 1000),
    CONSTRAINT documents_page_count_chk CHECK (page_count IS NULL OR page_count >= 0),
    CONSTRAINT documents_category_chk CHECK (char_length(category) <= 40),
    CONSTRAINT documents_week_chk CHECK (week_no IS NULL OR week_no BETWEEN 1 AND 20),
    CONSTRAINT documents_download_chk CHECK (download_count >= 0),
    CONSTRAINT documents_version_chk CHECK (version >= 1),
    CONSTRAINT documents_answer_key_chk CHECK (type <> 'ANSWER_KEY' OR visible_to_students = false)
);
CREATE INDEX documents_course_created_idx ON documents (course_id, created_at DESC, id DESC);
CREATE INDEX documents_course_type_idx ON documents (course_id, type);
CREATE INDEX documents_course_week_idx ON documents (course_id, week_no) WHERE week_no IS NOT NULL;
CREATE INDEX documents_course_sha_idx ON documents (course_id, sha256) WHERE sha256 IS NOT NULL;

CREATE TABLE content_chunks (
    id          uuid           NOT NULL DEFAULT uuidv7(),
    document_id uuid           NOT NULL,
    course_ids  uuid[]         NOT NULL,
    audience    chunk_audience NOT NULL DEFAULT 'ALL',
    ord         integer        NOT NULL,
    page_no     integer,
    heading     text,
    text        text           NOT NULL,
    token_count integer,
    embedding   vector(1536),
    created_at  timestamptz    NOT NULL DEFAULT now(),
    updated_at  timestamptz    NOT NULL DEFAULT now(),
    CONSTRAINT content_chunks_pkey PRIMARY KEY (id),
    CONSTRAINT content_chunks_document_fkey FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT content_chunks_course_ids_chk CHECK (cardinality(course_ids) >= 1),
    CONSTRAINT content_chunks_ord_chk CHECK (ord >= 0),
    CONSTRAINT content_chunks_heading_chk CHECK (char_length(heading) <= 200)
);
CREATE UNIQUE INDEX content_chunks_doc_ord_key ON content_chunks (document_id, ord);
CREATE INDEX content_chunks_course_ids_gin ON content_chunks USING gin (course_ids);
CREATE INDEX content_chunks_doc_idx ON content_chunks (document_id);

CREATE TABLE document_courses (
    document_id uuid        NOT NULL,
    course_id   uuid        NOT NULL,
    shared_by   uuid,
    created_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT document_courses_pkey PRIMARY KEY (document_id, course_id),
    CONSTRAINT document_courses_document_fkey FOREIGN KEY (document_id) REFERENCES documents (id) ON DELETE CASCADE,
    CONSTRAINT document_courses_course_fkey FOREIGN KEY (course_id) REFERENCES courses (id),
    CONSTRAINT document_courses_shared_by_fkey FOREIGN KEY (shared_by) REFERENCES users (id)
);
CREATE INDEX document_courses_course_idx ON document_courses (course_id, document_id);

CREATE TRIGGER courses_set_updated_at BEFORE UPDATE ON courses FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER enrollments_set_updated_at BEFORE UPDATE ON enrollments FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER class_sessions_set_updated_at BEFORE UPDATE ON class_sessions FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER notifications_set_updated_at BEFORE UPDATE ON notifications FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER user_settings_set_updated_at BEFORE UPDATE ON user_settings FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER documents_set_updated_at BEFORE UPDATE ON documents FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER content_chunks_set_updated_at BEFORE UPDATE ON content_chunks FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- +goose Down
DROP TABLE IF EXISTS document_courses;
DROP TABLE IF EXISTS content_chunks;
DROP TABLE IF EXISTS documents;
DROP TABLE IF EXISTS user_settings;
DROP TABLE IF EXISTS notifications;
DROP TABLE IF EXISTS class_sessions;
DROP TABLE IF EXISTS enrollments;
DROP TABLE IF EXISTS courses;
DROP TYPE IF EXISTS chunk_audience;
DROP TYPE IF EXISTS document_status;
DROP TYPE IF EXISTS document_type;
DROP TYPE IF EXISTS enrollment_joined_via;
DROP TYPE IF EXISTS enrollment_status;
DROP TYPE IF EXISTS enrollment_role;
DROP TYPE IF EXISTS course_status;
