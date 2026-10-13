-- Quan sát: thời gian học và cấu hình đo (SRS FEAT-crm-360 mục 5.7–5.8, proposals #5, #20, #25).
-- CREATE: study_time_daily, insight_reports (+ chỉ mục, trigger). ALTER có chủ ý — đúng HAI câu (chủ dự án duyệt schema, proposals #5): user_settings.track_study_time,
-- courses.allow_study_time_opt_out. CREATE OR REPLACE VIEW v_student_360 (không phải ALTER TABLE) thay study_seconds_7d = 0 của 00013 bằng tổng 7 ngày gần nhất.

-- +goose Up
CREATE TABLE study_time_daily (
    id             uuid        NOT NULL DEFAULT uuidv7(),
    course_id      uuid        NOT NULL,
    student_id     uuid        NOT NULL,
    date           date        NOT NULL,
    area           text        NOT NULL,
    active_seconds integer     NOT NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT study_time_daily_pkey PRIMARY KEY (id),
    CONSTRAINT study_time_daily_key UNIQUE (course_id, student_id, date, area),
    CONSTRAINT study_time_daily_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT study_time_daily_area_chk CHECK (area IN ('CHAT', 'THREADS', 'PRACTICE', 'LIBRARY', 'CALENDAR')),
    CONSTRAINT study_time_daily_seconds_chk CHECK (active_seconds BETWEEN 0 AND 86400)
);
CREATE INDEX study_time_course_date_idx ON study_time_daily (course_id, date, student_id);
CREATE INDEX study_time_student_idx ON study_time_daily (course_id, student_id, date DESC);
CREATE TRIGGER study_time_daily_set_updated_at BEFORE UPDATE ON study_time_daily FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE insight_reports (
    id             uuid        NOT NULL DEFAULT uuidv7(),
    course_id      uuid        NOT NULL REFERENCES courses (id),
    kind           text        NOT NULL,
    period_from    date,
    period_to      date,
    payload        jsonb       NOT NULL,
    data_watermark timestamptz NOT NULL,
    generated_by   uuid        NOT NULL REFERENCES users (id),
    version        integer     NOT NULL DEFAULT 1,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT insight_reports_pkey PRIMARY KEY (id),
    CONSTRAINT insight_reports_kind_chk CHECK (kind IN ('KNOWLEDGE_GAP')),
    CONSTRAINT insight_reports_payload_chk CHECK (jsonb_typeof(payload) = 'object'),
    CONSTRAINT insight_reports_version_chk CHECK (version >= 1)
);
CREATE INDEX insight_reports_course_kind_idx ON insight_reports (course_id, kind, created_at DESC);
CREATE TRIGGER insight_reports_set_updated_at BEFORE UPDATE ON insight_reports FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Hai ALTER có chủ ý (đúng hai dòng bắt đầu bằng "ALTER TABLE" trong tệp này).
ALTER TABLE user_settings ADD COLUMN track_study_time boolean NOT NULL DEFAULT true;
ALTER TABLE courses ADD COLUMN allow_study_time_opt_out boolean NOT NULL DEFAULT true;

CREATE OR REPLACE VIEW v_student_360 AS
SELECT e.course_id, e.user_id AS student_id, u.full_name, e.student_code_snapshot AS student_code, e.created_at AS joined_at,
       coalesce(a.recorded, 0)  AS recorded,
       coalesce(a.present, 0)   AS present,
       coalesce(a.late, 0)      AS late,
       coalesce(a.excused, 0)   AS excused,
       coalesce(a.absent, 0)    AS absent,
       coalesce(p.speak_count, 0)::int            AS speak_count,
       coalesce(p.points_total, 0)::numeric(8,2)  AS points_total,
       coalesce(st.seconds, 0)::bigint            AS study_seconds_7d,
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
LEFT JOIN (SELECT course_id, student_id, sum(active_seconds) AS seconds
             FROM study_time_daily WHERE date > (now() AT TIME ZONE 'Asia/Ho_Chi_Minh')::date - 7 GROUP BY course_id, student_id) st ON st.course_id = e.course_id AND st.student_id = e.user_id
LEFT JOIN student_risk r ON r.course_id = e.course_id AND r.student_id = e.user_id
WHERE e.role_in_course = 'STUDENT' AND e.status = 'ACTIVE';

-- +goose Down
CREATE OR REPLACE VIEW v_student_360 AS
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
alter table courses drop column if exists allow_study_time_opt_out;
alter table user_settings drop column if exists track_study_time;
DROP TABLE IF EXISTS insight_reports;
DROP TABLE IF EXISTS study_time_daily;
