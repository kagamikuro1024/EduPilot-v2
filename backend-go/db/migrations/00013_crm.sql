-- Điểm danh, tham gia, ghi chú quan sát, rủi ro (SRS FEAT-crm-360 mục 5.1–5.6, proposals #5).
-- CREATE: attendance_records, participation_events, student_observations, student_risk (+ enum, chỉ mục, trigger), hai view.
-- ALTER có chủ ý — đúng MỘT câu, trên class_sessions (chủ dự án duyệt 2026-10-13, cùng tiền lệ 00010 / 00011 / 00012).
-- v_student_360.study_seconds_7d là 0 ở bản này: bảng study_time_daily thuộc 00014 (US-P5-03) và 00014 thay view bằng bản có join thật.

-- +goose Up
CREATE TYPE attendance_status AS ENUM ('PRESENT', 'LATE', 'EXCUSED', 'ABSENT');
CREATE TYPE participation_type AS ENUM ('SPEAK', 'BONUS', 'PENALTY');
CREATE TYPE observation_kind AS ENUM ('NOTE', 'AI_SUMMARY');
CREATE TYPE risk_level AS ENUM ('NONE', 'WATCH', 'AT_RISK');

-- ALTER duy nhất: khoá ngoại phức hợp (course_id, session_id) cần UNIQUE (course_id, id); "hoàn tất buổi" cần cờ có người + thời điểm.
ALTER TABLE class_sessions
    ADD COLUMN attendance_done_at timestamptz,
    ADD COLUMN attendance_done_by uuid REFERENCES users (id),
    ADD CONSTRAINT class_sessions_course_id_key UNIQUE (course_id, id),
    ADD CONSTRAINT class_sessions_done_chk CHECK ((attendance_done_at IS NULL) = (attendance_done_by IS NULL));

-- 5.2 attendance_records ------------------------------------------------------------------------------------------------------
CREATE TABLE attendance_records (
    id         uuid              NOT NULL DEFAULT uuidv7(),
    course_id  uuid              NOT NULL,
    session_id uuid              NOT NULL,
    student_id uuid              NOT NULL,
    status     attendance_status NOT NULL,
    note       text,
    marked_by  uuid              NOT NULL,
    version    integer           NOT NULL DEFAULT 1,
    created_at timestamptz       NOT NULL DEFAULT now(),
    updated_at timestamptz       NOT NULL DEFAULT now(),
    CONSTRAINT attendance_records_pkey PRIMARY KEY (id),
    CONSTRAINT attendance_records_session_student_key UNIQUE (course_id, session_id, student_id),
    CONSTRAINT attendance_records_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES class_sessions (course_id, id),
    CONSTRAINT attendance_records_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT attendance_records_marked_by_fkey FOREIGN KEY (marked_by) REFERENCES users (id),
    CONSTRAINT attendance_records_note_chk CHECK (char_length(note) <= 200),
    CONSTRAINT attendance_records_version_chk CHECK (version >= 1)
);
CREATE INDEX attendance_course_student_idx ON attendance_records (course_id, student_id, session_id);

-- 5.3 participation_events ----------------------------------------------------------------------------------------------------
CREATE TABLE participation_events (
    id              uuid               NOT NULL DEFAULT uuidv7(),
    course_id       uuid               NOT NULL,
    student_id      uuid               NOT NULL,
    session_id      uuid,
    type            participation_type NOT NULL,
    points          numeric(4,2)       NOT NULL,
    note            text,
    client_event_id uuid               NOT NULL,
    created_by      uuid               NOT NULL,
    created_at      timestamptz        NOT NULL DEFAULT now(),
    updated_at      timestamptz        NOT NULL DEFAULT now(),
    CONSTRAINT participation_events_pkey PRIMARY KEY (id),
    CONSTRAINT participation_events_client_event_key UNIQUE (client_event_id),
    CONSTRAINT participation_events_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT participation_events_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES class_sessions (course_id, id),
    CONSTRAINT participation_events_created_by_fkey FOREIGN KEY (created_by) REFERENCES users (id),
    CONSTRAINT participation_events_points_chk CHECK (points BETWEEN -5 AND 5),
    CONSTRAINT participation_events_speak_chk CHECK (type <> 'SPEAK' OR points >= 0),
    CONSTRAINT participation_events_bonus_chk CHECK (type <> 'BONUS' OR points > 0),
    CONSTRAINT participation_events_penalty_chk CHECK (type <> 'PENALTY' OR points < 0),
    CONSTRAINT participation_events_note_chk CHECK (char_length(note) <= 200)
);
CREATE INDEX participation_course_student_idx ON participation_events (course_id, student_id, created_at DESC, id DESC);
CREATE INDEX participation_course_session_idx ON participation_events (course_id, session_id) WHERE session_id IS NOT NULL;

-- 5.4 student_observations ----------------------------------------------------------------------------------------------------
CREATE TABLE student_observations (
    id         uuid             NOT NULL DEFAULT uuidv7(),
    course_id  uuid             NOT NULL,
    student_id uuid             NOT NULL,
    author_id  uuid             NOT NULL,
    kind       observation_kind NOT NULL DEFAULT 'NOTE',
    tags       text[]           NOT NULL DEFAULT '{}',
    body       text             NOT NULL,
    ai_edited  boolean          NOT NULL DEFAULT false,
    version    integer          NOT NULL DEFAULT 1,
    created_at timestamptz      NOT NULL DEFAULT now(),
    updated_at timestamptz      NOT NULL DEFAULT now(),
    CONSTRAINT student_observations_pkey PRIMARY KEY (id),
    CONSTRAINT student_observations_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT student_observations_author_fkey FOREIGN KEY (author_id) REFERENCES users (id),
    CONSTRAINT student_observations_tags_chk CHECK (forum_tags_valid(tags)),
    CONSTRAINT student_observations_body_chk CHECK (char_length(body) BETWEEN 1 AND 2000),
    CONSTRAINT student_observations_ai_edited_chk CHECK (kind <> 'NOTE' OR NOT ai_edited),
    CONSTRAINT student_observations_version_chk CHECK (version >= 1)
);
CREATE INDEX observations_student_idx ON student_observations (course_id, student_id, created_at DESC, id DESC);

-- 5.5 student_risk ------------------------------------------------------------------------------------------------------------
CREATE TABLE student_risk (
    course_id       uuid        NOT NULL,
    student_id      uuid        NOT NULL,
    level           risk_level  NOT NULL DEFAULT 'NONE',
    reasons         jsonb       NOT NULL DEFAULT '[]',
    weeks           jsonb       NOT NULL DEFAULT '[]',
    last_week_asks  integer     NOT NULL DEFAULT 0,
    last_week_posts integer     NOT NULL DEFAULT 0,
    entered_at      timestamptz,
    computed_at     timestamptz NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT student_risk_pkey PRIMARY KEY (course_id, student_id),
    CONSTRAINT student_risk_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT student_risk_reasons_chk CHECK (jsonb_typeof(reasons) = 'array'),
    CONSTRAINT student_risk_weeks_chk CHECK (jsonb_typeof(weeks) = 'array'),
    CONSTRAINT student_risk_counts_chk CHECK (last_week_asks >= 0 AND last_week_posts >= 0),
    CONSTRAINT student_risk_entered_chk CHECK ((level = 'AT_RISK') = (entered_at IS NOT NULL))
);
CREATE INDEX student_risk_level_idx ON student_risk (course_id, level, student_id);
CREATE INDEX student_risk_entered_idx ON student_risk (course_id, entered_at DESC) WHERE entered_at IS NOT NULL;

CREATE TRIGGER attendance_records_set_updated_at BEFORE UPDATE ON attendance_records FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER participation_events_set_updated_at BEFORE UPDATE ON participation_events FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER student_observations_set_updated_at BEFORE UPDATE ON student_observations FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER student_risk_set_updated_at BEFORE UPDATE ON student_risk FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 5.6 views -------------------------------------------------------------------------------------------------------------------
-- Mọi nhóm gộp có course_id ở khoá nhóm để vị từ course_id = $1 được đẩy xuống; không có chat_messages (đếm hoạt động nằm ở student_risk).
CREATE VIEW v_attendance_summary AS
SELECT course_id, student_id,
       count(*)::int                                          AS recorded,
       (count(*) FILTER (WHERE status = 'PRESENT'))::int      AS present,
       (count(*) FILTER (WHERE status = 'LATE'))::int         AS late,
       (count(*) FILTER (WHERE status = 'EXCUSED'))::int      AS excused,
       (count(*) FILTER (WHERE status = 'ABSENT'))::int       AS absent,
       max(updated_at)                                        AS last_marked_at
FROM attendance_records
GROUP BY course_id, student_id;

CREATE VIEW v_student_360 AS
SELECT e.course_id, e.user_id AS student_id, u.full_name, e.student_code_snapshot AS student_code, e.created_at AS joined_at,
       coalesce(a.recorded, 0)  AS recorded,
       coalesce(a.present, 0)   AS present,
       coalesce(a.late, 0)      AS late,
       coalesce(a.excused, 0)   AS excused,
       coalesce(a.absent, 0)    AS absent,
       coalesce(p.speak_count, 0)::int            AS speak_count,
       coalesce(p.points_total, 0)::numeric(8,2)  AS points_total,
       0::bigint                                  AS study_seconds_7d,
       coalesce(r.last_week_asks, 0)              AS last_week_asks,
       coalesce(r.last_week_posts, 0)             AS last_week_posts,
       coalesce(r.level, 'NONE'::risk_level)      AS risk_level,
       r.entered_at                               AS risk_entered_at,
       r.computed_at                              AS risk_computed_at
FROM enrollments e
JOIN users u ON u.id = e.user_id
LEFT JOIN v_attendance_summary a ON a.course_id = e.course_id AND a.student_id = e.user_id
LEFT JOIN (SELECT course_id, student_id, (count(*) FILTER (WHERE type = 'SPEAK'))::int AS speak_count, sum(points) AS points_total
             FROM participation_events GROUP BY course_id, student_id) p ON p.course_id = e.course_id AND p.student_id = e.user_id
LEFT JOIN student_risk r ON r.course_id = e.course_id AND r.student_id = e.user_id
WHERE e.role_in_course = 'STUDENT' AND e.status = 'ACTIVE';

-- +goose Down
DROP VIEW IF EXISTS v_student_360;
DROP VIEW IF EXISTS v_attendance_summary;
DROP TABLE IF EXISTS student_risk;
DROP TABLE IF EXISTS student_observations;
DROP TABLE IF EXISTS participation_events;
DROP TABLE IF EXISTS attendance_records;
alter table class_sessions DROP CONSTRAINT IF EXISTS class_sessions_done_chk, DROP CONSTRAINT IF EXISTS class_sessions_course_id_key,
    DROP COLUMN IF EXISTS attendance_done_by, DROP COLUMN IF EXISTS attendance_done_at;
DROP TYPE IF EXISTS risk_level;
DROP TYPE IF EXISTS observation_kind;
DROP TYPE IF EXISTS participation_type;
DROP TYPE IF EXISTS attendance_status;
