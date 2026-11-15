-- Điểm danh tự phục vụ chống gian lận (SRS FEAT-attendance-selfcheck mục 5, sprint 10.8). Một giao dịch, chỉ CREATE; 00022 / 00023 giữ chỗ.
-- Không toạ độ, không IP, không token / chữ ký trong bảng nào. user_devices KHÔNG có user_id: một máy dùng cho nhiều tài khoản chính là tín hiệu.
-- attendance_records không đổi.

-- +goose Up
CREATE TYPE checkin_channel AS ENUM ('WEB', 'APP', 'CODE');
CREATE TYPE device_kind AS ENUM ('WEB_KEY', 'ANDROID_KEY');

CREATE TABLE checkin_windows (
    id             uuid        NOT NULL DEFAULT uuidv7(),
    course_id      uuid        NOT NULL,
    session_id     uuid        NOT NULL,
    round          smallint    NOT NULL,
    rotate_seconds smallint    NOT NULL,
    headcount      integer,
    opened_by      uuid        NOT NULL,
    opened_at      timestamptz NOT NULL DEFAULT now(),
    closes_at      timestamptz NOT NULL,
    closed_by      uuid,
    closed_at      timestamptz,
    CONSTRAINT checkin_windows_pkey PRIMARY KEY (id),
    CONSTRAINT checkin_windows_round_chk CHECK (round BETWEEN 1 AND 3),
    CONSTRAINT checkin_windows_rotate_chk CHECK (rotate_seconds BETWEEN 5 AND 60),
    CONSTRAINT checkin_windows_headcount_chk CHECK (headcount BETWEEN 0 AND 2000),
    CONSTRAINT checkin_windows_opened_by_fkey FOREIGN KEY (opened_by) REFERENCES users (id),
    CONSTRAINT checkin_windows_closed_by_fkey FOREIGN KEY (closed_by) REFERENCES users (id),
    CONSTRAINT checkin_windows_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES class_sessions (course_id, id),
    CONSTRAINT checkin_windows_round_key UNIQUE (course_id, session_id, round)
);
CREATE UNIQUE INDEX checkin_windows_one_open ON checkin_windows (course_id, session_id) WHERE closed_at IS NULL;
CREATE INDEX checkin_windows_sweep_idx ON checkin_windows (closes_at) WHERE closed_at IS NULL;
CREATE INDEX checkin_windows_purge_idx ON checkin_windows (closed_at) WHERE closed_at IS NOT NULL;

CREATE TABLE user_devices (
    id             uuid        NOT NULL DEFAULT uuidv7(),
    key_sha256     bytea       NOT NULL,
    public_key     bytea       NOT NULL,
    kind           device_kind NOT NULL,
    attested       boolean     NOT NULL DEFAULT false,
    boot_verified  boolean,
    security_level text,
    label          text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT user_devices_pkey PRIMARY KEY (id),
    CONSTRAINT user_devices_key_key UNIQUE (key_sha256),
    CONSTRAINT user_devices_pubkey_chk CHECK (octet_length(public_key) <= 512),
    CONSTRAINT user_devices_label_chk CHECK (label IS NULL) -- luôn NULL (US-AS-12 AC7): không gắn nhãn / tên máy
);

CREATE TABLE attendance_checkins (
    id               uuid             NOT NULL DEFAULT uuidv7(),
    course_id        uuid             NOT NULL,
    session_id       uuid             NOT NULL,
    window_id        uuid             NOT NULL,
    student_id       uuid             NOT NULL,
    device_id        uuid,
    channel          checkin_channel  NOT NULL,
    assurance        smallint         NOT NULL,
    token_age_ms     integer          NOT NULL,
    geo_distance_m   integer,
    geo_accuracy_m   integer,
    in_campus        boolean,
    flags            text[]           NOT NULL DEFAULT '{}',
    flags_cleared_by uuid,
    flags_cleared_at timestamptz,
    created_at       timestamptz      NOT NULL DEFAULT now(),
    CONSTRAINT attendance_checkins_pkey PRIMARY KEY (id),
    CONSTRAINT attendance_checkins_assurance_chk CHECK (assurance BETWEEN 1 AND 3),
    CONSTRAINT attendance_checkins_age_chk CHECK (token_age_ms >= 0),
    CONSTRAINT attendance_checkins_flags_chk CHECK (flags <@ ARRAY['DEVICE_SHARED', 'DEVICE_NEW', 'NO_DEVICE_KEY', 'DEVICE_UNTRUSTED', 'TOKEN_LATE', 'GEO_OUT', 'GEO_WEAK', 'GEO_DENIED', 'GEO_MOCK', 'OFF_CAMPUS', 'NOT_APP', 'RECHECK_MISS']::text[]),
    CONSTRAINT attendance_checkins_window_fkey FOREIGN KEY (window_id) REFERENCES checkin_windows (id),
    CONSTRAINT attendance_checkins_device_fkey FOREIGN KEY (device_id) REFERENCES user_devices (id),
    CONSTRAINT attendance_checkins_cleared_by_fkey FOREIGN KEY (flags_cleared_by) REFERENCES users (id),
    CONSTRAINT attendance_checkins_session_fkey FOREIGN KEY (course_id, session_id) REFERENCES class_sessions (course_id, id),
    CONSTRAINT attendance_checkins_student_fkey FOREIGN KEY (course_id, student_id) REFERENCES enrollments (course_id, user_id),
    CONSTRAINT attendance_checkins_natural_key UNIQUE (course_id, window_id, student_id)
);
CREATE INDEX attendance_checkins_session_idx ON attendance_checkins (course_id, session_id, created_at, id);
CREATE INDEX attendance_checkins_device_idx ON attendance_checkins (course_id, device_id) WHERE device_id IS NOT NULL;
CREATE INDEX attendance_checkins_student_idx ON attendance_checkins (course_id, student_id, created_at);

-- +goose Down
DROP TABLE IF EXISTS attendance_checkins;
DROP TABLE IF EXISTS user_devices;
DROP TABLE IF EXISTS checkin_windows;
DROP TYPE IF EXISTS device_kind;
DROP TYPE IF EXISTS checkin_channel;
